package configsync

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
)

const (
	postWait = 10 * time.Second
	bodyMost = 1 << 20
)

type envelope struct {
	ID    string   `json:"id"`
	From  string   `json:"from,omitempty"`
	Name  string   `json:"name,omitempty"`
	Model string   `json:"model,omitempty"`
	Board string   `json:"board,omitempty"`
	Caps  []string `json:"caps,omitempty"`
	Proof []byte   `json:"proof,omitempty"`
}

type sealed struct {
	Nonce  []byte `json:"nonce,omitempty"`
	Sealed []byte `json:"sealed,omitempty"`
	Left   int    `json:"left,omitempty"`
}

func (s *Sync) OfferHandler() http.Handler  { return http.HandlerFunc(s.offered) }
func (s *Sync) AskHandler() http.Handler    { return http.HandlerFunc(s.asked) }
func (s *Sync) PullHandler() http.Handler   { return http.HandlerFunc(s.pulled) }
func (s *Sync) CancelHandler() http.Handler { return http.HandlerFunc(s.cancelled) }

func (s *Sync) offered(w http.ResponseWriter, r *http.Request) {
	m, p, ok := s.caller(w, r)
	if !ok {
		return
	}
	addr, err := p.Addr()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := s.receiver(m.ID, p, addr); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	slog.Info("settings offered", "from", p.Name)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Sync) asked(w http.ResponseWriter, r *http.Request) {
	m, p, ok := s.caller(w, r)
	if !ok {
		return
	}
	addr, err := p.Addr()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := s.source(m.ID, p, addr); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	slog.Info("settings asked for", "by", p.Name)
	w.WriteHeader(http.StatusAccepted)
}

func (s *Sync) pulled(w http.ResponseWriter, r *http.Request) {
	var m envelope
	if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).
		Decode(&m); err != nil ||
		m.ID == "" {
		http.Error(w, "bad pull", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	ses := s.now
	if ses == nil || ses.ID != m.ID || ses.Role != Sending || ses.State != Waiting {
		s.mu.Unlock()
		http.Error(w, "no such exchange", http.StatusGone)
		return
	}
	if !ses.key.proves(ses.ID, m.Proof) {
		ses.Left--
		left := ses.Left
		s.mu.Unlock()
		if left <= 0 {
			s.fail(ses, ReasonWrong)
			http.Error(w, "too many tries", http.StatusGone)
			return
		}
		s.redraw()
		reply(w, http.StatusForbidden, sealed{Left: left})
		return
	}
	ses.State = Working
	k, id := ses.key, ses.ID
	s.mu.Unlock()

	plain, err := json.Marshal(s.o.Store.Export())
	if err != nil {
		s.fail(ses, ReasonFailed)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nonce, box, err := k.seal(id, plain)
	if err != nil {
		s.fail(ses, ReasonFailed)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	reply(w, http.StatusOK, sealed{Nonce: nonce, Sealed: box})
	slog.Info("settings sent", "to", ses.Peer.Name)
	s.mu.Lock()
	if s.now == ses {
		ses.State, ses.Reason = Done, ""
		s.settle(ses)
	}
	s.mu.Unlock()
	s.redraw()
}

func (s *Sync) cancelled(w http.ResponseWriter, r *http.Request) {
	var m envelope
	if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).
		Decode(&m); err != nil ||
		m.ID == "" {
		http.Error(w, "bad cancel", http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	ses := s.now
	s.mu.Unlock()
	if ses != nil && ses.ID == m.ID {
		s.fail(ses, ReasonCancelled)
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Sync) caller(w http.ResponseWriter, r *http.Request) (envelope, peer.Peer, bool) {
	var m envelope
	if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).
		Decode(&m); err != nil || m.ID == "" ||
		m.From == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return envelope{}, peer.Peer{}, false
	}
	if p, ok := s.o.Peers.Find(m.From); ok {
		return m, p, true
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "bad address", http.StatusBadRequest)
		return envelope{}, peer.Peer{}, false
	}
	return m, peer.Peer{
		ID:    m.From,
		Name:  m.Name,
		Model: m.Model,
		Board: m.Board,
		Caps:  m.Caps,
		Addrs: []net.IP{net.ParseIP(host)},
		Port:  s.o.Port,
		Seen:  time.Now(),
	}, true
}

func (s *Sync) post(addr, what string, m envelope, into *sealed) (int, error) {
	body, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), postWait)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"http://"+addr+"/config/"+what,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := fetch.Client(postWait).Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if into != nil && resp.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(io.LimitReader(resp.Body, bodyMost)).Decode(into); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func reply(w http.ResponseWriter, status int, v sealed) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("sync reply not written", "err", err)
	}
}
