package session

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/camerafeed"
)

var ErrMuted = errors.New("livecam: the camera is off while the microphones are muted")
var ErrShutter = errors.New("livecam: the camera shutter is closed")
var ErrNoStill = errors.New("livecam: the helper could not make a picture")
var ErrRestarted = errors.New("livecam: the camera configuration changed")

type Block int

const (
	Open Block = iota
	Muted
	Shuttered
)

func (b Block) Err() error {
	switch b {
	case Muted:
		return ErrMuted
	case Shuttered:
		return ErrShutter
	}
	return nil
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Picture struct {
	RGBA          []byte
	Width, Height int
}

type Transport interface {
	Next(time.Duration) (camerafeed.Frame, error)
	AskStill() error
	AskKey() error
	Want(main, sub bool) error
	SetParams(string) error
	Close() error
}

type Snapshot struct {
	Config     camerafeed.Config
	Key        string
	StillExtra time.Duration
}
type Options struct {
	Snapshot                   func() Snapshot
	Open                       func(camerafeed.Config) (Transport, error)
	Blocked                    func() Block
	Recover                    func(error)
	RestartError               error
	Wait, Settle, Linger, Poll time.Duration
	Depth                      int
	Keep                       bool
	Card                       func(w, h int, b Block) []byte
}
type Hub struct {
	mu      sync.Mutex
	options Options
	cur     *Session
	card    *Session
	closed  bool
}
type Session struct {
	hub      *Hub
	stream   Transport
	snapshot Snapshot
	block    Block
	out      [2][]chan camerafeed.Frame
	config   [2][]byte
	stills   []chan camerafeed.Frame
	wants    [2]bool
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

func (h *Hub) Start() {
	if h.options.Keep {
		go h.keep()
	}
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
func (h *Hub) blocked() Block {
	if h.options.Blocked == nil {
		return Open
	}
	return h.options.Blocked()
}

func (h *Hub) acquire() (*Session, error) {
	if h.closed {
		return nil, errors.New("livecam: hub is closed")
	}
	if b := h.blocked(); b != Open {
		if h.cur != nil {
			h.retire(h.cur, b.Err())
		}
		return nil, b.Err()
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
		s := &Session{
			hub:      h,
			stream:   stream,
			snapshot: snapshot,
			done:     make(chan struct{}),
			opened:   time.Now(),
			wants:    [2]bool{true, true},
		}
		h.cur = s
		go h.read(s)
		go h.follow(s)
		h.tell(s)
	}
	h.cur.users++
	return h.cur, nil
}
func (h *Hub) release(s *Session) {
	if s.users <= 0 {
		return
	}
	s.users--
	if s.users > 0 || !h.current(s) {
		return
	}
	if b := h.blocked(); s.block == Open && b != Open {
		h.retire(s, b.Err())
		return
	}
	if s.block == Open && h.options.Keep {
		return
	}
	s.idle.Schedule(h.options.Linger, &h.mu, func() {
		if h.current(s) && s.users == 0 {
			h.retire(s, nil)
		}
	})
}

func (h *Hub) current(s *Session) bool { return s != nil && (h.cur == s || h.card == s) }

func (h *Hub) retire(s *Session, cause error) {
	s.idle.Cancel()
	if cause != nil && s.err == nil {
		s.err = cause
	}
	if h.cur == s {
		h.cur = nil
	}
	if h.card == s {
		h.card = nil
	}
	_ = s.stream.Close()
}
func (h *Hub) Join(at int) (*Session, <-chan camerafeed.Frame, error) {
	if at < 0 || at >= 2 {
		return nil, nil, fmt.Errorf("livecam: invalid stream %d", at)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	s, err := h.acquire()
	if err != nil {
		return nil, nil, err
	}
	return h.subscribe(s, at)
}

func (h *Hub) JoinOrCard(at int) (*Session, <-chan camerafeed.Frame, error) {
	if at < 0 || at >= 2 {
		return nil, nil, fmt.Errorf("livecam: invalid stream %d", at)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	b := h.blocked()
	if h.options.Card == nil || b == Open {
		s, err := h.acquire()
		if err != nil {
			return nil, nil, err
		}
		return h.subscribe(s, at)
	}
	s, err := h.acquireCard(b)
	if err != nil {
		return nil, nil, err
	}
	return h.subscribe(s, at)
}

func (h *Hub) acquireCard(b Block) (*Session, error) {
	if h.closed {
		return nil, errors.New("livecam: hub is closed")
	}
	if h.cur != nil {
		h.retire(h.cur, b.Err())
	}
	snapshot := h.options.Snapshot()
	if h.card != nil && (h.card.snapshot.Key != snapshot.Key || h.card.block != b) {
		h.retire(h.card, h.options.RestartError)
	}
	if h.card == nil {
		cfg := snapshot.Config
		cfg.Card[0] = h.options.Card(cfg.Width, cfg.Height, b)
		if cfg.SubWidth > 0 {
			cfg.Card[1] = h.options.Card(cfg.SubWidth, cfg.SubHeight, b)
		}
		stream, err := h.options.Open(cfg)
		if err != nil {
			return nil, err
		}
		s := &Session{
			hub:      h,
			stream:   stream,
			snapshot: snapshot,
			block:    b,
			done:     make(chan struct{}),
			opened:   time.Now(),
		}
		h.card = s
		go h.read(s)
		go h.follow(s)
	}
	h.card.idle.Cancel()
	h.card.users++
	return h.card, nil
}

func (h *Hub) subscribe(s *Session, at int) (*Session, <-chan camerafeed.Frame, error) {
	if at == 1 && s.snapshot.Config.SubWidth <= 0 {
		h.release(s)
		return nil, nil, fmt.Errorf("livecam: stream %d is unavailable", at)
	}
	ch := make(chan camerafeed.Frame, h.options.Depth)
	if s.config[at] != nil {
		ch <- camerafeed.Frame{Data: s.config[at], Config: true}
	}
	s.out[at] = append(s.out[at], ch)
	h.tell(s)
	return s, ch, nil
}

func (h *Hub) Leave(s *Session, frames <-chan camerafeed.Frame) {
	if s == nil || s.hub != h {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	found := false
	for at := range s.out {
		before := len(s.out[at])
		s.out[at] = slices.DeleteFunc(
			s.out[at],
			func(ch chan camerafeed.Frame) bool { return ch == frames },
		)
		found = found || len(s.out[at]) != before
	}
	if found {
		h.tell(s)
		h.release(s)
	}
}

func (h *Hub) tell(s *Session) {
	if !h.options.Keep || h.cur != s {
		return
	}
	want := [2]bool{len(s.out[0]) > 0, len(s.out[1]) > 0}
	if want == s.wants {
		return
	}
	s.wants = want
	if err := s.stream.Want(want[0], want[1]); err != nil {
		h.retire(s, err)
	}
}

func (h *Hub) keep() {
	ticker := time.NewTicker(h.options.Poll)
	defer ticker.Stop()
	for {
		h.mu.Lock()
		if h.closed {
			h.mu.Unlock()
			return
		}
		if h.cur == nil && h.blocked() == Open {
			s, err := h.acquire()
			if err != nil {
				slog.Debug("opening the camera failed", "err", err)
			} else {
				h.release(s)
			}
		}
		h.mu.Unlock()
		<-ticker.C
	}
}
func (h *Hub) RequestKey() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch {
	case h.cur != nil:
		return h.cur.stream.AskKey()
	case h.card != nil:
		return h.card.stream.AskKey()
	}
	return errors.New("livecam: no camera open")
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
	got := make(chan camerafeed.Frame, 1)
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
	defer func() {
		h.mu.Lock()
		s.stills = slices.DeleteFunc(
			s.stills,
			func(ch chan camerafeed.Frame) bool { return ch == got },
		)
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
	if h.card != nil {
		h.retire(h.card, nil)
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
			b := h.blocked()
			snapshot := h.options.Snapshot()
			h.mu.Lock()
			if !h.current(s) {
				h.mu.Unlock()
				return
			}
			if b != s.block {
				cause := b.Err()
				if s.block != Open {
					cause = h.options.RestartError
				}
				h.retire(s, cause)
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
	wait := h.options.Wait
	if h.options.Keep {
		wait = 0
	}
	for {
		frame, err := s.stream.Next(wait)
		if err != nil {
			if camerafeed.Timeout(err) {
				err = fmt.Errorf("livecam: no frame from the camera within %s", h.options.Wait)
			}
			h.mu.Lock()
			if s.err == nil {
				s.err = err
			}
			if h.current(s) {
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
