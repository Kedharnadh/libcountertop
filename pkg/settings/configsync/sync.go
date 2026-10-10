package configsync

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

const (
	codeFor      = 2 * time.Minute
	dismissAfter = 15 * time.Second
	tries        = 5
)

type Role int

const (
	Sending Role = iota + 1
	Receiving
)

type State int

const (
	Waiting State = iota + 1
	Working
	Done
	Failed
)

type Reason string

const (
	ReasonWrong     Reason = "wrong"
	ReasonExpired   Reason = "expired"
	ReasonBusy      Reason = "busy"
	ReasonCancelled Reason = "cancelled"
	ReasonFailed    Reason = "failed"
)

var ErrBusy = errors.New("already syncing")

type Exchange struct {
	ID      string
	Role    Role
	Peer    peer.Peer
	State   State
	Code    string
	Left    int
	Applied int
	Reason  Reason
	Expires time.Time
}

type Store interface {
	Export() setting.Snapshot
	Import(setting.Snapshot) (int, error)
}

type Peers interface {
	Self() peer.Peer
	Find(string) (peer.Peer, bool)
}

type Stage interface {
	Hold() (release func())
	Redraw()
}

type Dependencies struct {
	Store Store
	Peers Peers
	Port  int
	Stage Stage
}

type Sync struct {
	o   Dependencies
	mu  sync.Mutex
	now *session
}

type session struct {
	Exchange
	addr    string
	key     key
	release func()
	timer   *time.Timer
}

func New(o Dependencies) *Sync { return &Sync{o: o} }

func (s *Sync) Now() (Exchange, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.now == nil {
		return Exchange{}, false
	}
	return s.now.Exchange, true
}

func (s *Sync) Send(p peer.Peer) error {
	addr, err := p.Addr()
	if err != nil {
		return err
	}
	ses, err := s.source(newID(), p, addr)
	if err != nil {
		return err
	}
	safe.Go("sync offer", func() {
		status, err := s.post(ses.addr, "offer", s.envelope(ses.ID), nil)
		switch {
		case err != nil:
			s.fail(ses, ReasonFailed)
		case status == http.StatusConflict:
			s.fail(ses, ReasonBusy)
		case status != http.StatusAccepted:
			s.fail(ses, ReasonFailed)
		}
	})
	return nil
}

func (s *Sync) Fetch(p peer.Peer) error {
	addr, err := p.Addr()
	if err != nil {
		return err
	}
	ses, err := s.receiver(newID(), p, addr)
	if err != nil {
		return err
	}
	safe.Go("sync ask", func() {
		status, err := s.post(ses.addr, "ask", s.envelope(ses.ID), nil)
		switch {
		case err != nil:
			s.fail(ses, ReasonFailed)
		case status == http.StatusConflict:
			s.fail(ses, ReasonBusy)
		case status != http.StatusAccepted:
			s.fail(ses, ReasonFailed)
		}
	})
	return nil
}

func (s *Sync) Enter(code string) {
	s.mu.Lock()
	ses := s.now
	if ses == nil || ses.Role != Receiving || ses.State != Waiting {
		s.mu.Unlock()
		return
	}
	ses.State = Working
	s.mu.Unlock()
	s.redraw()
	safe.Go("sync pull", func() { s.pull(ses, code) })
}

func (s *Sync) Close() {
	s.mu.Lock()
	ses := s.now
	s.mu.Unlock()
	if ses == nil {
		return
	}
	if ses.State == Waiting || ses.State == Working {
		addr, id := ses.addr, ses.ID
		safe.Go("sync cancel", func() {
			if _, err := s.post(addr, "cancel", envelope{ID: id}, nil); err != nil {
				slog.Debug("sync cancel not acknowledged", "err", err)
			}
		})
	}
	s.finish(ses)
}

func (s *Sync) pull(ses *session, code string) {
	k := derive(code, ses.ID)
	var got sealed
	status, err := s.post(ses.addr, "pull", envelope{ID: ses.ID, Proof: k.proof(ses.ID)}, &got)
	switch {
	case err != nil:
		s.fail(ses, ReasonFailed)
		return
	case status == http.StatusForbidden:
		s.mu.Lock()
		if s.now == ses {
			ses.State, ses.Left, ses.Reason = Waiting, got.Left, ReasonWrong
		}
		s.mu.Unlock()
		s.redraw()
		return
	case status == http.StatusGone:
		s.fail(ses, ReasonExpired)
		return
	case status != http.StatusOK:
		s.fail(ses, ReasonFailed)
		return
	}
	plain, err := k.open(ses.ID, got.Nonce, got.Sealed)
	if err != nil {
		s.fail(ses, ReasonFailed)
		return
	}
	var snap setting.Snapshot
	if err := json.Unmarshal(plain, &snap); err != nil {
		s.fail(ses, ReasonFailed)
		return
	}
	applied, err := s.o.Store.Import(snap)
	if err != nil {
		slog.Warn("sync applied with errors", "from", ses.Peer.Name, "applied", applied, "err", err)
	}
	slog.Info("settings received", "from", ses.Peer.Name, "applied", applied)
	s.mu.Lock()
	if s.now == ses {
		ses.State, ses.Applied, ses.Reason = Done, applied, ""
		s.settle(ses)
	}
	s.mu.Unlock()
	s.redraw()
}

func (s *Sync) source(id string, p peer.Peer, addr string) (*session, error) {
	code, err := newCode()
	if err != nil {
		return nil, err
	}
	ses := &session{
		Exchange: Exchange{ID: id, Role: Sending, Peer: p, State: Waiting, Code: code, Left: tries},
		addr:     addr,
		key:      derive(code, id),
	}
	return ses, s.open(ses)
}

func (s *Sync) receiver(id string, p peer.Peer, addr string) (*session, error) {
	ses := &session{
		Exchange: Exchange{ID: id, Role: Receiving, Peer: p, State: Waiting, Left: tries},
		addr:     addr,
	}
	return ses, s.open(ses)
}

func (s *Sync) open(ses *session) error {
	s.mu.Lock()
	if s.now != nil && (s.now.State == Waiting || s.now.State == Working) {
		s.mu.Unlock()
		return ErrBusy
	}
	old := s.now
	ses.Expires = time.Now().Add(codeFor)
	ses.timer = time.AfterFunc(codeFor, func() { s.fail(ses, ReasonExpired) })
	s.now = ses
	s.mu.Unlock()
	if old != nil {
		s.release(old)
	}
	if s.o.Stage != nil {
		release := s.o.Stage.Hold()
		s.mu.Lock()
		if s.now == ses {
			ses.release = release
			release = nil
		}
		s.mu.Unlock()
		if release != nil {
			release()
		}
	}
	s.redraw()
	return nil
}

func (s *Sync) fail(ses *session, r Reason) {
	s.mu.Lock()
	if s.now != ses || ses.State == Done || ses.State == Failed {
		s.mu.Unlock()
		return
	}
	ses.State, ses.Reason = Failed, r
	s.settle(ses)
	s.mu.Unlock()
	slog.Info("sync ended", "peer", ses.Peer.Name, "reason", r)
	s.redraw()
}

func (s *Sync) settle(ses *session) {
	ses.timer.Stop()
	ses.timer = time.AfterFunc(dismissAfter, func() { s.finish(ses) })
}

func (s *Sync) finish(ses *session) {
	s.mu.Lock()
	if s.now != ses {
		s.mu.Unlock()
		return
	}
	s.now = nil
	ses.timer.Stop()
	s.mu.Unlock()
	s.release(ses)
	s.redraw()
}

func (s *Sync) release(ses *session) {
	s.mu.Lock()
	release := ses.release
	ses.release = nil
	s.mu.Unlock()
	if release != nil {
		release()
	}
}

func (s *Sync) redraw() {
	if s.o.Stage != nil {
		s.o.Stage.Redraw()
	}
}

func (s *Sync) envelope(id string) envelope {
	me := s.o.Peers.Self()
	return envelope{
		ID:    id,
		From:  me.ID,
		Name:  me.Name,
		Model: me.Model,
		Board: me.Board,
		Caps:  me.Caps,
	}
}

func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
