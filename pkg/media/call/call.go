package call

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"

	"github.com/ygelfand/libcountertop/pkg/media/rtc"
	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	ringFor    = 45 * time.Second
	setupWait  = 10 * time.Second
	noticeFor  = 4 * time.Second
	clockEvery = time.Second
)

type State int

const (
	Idle State = iota
	Calling
	Ringing
	Talking
)

func (s State) String() string {
	return [...]string{"idle", "calling", "ringing", "talking"}[s]
}

type Reason string

const (
	ReasonHangup      Reason = "hangup"
	ReasonDeclined    Reason = "declined"
	ReasonCancelled   Reason = "cancelled"
	ReasonUnanswered  Reason = "unanswered"
	ReasonBusy        Reason = "busy"
	ReasonUnavailable Reason = "unavailable"
	ReasonDropped     Reason = "dropped"
	ReasonFailed      Reason = "failed"
	ReasonUnreachable Reason = "unreachable"
)

var (
	ErrBusy    = errors.New("already in a call")
	ErrForeign = errors.New("not in this home")
	ErrNoRoute = discovery.ErrNoRoute
)

type Call struct {
	ID              string
	Peer            discovery.Peer
	State           State
	Since           time.Time
	Muted           bool
	Video           bool
	CameraOff       bool
	RemoteMuted     bool
	RemoteCameraOff bool
	Reconnecting    bool
}

type session struct {
	workerMu sync.Mutex
	workers  sync.WaitGroup
	stopped  bool
	Call
	line    line
	offer   string
	link    *rtc.Link
	remote  Size
	own     Size
	pic     Picture
	ctx     context.Context
	stop    context.CancelFunc
	quiet   context.CancelFunc
	release func()
	unhook  func()
}

type Calls struct {
	options Dependencies
	ctx     context.Context
	stop    context.CancelFunc
	mu      sync.Mutex
	now     *session
	ha      entities
}

func New(o Dependencies) *Calls {
	ctx, stop := context.WithCancel(context.Background())
	c := &Calls{options: o, ctx: ctx, stop: stop}
	c.build()
	return c
}

func (c *Calls) Name() string { return "call" }

func (c *Calls) Now() (Call, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.now == nil {
		return Call{}, false
	}
	return c.now.Call, true
}

type Stats struct {
	Link   rtc.Stats
	Remote LayerStats
	Self   LayerStats
}

func (c *Calls) Stats() (Stats, bool) {
	c.mu.Lock()
	s := c.now
	var link *rtc.Link
	var pic Picture
	if s != nil {
		link, pic = s.link, s.pic
	}
	c.mu.Unlock()
	if link == nil {
		return Stats{}, false
	}
	st := Stats{Link: link.Stats()}
	if pic != nil {
		st.Remote, st.Self = pic.Stats()
	}
	return st, true
}

func (c *Calls) Sizes() (remote, own Size) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.now == nil {
		return Size{}, Size{}
	}
	return c.now.remote, c.now.own
}

func (c *Calls) Talking() bool {
	call, ok := c.Now()
	return ok && call.State == Talking
}

func (c *Calls) PausesWake() bool { return c.Talking() && c.options.Settings.Read().PauseWake }

func (c *Calls) Start(p discovery.Peer, video bool) error {
	if !p.Standing.Trusted() {
		c.inform(p.Name, say.T("peer.foreign"))
		return ErrForeign
	}
	addr, err := p.URL()
	if err != nil {
		c.notice(p.Name, ReasonUnreachable)
		return err
	}
	c.mu.Lock()
	if c.now != nil {
		c.mu.Unlock()
		c.inform(p.Name, say.T("actions.call.busy"))
		return ErrBusy
	}
	s := c.open(
		Call{ID: newID(), Peer: p, State: Calling, Since: time.Now(), Video: video},
		peerLine{c: c, base: addr},
	)
	c.mu.Unlock()
	slog.Info("calling", "to", p.Name, "call", s.ID, "video", video)
	c.changed(s)
	s.worker("call dial", func() { c.dial(s) })
	return nil
}

func (c *Calls) dial(s *session) {
	m, own := c.media(s)
	link, err := rtc.New(m, c.linked(s))
	if err != nil {
		c.fail(s, "link", err)
		return
	}
	c.mu.Lock()
	s.link = link
	c.mu.Unlock()
	c.apply(s)
	ctx, cancel := context.WithTimeout(s.ctx, setupWait)
	sdp, err := link.Offer(ctx)
	cancel()
	if err != nil {
		c.fail(s, "offer", err)
		return
	}
	me := c.options.Peers.Self()
	status, err := s.line.send(
		s.ctx,
		"offer",
		envelope{
			Call:   s.ID,
			From:   me.ID,
			Name:   me.Name,
			Model:  me.Model,
			Board:  me.Board,
			Caps:   me.Caps,
			SDP:    sdp,
			Video:  s.Video,
			Width:  own.Width,
			Height: own.Height,
		},
	)
	switch {
	case err != nil:
		c.fail(s, "offer", err)
		return
	case status == http.StatusConflict:
		c.requestClose(s, ReasonBusy)
		return
	case status == http.StatusForbidden:
		c.requestClose(s, ReasonUnavailable)
		return
	case status != http.StatusAccepted:
		c.fail(s, "offer", fmt.Errorf("status %d", status))
		return
	}
	ringing, quiet := context.WithCancel(s.ctx)
	c.mu.Lock()
	s.quiet = quiet
	c.mu.Unlock()
	back := c.tone(c.options.Settings.Read().Ringback, schema.ChimeRingback)
	s.worker("call ringback", func() { ringback(ringing, c.options.Sound, back) })
	select {
	case <-s.ctx.Done():
	case <-time.After(ringFor):
		if c.state(s) == Calling {
			c.requestEnd(s, ReasonUnanswered)
		}
	}
}

func (c *Calls) Answer() {
	c.mu.Lock()
	s := c.now
	if s == nil || s.State != Ringing {
		c.mu.Unlock()
		return
	}
	s.quiet()
	c.mu.Unlock()
	s.worker("call answer", func() { c.answer(s) })
}

func (c *Calls) answer(s *session) {
	m, own := c.media(s)
	link, err := rtc.New(m, c.linked(s))
	if err != nil {
		c.fail(s, "link", err)
		return
	}
	c.mu.Lock()
	s.link = link
	c.mu.Unlock()
	c.apply(s)
	ctx, cancel := context.WithTimeout(s.ctx, setupWait)
	sdp, err := link.Accept(ctx, s.offer)
	cancel()
	if err != nil {
		c.fail(s, "answer", err)
		return
	}
	status, err := s.line.send(
		s.ctx,
		"answer",
		envelope{Call: s.ID, SDP: sdp, Width: own.Width, Height: own.Height},
	)
	if err != nil || status != http.StatusOK {
		c.fail(s, "answer", fmt.Errorf("status %d: %v", status, err))
		return
	}
	c.talking(s)
}

func (c *Calls) AnswerAudio() {
	c.mu.Lock()
	if s := c.now; s != nil && s.State == Ringing {
		s.Video = false
	}
	c.mu.Unlock()
	c.Answer()
}

func (c *Calls) Hangup() {
	c.mu.Lock()
	s := c.now
	c.mu.Unlock()
	if s == nil {
		return
	}
	reason := ReasonHangup
	switch c.state(s) {
	case Ringing:
		reason = ReasonDeclined
	case Calling:
		reason = ReasonCancelled
	}
	c.end(s, reason)
}

func (c *Calls) Mute(on bool) {
	c.mu.Lock()
	s := c.now
	if s != nil {
		s.Muted = on
	}
	c.mu.Unlock()
	if s == nil {
		return
	}
	c.apply(s)
	c.changed(s)
}

func (c *Calls) Camera(on bool) {
	c.mu.Lock()
	s := c.now
	if s != nil {
		s.CameraOff = !on
	}
	c.mu.Unlock()
	if s == nil {
		return
	}
	c.apply(s)
	c.changed(s)
}

func (c *Calls) apply(s *session) {
	silenced := c.options.Privacy.MicMuted()
	c.mu.Lock()
	if c.now != s {
		c.mu.Unlock()
		return
	}
	link := s.link
	muted, blind := s.Muted || silenced, s.CameraOff
	c.mu.Unlock()
	if link != nil {
		link.Mute(muted)
		link.CameraOff(blind)
		c.tell(s)
	}
	c.show(s)
}

func (c *Calls) media(s *session) (rtc.Media, Size) {
	m := rtc.Media{
		Audio:    conversation{c.options.Sound},
		Opened:   func() { c.tell(s); c.tellSize(s) },
		Received: c.received(s),
	}
	c.mu.Lock()
	if c.options.Video == nil {
		s.Video = false
	}
	video, shown := s.Video, s.shown()
	c.mu.Unlock()
	if !video {
		return m, Size{}
	}
	pic, cam := c.options.Video.Start(Feed{
		Stream: c.options.Settings.Read().Stream,
		Shown:  shown,
		Key:    func() { c.requestKey(s) },
		Sized:  func(sz Size) { c.ownSized(s, sz) },
	})
	m.Screen, m.Camera = pic, cam
	c.mu.Lock()
	s.pic = pic
	own := s.own
	c.mu.Unlock()
	return m, own
}

func (s *session) shown() Shown {
	return Shown{
		Remote:   s.remote,
		Own:      s.own,
		RemoteOn: !s.RemoteCameraOff,
		SelfOn:   !s.CameraOff,
	}
}

func (c *Calls) show(s *session) {
	c.mu.Lock()
	pic, shown := s.pic, s.shown()
	c.mu.Unlock()
	if pic != nil {
		pic.Update(shown)
	}
}

func (c *Calls) requestKey(s *session) {
	c.mu.Lock()
	link := s.link
	c.mu.Unlock()
	if link != nil {
		link.RequestKey()
	}
}

func (c *Calls) redraw() {
	if c.options.Stage != nil {
		c.options.Stage.Redraw()
	}
}

func (c *Calls) open(call Call, l line) *session {
	ctx, stop := context.WithCancel(c.ctx)
	s := &session{Call: call, line: l, ctx: ctx, stop: stop, quiet: func() {}}
	s.unhook = c.options.Privacy.Watch(func() {
		c.apply(s)
		c.changed(s)
	})
	c.now = s
	return s
}

func (c *Calls) talking(s *session) {
	c.mu.Lock()
	if c.now != s {
		c.mu.Unlock()
		return
	}
	s.State, s.Since = Talking, time.Now()
	s.quiet()
	c.mu.Unlock()
	slog.Info("call connected", "peer", s.Peer.Name, "video", s.Video)
	c.options.Media.Pause()
	c.options.Sound.Claim("call", func(ctx context.Context) error {
		select {
		case <-ctx.Done():
		case <-s.ctx.Done():
		}
		return nil
	})
	c.changed(s)
	s.worker("call clock", func() {
		tick := time.NewTicker(clockEvery)
		defer tick.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-tick.C:
				c.options.Peers.Touch(s.Peer.ID)
				c.redraw()
			}
		}
	})
}

func (c *Calls) linked(s *session) func(webrtc.PeerConnectionState) {
	return func(state webrtc.PeerConnectionState) {
		if c.state(s) == Idle {
			return
		}
		switch state {
		case webrtc.PeerConnectionStateDisconnected, webrtc.PeerConnectionStateConnected:
			lost := state == webrtc.PeerConnectionStateDisconnected
			c.mu.Lock()
			was := s.Reconnecting
			s.Reconnecting = lost
			c.mu.Unlock()
			if was != lost {
				slog.Info("call link", "peer", s.Peer.Name, "state", state)
				c.changed(s)
			}
		default:
			slog.Info("call link lost", "peer", s.Peer.Name, "state", state)
			c.end(s, ReasonDropped)
		}
	}
}

func (c *Calls) state(s *session) State {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.now != s {
		return Idle
	}
	return s.State
}

func (c *Calls) fail(s *session, step string, err error) {
	slog.Warn("call failed", "step", step, "peer", s.Peer.Name, "err", err)
	c.requestEnd(s, ReasonFailed)
}

func (c *Calls) end(s *session, reason Reason) {
	l, id := s.line, s.ID
	if !c.finish(s, reason) {
		return
	}
	safe.Go("call end", func() {
		ctx, cancel := context.WithTimeout(c.ctx, postWait)
		defer cancel()
		if _, err := l.send(ctx, "end", envelope{Call: id, Reason: reason}); err != nil {
			slog.Debug("call end not acknowledged", "err", err)
		}
	})
}
func (c *Calls) close(s *session, reason Reason) { c.finish(s, reason) }
func (c *Calls) finish(s *session, reason Reason) bool {
	c.mu.Lock()
	if c.now != s {
		c.mu.Unlock()
		return false
	}
	c.now = nil
	link, release, pic, peer := s.link, s.release, s.pic, s.Peer.Name
	c.mu.Unlock()
	s.unhook()
	s.workerMu.Lock()
	s.stopped = true
	s.stop()
	s.workerMu.Unlock()
	if link != nil {
		_ = link.Close()
	}
	s.workers.Wait()
	if pic != nil {
		pic.Close()
	}
	if release != nil {
		release()
	}
	c.notice(peer, reason)
	c.publish(nil)
	c.redraw()
	slog.Info("call ended", "peer", peer, "reason", reason)
	return true
}

func (c *Calls) changed(s *session) {
	c.mu.Lock()
	if c.now != s {
		c.mu.Unlock()
		return
	}
	if s.release == nil && c.options.Stage != nil {
		c.mu.Unlock()
		release := c.options.Stage.Hold()
		c.mu.Lock()
		if c.now != s {
			c.mu.Unlock()
			release()
			return
		}
		s.release = release
	}
	c.mu.Unlock()
	c.show(s)
	c.publish(s)
	c.redraw()
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *session) worker(name string, run func()) {
	s.workerMu.Lock()
	if s.stopped {
		s.workerMu.Unlock()
		return
	}
	s.workers.Add(1)
	s.workerMu.Unlock()
	safe.Go(name, func() { defer s.workers.Done(); run() })
}
func (c *Calls) requestClose(s *session, r Reason) {
	safe.Go("call completion", func() { c.close(s, r) })
}

func (c *Calls) requestEnd(
	s *session,
	r Reason,
) {
	safe.Go("call completion", func() { c.end(s, r) })
}

func (c *Calls) Run(ctx context.Context) error { <-ctx.Done(); c.Close(); return nil }
func (c *Calls) Close() {
	c.stop()
	c.mu.Lock()
	s := c.now
	c.mu.Unlock()
	if s != nil {
		c.close(s, ReasonCancelled)
	}
}
