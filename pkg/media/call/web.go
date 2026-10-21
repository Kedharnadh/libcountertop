package call

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	webName  = "API"
	webWait  = 25 * time.Second
	bodyMost = 256 << 10
)

type webSignal struct {
	Type   string `json:"type"`
	SDP    string `json:"sdp,omitempty"`
	Reason Reason `json:"reason,omitempty"`
}

type webLine struct {
	signals chan webSignal
	ended   func()
}

func (l webLine) send(_ context.Context, what string, m envelope) (int, error) {
	if what == "end" {
		defer l.ended()
	}
	select {
	case l.signals <- webSignal{Type: what, SDP: m.SDP, Reason: m.Reason}:
		return http.StatusOK, nil
	default:
		return http.StatusServiceUnavailable, nil
	}
}

type webCalls struct {
	c     *Calls
	peers func() []discovery.Peer
	mu    sync.Mutex
	lines map[string]webLine
}

type webPeer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
	Board string `json:"board"`
	Video bool   `json:"video"`
}

type webCall struct {
	ID              string    `json:"id"`
	Peer            webPeer   `json:"peer"`
	State           string    `json:"state"`
	Since           time.Time `json:"since"`
	Video           bool      `json:"video"`
	Muted           bool      `json:"muted"`
	CameraOff       bool      `json:"camera_off"`
	RemoteMuted     bool      `json:"remote_muted"`
	RemoteCameraOff bool      `json:"remote_camera_off"`
	Reconnecting    bool      `json:"reconnecting"`
	Browser         bool      `json:"browser"`
}

type webView struct {
	Call  *webCall  `json:"call"`
	Peers []webPeer `json:"peers"`
	Video bool      `json:"video"`
}

func (c *Calls) Handler(peers func() []discovery.Peer) http.Handler {
	h := &webCalls{c: c, peers: peers, lines: map[string]webLine{}}
	r := chi.NewRouter()
	r.Get("/", h.view)
	r.Post("/dial", h.dial)
	r.Post("/answer", func(w http.ResponseWriter, _ *http.Request) { c.Answer(); done(w) })
	r.Post("/hangup", func(w http.ResponseWriter, _ *http.Request) { c.Hangup(); done(w) })
	r.Post("/mute", h.flag(c.Mute))
	r.Post("/camera", h.flag(c.Camera))
	r.Post("/offer", h.offer)
	r.Get("/signal/{session}", h.signal)
	r.Post("/end/{session}", h.end)
	return r
}

func (h *webCalls) view(w http.ResponseWriter, _ *http.Request) {
	v := webView{Peers: []webPeer{}, Video: h.c.options.Video != nil}
	for _, p := range h.peers() {
		if p.Standing.Trusted() {
			v.Peers = append(v.Peers, peerOf(p))
		}
	}
	slices.SortFunc(v.Peers, func(x, y webPeer) int { return strings.Compare(x.Name, y.Name) })
	if now, ok := h.c.Now(); ok {
		v.Call = &webCall{
			ID:              now.ID,
			Peer:            peerOf(now.Peer),
			State:           now.State.String(),
			Since:           now.Since,
			Video:           now.Video,
			Muted:           now.Muted,
			CameraOff:       now.CameraOff,
			RemoteMuted:     now.RemoteMuted,
			RemoteCameraOff: now.RemoteCameraOff,
			Reconnecting:    now.Reconnecting,
			Browser:         now.Peer.ID == config.APICaller,
		}
	}
	reply(w, v)
}

func (h *webCalls) dial(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Peer  string `json:"peer"`
		Video bool   `json:"video"`
	}
	if !decode(w, r, &body) {
		return
	}
	p, ok := Find(h.peers(), body.Peer)
	if !ok {
		http.Error(w, "no such device", http.StatusNotFound)
		return
	}
	err := h.c.Start(p, body.Video && p.Video() && h.c.options.Video != nil)
	switch {
	case errors.Is(err, ErrBusy):
		http.Error(w, err.Error(), http.StatusConflict)
	case err != nil:
		http.Error(w, err.Error(), http.StatusBadGateway)
	default:
		done(w)
	}
}

func (h *webCalls) flag(set func(bool)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			On bool `json:"on"`
		}
		if !decode(w, r, &body) {
			return
		}
		set(body.On)
		done(w)
	}
}

func (h *webCalls) offer(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SDP    string `json:"sdp"`
		Video  bool   `json:"video"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if !decode(w, r, &body) {
		return
	}
	if body.SDP == "" {
		http.Error(w, "send an offer", http.StatusBadRequest)
		return
	}
	session := newID()
	line := webLine{signals: make(chan webSignal, 4), ended: func() { h.forget(session) }}
	h.mu.Lock()
	h.lines[session] = line
	h.mu.Unlock()
	status := h.c.incoming(
		envelope{
			Call:   session,
			SDP:    body.SDP,
			Video:  body.Video,
			Width:  body.Width,
			Height: body.Height,
		},
		discovery.Peer{ID: config.APICaller, Name: webName},
		line,
	)
	if status != http.StatusAccepted {
		h.forget(session)
	}
	switch status {
	case http.StatusAccepted:
		reply(w, map[string]string{"session": session, "status": "ringing"})
	case http.StatusConflict:
		reply(w, map[string]string{"status": "busy"})
	default:
		reply(w, map[string]string{"status": "refused"})
	}
}

func (h *webCalls) signal(w http.ResponseWriter, r *http.Request) {
	session := chi.URLParam(r, "session")
	h.mu.Lock()
	line, ok := h.lines[session]
	h.mu.Unlock()
	if !ok {
		reply(w, webSignal{Type: "end", Reason: ReasonHangup})
		return
	}
	wait, stop := context.WithTimeout(r.Context(), webWait)
	defer stop()
	select {
	case s := <-line.signals:
		reply(w, s)
	case <-wait.Done():
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *webCalls) end(w http.ResponseWriter, r *http.Request) {
	session := chi.URLParam(r, "session")
	h.forget(session)
	h.c.peerEnded(session, ReasonHangup)
	done(w)
}

func (h *webCalls) forget(session string) {
	h.mu.Lock()
	delete(h.lines, session)
	h.mu.Unlock()
}

func peerOf(p discovery.Peer) webPeer {
	return webPeer{ID: p.ID, Name: p.Name, Model: p.Model, Board: p.Board, Video: p.Video()}
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(v); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func done(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

func reply(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Debug("writing a call reply failed", "err", err)
	}
}
