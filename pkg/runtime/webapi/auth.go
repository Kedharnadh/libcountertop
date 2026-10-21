package webapi

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	cookieName   = "session"
	sessionLife  = 30 * 24 * time.Hour
	sessionRenew = 15 * 24 * time.Hour
	nonceBytes   = 16
	signingLabel = "countertop web session"
	passwordSize = 12
	failsAllowed = 5
	failWindow   = 10 * time.Minute
	loginMost    = 4 << 10
)

const passwordAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

type sessions struct {
	mu      sync.Mutex
	revoked map[string]time.Time
	failed  map[string][]time.Time
}

func newPassword() (string, error) {
	raw := make([]byte, passwordSize)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, passwordSize)
	for i, b := range raw {
		out[i] = passwordAlphabet[int(b)%len(passwordAlphabet)]
	}
	return string(out), nil
}

func (a *API) signer() []byte {
	m := hmac.New(sha256.New, []byte(a.o.Settings.Read().Password))
	m.Write([]byte(signingLabel))
	return m.Sum(nil)
}

func (a *API) open(now time.Time) (string, error) {
	payload := make([]byte, 8+nonceBytes)
	binary.BigEndian.PutUint64(payload, uint64(now.Add(sessionLife).Unix()))
	if _, err := rand.Read(payload[8:]); err != nil {
		return "", err
	}
	return a.sign(payload), nil
}

func (a *API) sign(payload []byte) string {
	m := hmac.New(sha256.New, a.signer())
	m.Write(payload)
	enc := base64.RawURLEncoding
	return enc.EncodeToString(payload) + "." + enc.EncodeToString(m.Sum(nil))
}

func (a *API) expiry(token string, now time.Time) (time.Time, bool) {
	body, mac, ok := strings.Cut(token, ".")
	if !ok || a.o.Settings.Read().Password == "" {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil || len(payload) != 8+nonceBytes {
		return time.Time{}, false
	}
	got, err := base64.RawURLEncoding.DecodeString(mac)
	if err != nil {
		return time.Time{}, false
	}
	m := hmac.New(sha256.New, a.signer())
	m.Write(payload)
	if !hmac.Equal(got, m.Sum(nil)) {
		return time.Time{}, false
	}
	until := time.Unix(int64(binary.BigEndian.Uint64(payload)), 0)
	if now.After(until) {
		return time.Time{}, false
	}
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	if _, gone := a.s.revoked[token]; gone {
		return time.Time{}, false
	}
	return until, true
}

func (a *API) valid(token string, now time.Time) bool {
	_, ok := a.expiry(token, now)
	return ok
}

func (a *API) close(token string, now time.Time) {
	until, ok := a.expiry(token, now)
	if !ok {
		return
	}
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	for t, u := range a.s.revoked {
		if now.After(u) {
			delete(a.s.revoked, t)
		}
	}
	a.s.revoked[token] = until
}

func (a *API) setCookie(w http.ResponseWriter, token string, now time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Expires:  now.Add(sessionLife),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (a *API) matches(password string) bool {
	want := a.o.Settings.Read().Password
	return want != "" && subtle.ConstantTimeCompare([]byte(password), []byte(want)) == 1
}

func (a *API) throttled(from string, now time.Time) bool {
	a.s.mu.Lock()
	defer a.s.mu.Unlock()
	recent := a.s.failed[from][:0]
	for _, at := range a.s.failed[from] {
		if now.Sub(at) < failWindow {
			recent = append(recent, at)
		}
	}
	a.s.failed[from] = recent
	return len(recent) >= failsAllowed
}

func (a *API) failedFrom(from string, now time.Time) {
	a.s.mu.Lock()
	a.s.failed[from] = append(a.s.failed[from], now)
	a.s.mu.Unlock()
}

func remote(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *API) authorized(w http.ResponseWriter, r *http.Request) bool {
	now := a.o.Now()
	if c, err := r.Cookie(cookieName); err == nil {
		if until, ok := a.expiry(c.Value, now); ok {
			if until.Sub(now) < sessionRenew {
				if token, err := a.open(now); err == nil {
					a.setCookie(w, token, now)
				}
			}
			return true
		}
	}
	header := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(header, "Bearer "); ok {
		return a.valid(token, now)
	}
	if _, password, ok := r.BasicAuth(); ok {
		from := remote(r)
		if a.throttled(from, now) {
			return false
		}
		if a.matches(password) {
			return true
		}
		a.failedFrom(from, now)
	}
	return false
}

func (a *API) require(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.authorized(w, r) {
			http.Error(w, "sign in", http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(w, r)
	})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	now, from := a.o.Now(), remote(r)
	if a.throttled(from, now) {
		http.Error(w, "try again later", http.StatusTooManyRequests)
		return
	}
	var body struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, loginMost)).Decode(&body); err != nil {
		http.Error(w, "send a password", http.StatusBadRequest)
		return
	}
	if !a.matches(body.Password) {
		a.failedFrom(from, now)
		http.Error(w, "wrong password", http.StatusUnauthorized)
		return
	}
	token, err := a.open(now)
	if err != nil {
		http.Error(w, "try again", http.StatusInternalServerError)
		return
	}
	a.setCookie(w, token, now)
	reply(w, map[string]string{"token": token})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	now := a.o.Now()
	if c, err := r.Cookie(cookieName); err == nil {
		a.close(c.Value, now)
	}
	if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
		a.close(token, now)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	w.WriteHeader(http.StatusNoContent)
}
