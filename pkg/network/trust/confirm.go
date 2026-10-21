package trust

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"

	"filippo.io/cpace"

	"github.com/ygelfand/libcountertop/pkg/network/peer"
)

const (
	ConfirmPath = "/peer/confirm"
	ProvePath   = "/peer/prove"
	bodyMost    = 4 << 10
)

var binding = []byte("countertop home key 1")

type hello struct {
	From string `json:"from"`
	Msg  []byte `json:"msg"`
}

type answer struct {
	Msg   []byte `json:"msg"`
	Proof []byte `json:"proof"`
}

type proof struct {
	Proof []byte `json:"proof"`
}

func (t *Trust) ConfirmHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp, ok := caller(r)
		if !ok {
			http.Error(w, "no certificate", http.StatusUnauthorized)
			return
		}
		key := t.o.Key()
		if !t.admit(source(r)) {
			http.Error(w, "too many tries", http.StatusTooManyRequests)
			return
		}
		var h hello
		if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(&h); err != nil {
			http.Error(w, "bad hello", http.StatusBadRequest)
			return
		}
		msg, secret, err := cpace.Exchange(key, cpace.NewContextInfo(fp, t.self, binding), h.Msg)
		if err != nil {
			http.Error(w, "bad hello", http.StatusBadRequest)
			return
		}
		t.mu.Lock()
		forgot := t.rekey(key)
		t.pending[fp] = pending{secret: secret, from: h.From, at: t.o.Now()}
		t.mu.Unlock()
		t.reconfirm(forgot)
		writeJSON(w, answer{Msg: msg, Proof: mac(secret, "b")})
	})
}

func (t *Trust) ProveHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fp, ok := caller(r)
		if !ok {
			http.Error(w, "no certificate", http.StatusUnauthorized)
			return
		}
		var p proof
		if err := json.NewDecoder(io.LimitReader(r.Body, bodyMost)).Decode(&p); err != nil {
			http.Error(w, "bad proof", http.StatusBadRequest)
			return
		}
		key := t.o.Key()
		t.mu.Lock()
		forgot := t.rekey(key)
		half, ok := t.pending[fp]
		delete(t.pending, fp)
		t.mu.Unlock()
		t.reconfirm(forgot)
		if !ok || t.o.Now().Sub(half.at) > pendingLife {
			http.Error(w, "no confirmation under way", http.StatusConflict)
			return
		}
		verdict := peer.Home
		if !hmac.Equal(mac(half.secret, "a"), p.Proof) {
			verdict = peer.Foreign
		}
		t.judge(key, "", fp, verdict)
		if verdict == peer.Home {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "not in this home", http.StatusForbidden)
		}
		t.recheck(half.from, verdict)
	})
}

func (t *Trust) recheck(id string, verdict peer.Standing) {
	p, ok := t.known(id)
	if !ok || t.Standing(id) == verdict {
		return
	}
	t.Found(p)
}

func (t *Trust) confirm(ctx context.Context, p peer.Peer, key string) bool {
	addr, err := p.SecureAddr()
	if err != nil {
		t.judge(key, p.ID, "", legacy(key))
		return true
	}
	ctx, cancel := context.WithTimeout(ctx, askWait)
	defer cancel()
	theirs, err := t.certAt(ctx, addr)
	if err != nil {
		slog.Debug("confirming a peer failed", "peer", p.Name, "err", err)
		return false
	}
	s, err := t.exchange(ctx, addr, theirs, key)
	if err != nil {
		slog.Debug("confirming a peer failed", "peer", p.Name, "err", err)
		return false
	}
	if s == peer.Foreign {
		slog.Info("peer is not in this home", "peer", p.Name)
	}
	t.judge(key, p.ID, theirs, s)
	return true
}

func (t *Trust) exchange(ctx context.Context, addr, theirs, key string) (peer.Standing, error) {
	pinned := t.transport(func(fp string) error {
		if fp != theirs {
			return errors.New("trust: the peer's certificate changed mid-confirmation")
		}
		return nil
	})
	defer pinned.CloseIdleConnections()
	client := &http.Client{Timeout: askWait, Transport: pinned}

	msg, state, err := cpace.Start(key, cpace.NewContextInfo(t.self, theirs, binding))
	if err != nil {
		return 0, err
	}
	var got answer
	status, err := post(
		ctx,
		client,
		"https://"+addr+ConfirmPath,
		hello{From: t.o.ID, Msg: msg},
		&got,
	)
	switch {
	case err != nil:
		return 0, err
	case status != http.StatusOK:
		return 0, fmt.Errorf("trust: confirm answered %d", status)
	}
	secret, err := state.Finish(got.Msg)
	if err != nil {
		return peer.Foreign, nil
	}
	matched := hmac.Equal(mac(secret, "b"), got.Proof)
	status, err = post(ctx, client, "https://"+addr+ProvePath, proof{Proof: mac(secret, "a")}, nil)
	switch {
	case !matched, status == http.StatusForbidden:
		return peer.Foreign, nil
	case err != nil:
		return 0, err
	case status != http.StatusNoContent:
		return 0, fmt.Errorf("trust: prove answered %d", status)
	}
	return peer.Home, nil
}

func (t *Trust) certAt(ctx context.Context, addr string) (string, error) {
	d := tls.Dialer{
		NetDialer: &net.Dialer{},
		Config: &tls.Config{
			Certificates:       []tls.Certificate{t.o.Identity},
			InsecureSkipVerify: true, //nolint:gosec // the fingerprint is what gets confirmed
			MinVersion:         tls.VersionTLS13,
		},
	}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	tc, ok := conn.(*tls.Conn)
	if !ok {
		return "", ErrNoCertificate
	}
	cert, ok := certOf(tc.ConnectionState())
	if !ok {
		return "", ErrNoCertificate
	}
	return Fingerprint(cert.Raw), nil
}

// TODO: remove with Legacy once no device predates the secure port.
func legacy(key string) peer.Standing {
	if key == "" {
		return peer.Open
	}
	return peer.Foreign
}

func (t *Trust) admit(from string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.o.Now()
	total := 0
	for ip, times := range t.asked {
		kept := times[:0]
		for _, at := range times {
			if now.Sub(at) < window {
				kept = append(kept, at)
			}
		}
		if len(kept) == 0 {
			delete(t.asked, ip)
			continue
		}
		t.asked[ip] = kept
		total += len(kept)
	}
	if len(t.asked[from]) >= burst || total >= ceiling {
		return false
	}
	t.asked[from] = append(t.asked[from], now)
	return true
}

func source(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func mac(secret []byte, side string) []byte {
	h := hmac.New(sha256.New, secret)
	h.Write([]byte(side))
	return h.Sum(nil)
}

func post(ctx context.Context, c *http.Client, url string, in, out any) (int, error) {
	body, err := json.Marshal(in)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if out != nil && resp.StatusCode == http.StatusOK {
		if err := json.NewDecoder(io.LimitReader(resp.Body, bodyMost)).Decode(out); err != nil {
			return 0, err
		}
	}
	return resp.StatusCode, nil
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
