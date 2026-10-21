package call

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"time"

	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const postWait = 5 * time.Second

type envelope struct {
	Call   string   `json:"call"`
	From   string   `json:"from,omitempty"`
	SDP    string   `json:"sdp,omitempty"`
	Video  bool     `json:"video,omitempty"`
	Width  int      `json:"width,omitempty"`
	Height int      `json:"height,omitempty"`
	Reason Reason   `json:"reason,omitempty"`
	Name   string   `json:"name,omitempty"`
	Model  string   `json:"model,omitempty"`
	Board  string   `json:"board,omitempty"`
	Caps   []string `json:"caps,omitempty"`
	User   string   `json:"-"`
}

func (c *Calls) offered(w http.ResponseWriter, r *http.Request) {
	var m envelope
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil || m.Call == "" || m.SDP == "" {
		http.Error(w, "bad offer", http.StatusBadRequest)
		return
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		http.Error(w, "bad address", http.StatusBadRequest)
		return
	}
	p, ok := c.options.Peers.Find(m.From)
	if !ok {
		p = discovery.Peer{
			ID:     m.From,
			Name:   m.Name,
			Model:  m.Model,
			Board:  m.Board,
			Caps:   m.Caps,
			Addrs:  []net.IP{net.ParseIP(host)},
			Port:   c.options.Port,
			Secure: c.options.Secure,
			Seen:   time.Now(),
		}
		if p.Name == "" {
			p.Name = host
		}
	}
	back := "https://" + net.JoinHostPort(host, strconv.Itoa(c.options.Secure))
	if r.TLS == nil {
		// TODO: remove the plain-HTTP peer path once no device predates the secure port.
		back = "http://" + net.JoinHostPort(host, strconv.Itoa(c.options.Port))
	}
	switch status := c.incoming(m, p, peerLine{c: c, base: back}); status {
	case http.StatusAccepted:
		w.WriteHeader(status)
	case http.StatusConflict:
		http.Error(w, "busy", status)
	default:
		http.Error(w, "not taking calls", status)
	}
}

func (c *Calls) incoming(m envelope, p discovery.Peer, l line) int {
	cfg := c.options.Settings.Read()
	if !cfg.Incoming {
		return http.StatusForbidden
	}
	auto := cfg.Answers(p.ID) || (m.User != "" && cfg.AnswersUser(m.User))
	c.mu.Lock()
	if c.now != nil {
		c.mu.Unlock()
		return http.StatusConflict
	}
	s := c.open(
		Call{
			ID:        m.Call,
			Peer:      p,
			State:     Ringing,
			Since:     time.Now(),
			Video:     m.Video,
			CameraOff: m.Video && auto && !cfg.AutoVideo,
		},
		l,
	)
	s.offer = m.SDP
	s.remote = Size{Width: m.Width, Height: m.Height}
	ringing, quiet := context.WithCancel(s.ctx)
	s.quiet = quiet
	c.mu.Unlock()
	slog.Info(
		"incoming call",
		"from",
		p.Name,
		"call",
		s.ID,
		"video",
		m.Video,
		"auto",
		auto,
	)
	c.changed(s)
	if auto {
		c.options.Sound.Tone(
			schema.StreamAlerts,
			ringLevel,
			c.tone(schema.ChimeDing, schema.ChimeDing),
		)
		if c.options.Messages != nil {
			c.options.Messages.Info(p.Name, say.T("call.connected"), noticeFor)
		}
		c.Answer()
		return http.StatusAccepted
	}
	chime, fallback := cfg.Ring, schema.ChimeRing
	if m.Video {
		chime = cfg.VideoRing
	}
	snd := c.tone(chime, fallback)
	s.worker("call ring", func() {
		if ring(ringing, c.options.Sound, snd) {
			slog.Info("call missed", "from", p.Name)
			c.requestClose(s, ReasonUnanswered)
		}
	})
	return http.StatusAccepted
}

func (c *Calls) answered(w http.ResponseWriter, r *http.Request) {
	var m envelope
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil || m.SDP == "" {
		http.Error(w, "bad answer", http.StatusBadRequest)
		return
	}
	c.mu.Lock()
	s := c.now
	ok := s != nil && s.ID == m.Call && s.State == Calling && s.link != nil
	if ok {
		s.remote = Size{Width: m.Width, Height: m.Height}
	}
	c.mu.Unlock()
	if !ok {
		http.Error(w, "no such call", http.StatusNotFound)
		return
	}
	c.show(s)
	if err := s.link.Answered(m.SDP); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		c.fail(s, "answered", err)
		return
	}
	c.talking(s)
}

func (c *Calls) ended(w http.ResponseWriter, r *http.Request) {
	var m envelope
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "bad end", http.StatusBadRequest)
		return
	}
	c.peerEnded(m.Call, m.Reason)
}

func (c *Calls) peerEnded(id string, reason Reason) {
	c.mu.Lock()
	s := c.now
	c.mu.Unlock()
	if s == nil || s.ID != id {
		return
	}
	given := reason
	switch reason {
	case ReasonCancelled, ReasonUnanswered:
		reason = ReasonHangup
	case "":
		reason = ReasonHangup
	}
	slog.Info("call ended by peer", "peer", s.Peer.Name, "reason", given)
	c.close(s, reason)
}

func (c *Calls) post(ctx context.Context, base, what string, m envelope) (int, error) {
	body, err := json.Marshal(m)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, postWait)
	defer cancel()
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		base+"/call/"+what,
		bytes.NewReader(body),
	)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.options.Client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

func (c *Calls) OfferHandler() http.Handler  { return http.HandlerFunc(c.offered) }
func (c *Calls) AnswerHandler() http.Handler { return http.HandlerFunc(c.answered) }
func (c *Calls) EndHandler() http.Handler    { return http.HandlerFunc(c.ended) }
