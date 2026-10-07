package session

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
)

var ErrMuted = errors.New("livecam: the camera is off while the microphones are muted")
var ErrNoStill = errors.New("livecam: the helper could not make a picture")
var ErrRestarted = errors.New("livecam: the camera configuration changed")

// Size is the encoded camera frame geometry.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Picture struct {
	RGBA          []byte
	Width, Height int
}

// Transport is the vendor camera helper's socket, not a direct camera-node owner.
type Transport interface {
	Next(time.Duration) (helper.Frame, error)
	AskStill() error
	AskKey() error
	SetParams(string) error
	Close() error
}

// Snapshot distinguishes session geometry/encoding changes from parameters applied in place.
type Snapshot struct {
	Config     helper.Config
	Key        string
	StillExtra time.Duration
}
type Options struct {
	Snapshot                   func() Snapshot
	Open                       func(helper.Config) (Transport, error)
	Muted                      func() bool
	Recover                    func(error)
	RestartError               error
	Wait, Settle, Linger, Poll time.Duration
	Depth                      int
}
type Hub struct {
	mu      sync.Mutex
	options Options
	cur     *Session
	closed  bool
}
type Session struct {
	hub      *Hub
	stream   Transport
	snapshot Snapshot
	out      [2][]chan helper.Frame
	config   [2][]byte
	stills   []chan helper.Frame
	users    int
	err      error
	done     chan struct{}
	opened   time.Time
	idle     Idle
}

func New(options Options) *Hub {
	if options.Wait <= 0 {
		options.Wait = 2 * time.Second
	}
	if options.Settle < 0 {
		options.Settle = 0
	}
	if options.Linger <= 0 {
		options.Linger = Linger
	}
	if options.Poll <= 0 {
		options.Poll = time.Second
	}
	if options.Depth <= 0 {
		options.Depth = 64
	}
	if options.RestartError == nil {
		options.RestartError = ErrRestarted
	}
	return &Hub{options: options}
}
func (s *Session) Done() <-chan struct{} { return s.done }
func (s *Session) Err() error {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()
	if s.err == nil {
		return errors.New("livecam: the camera helper went away")
	}
	return s.err
}
func (h *Hub) muted() bool { return h.options.Muted != nil && h.options.Muted() }
func (h *Hub) acquire() (*Session, error) {
	if h.closed {
		return nil, errors.New("livecam: hub is closed")
	}
	if h.muted() {
		if h.cur != nil {
			h.retire(h.cur, ErrMuted)
		}
		return nil, ErrMuted
	}
	snapshot := h.options.Snapshot()
	if h.cur != nil && h.cur.snapshot.Key != snapshot.Key {
		h.retire(h.cur, h.options.RestartError)
	}
	if h.cur != nil {
		h.cur.idle.Cancel()
	}
	if h.cur == nil {
		stream, err := h.options.Open(snapshot.Config)
		if err != nil {
			if h.options.Recover != nil {
				h.options.Recover(err)
			}
			return nil, err
		}
		s := &Session{hub: h, stream: stream, snapshot: snapshot, done: make(chan struct{}), opened: time.Now()}
		h.cur = s
		go h.read(s)
		go h.follow(s)
	}
	h.cur.users++
	return h.cur, nil
}
func (h *Hub) release(s *Session) {
	if s.users <= 0 {
		return
	}
	s.users--
	if s.users > 0 || h.cur != s {
		return
	}
	if h.muted() {
		h.retire(s, ErrMuted)
		return
	}
	s.idle.Schedule(h.options.Linger, &h.mu, func() {
		if h.cur == s && s.users == 0 {
			h.retire(s, nil)
		}
	})
}

// retire invalidates only this session. Existing consumers observe its read loop ending.
func (h *Hub) retire(s *Session, cause error) {
	s.idle.Cancel()
	if cause != nil && s.err == nil {
		s.err = cause
	}
	if h.cur == s {
		h.cur = nil
	}
	s.stream.Close()
}
func (h *Hub) Join(at int) (*Session, <-chan helper.Frame, error) {
	if at < 0 || at >= 2 {
		return nil, nil, fmt.Errorf("livecam: invalid stream %d", at)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s, err := h.acquire()
	if err != nil {
		return nil, nil, err
	}
	if at == 1 && s.snapshot.Config.SubWidth <= 0 {
		h.release(s)
		return nil, nil, fmt.Errorf("livecam: stream %d is unavailable", at)
	}
	ch := make(chan helper.Frame, h.options.Depth)
	if s.config[at] != nil {
		ch <- helper.Frame{Data: s.config[at], Config: true}
	}
	s.out[at] = append(s.out[at], ch)
	return s, ch, nil
}

// Leave is idempotent, including after the session was replaced.
func (h *Hub) Leave(s *Session, frames <-chan helper.Frame) {
	if s == nil || s.hub != h {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	found := false
	for at := range s.out {
		before := len(s.out[at])
		s.out[at] = slices.DeleteFunc(s.out[at], func(ch chan helper.Frame) bool { return ch == frames })
		found = found || len(s.out[at]) != before
	}
	if found {
		h.release(s)
	}
}
func (h *Hub) RequestKey() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cur == nil {
		return errors.New("livecam: no camera open")
	}
	return h.cur.stream.AskKey()
}
func (h *Hub) SetParams(params string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.cur == nil {
		return nil
	}
	return h.cur.stream.SetParams(params)
}
func (h *Hub) Still(within time.Duration) (Picture, error) {
	h.mu.Lock()
	s, err := h.acquire()
	if err != nil {
		h.mu.Unlock()
		return Picture{}, err
	}
	young := h.options.Settle - time.Since(s.opened)
	h.mu.Unlock()
	defer func() { h.mu.Lock(); h.release(s); h.mu.Unlock() }()
	if young > 0 {
		timer := time.NewTimer(young)
		defer timer.Stop()
		select {
		case <-timer.C:
		case <-s.done:
			return Picture{}, s.Err()
		}
	}
	got := make(chan helper.Frame, 1)
	h.mu.Lock()
	if s.err != nil || h.cur != s {
		h.mu.Unlock()
		return Picture{}, s.Err()
	}
	s.stills = append(s.stills, got)
	if len(s.stills) == 1 {
		err = s.stream.AskStill()
	}
	h.mu.Unlock()
	// Remove canceled/failed requests so a later response cannot block delivery or suppress a new ask.
	defer func() {
		h.mu.Lock()
		s.stills = slices.DeleteFunc(s.stills, func(ch chan helper.Frame) bool { return ch == got })
		h.mu.Unlock()
	}()
	if err != nil {
		return Picture{}, err
	}
	timer := time.NewTimer(max(within+s.snapshot.StillExtra, 0))
	defer timer.Stop()
	select {
	case frame := <-got:
		if len(frame.Data) == 0 {
			return Picture{}, ErrNoStill
		}
		return Picture{RGBA: frame.Data, Width: frame.Width, Height: frame.Height}, nil
	case <-s.done:
		return Picture{}, s.Err()
	case <-timer.C:
		return Picture{}, fmt.Errorf("livecam: no picture within %s", within)
	}
}
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
	if h.cur != nil {
		h.retire(h.cur, nil)
	}
}
func (h *Hub) follow(s *Session) {
	ticker := time.NewTicker(h.options.Poll)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			muted := h.muted()
			snapshot := h.options.Snapshot()
			h.mu.Lock()
			if h.cur != s {
				h.mu.Unlock()
				return
			}
			if muted {
				h.retire(s, ErrMuted)
				h.mu.Unlock()
				return
			}
			if snapshot.Key != s.snapshot.Key {
				h.retire(s, h.options.RestartError)
				h.mu.Unlock()
				return
			}
			h.mu.Unlock()
		}
	}
}
func (h *Hub) read(s *Session) {
	defer close(s.done)
	for {
		frame, err := s.stream.Next(h.options.Wait)
		if err != nil {
			if helper.Timeout(err) {
				err = fmt.Errorf("livecam: no frame from the camera within %s", h.options.Wait)
			}
			h.mu.Lock()
			if s.err == nil {
				s.err = err
			}
			if h.cur == s {
				h.retire(s, err)
			}
			h.mu.Unlock()
			return
		}
		h.mu.Lock()
		if frame.Still {
			for _, ch := range s.stills {
				select {
				case ch <- frame:
				default:
				}
			}
			s.stills = nil
			h.mu.Unlock()
			continue
		}
		at := 0
		if frame.Sub {
			at = 1
		}
		if frame.Config {
			s.config[at] = slices.Clone(frame.Data)
		}
		for _, ch := range s.out[at] {
			select {
			case ch <- frame:
			default:
			}
		}
		h.mu.Unlock()
	}
}

// Sizes is the encoded geometry of each enabled stream in the current snapshot.
func (h *Hub) Sizes() []Size {
	c := h.options.Snapshot().Config
	if c.Width <= 0 || c.Height <= 0 {
		return nil
	}
	out := []Size{{Width: c.Width, Height: c.Height}}
	if c.SubWidth > 0 && c.SubHeight > 0 {
		out = append(out, Size{Width: c.SubWidth, Height: c.SubHeight})
	}
	return out
}
