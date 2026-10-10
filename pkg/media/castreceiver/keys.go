package castreceiver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/ygelfand/libcountertop/pkg/fetch"
	"github.com/ygelfand/libcountertop/pkg/media/cast"
)

const retry = 30 * time.Second

const drift = 5 * time.Minute

// Stock Cast devices send this revocation list with every auth answer.
const crlURL = "https://clients3.google.com/cast/chromecast/device/crl"

const crlEvery = 24 * time.Hour

const (
	oracleMost = 1 << 20
	bundleMost = 16 << 20
)

type keys struct {
	name      string
	path      string
	authority *cast.Authority

	held     atomic.Pointer[held]
	crl      atomic.Pointer[[]byte]
	crlURL   string
	retarget chan struct{}
	changed  func(*cast.Credentials)
	oracle   func() string
	bundle   func() string
	synced   func(do func()) (stop func())

	bundlePath string
	tried      string
	dry        dryBundle
}

type dryBundle struct {
	url   string
	until time.Time
}

type held struct {
	creds *cast.Credentials
	from  string
}

type cached struct {
	URL    string          `json:"url"`
	Answer json.RawMessage `json:"answer"`
}

func newKeys(name, path, bundlePath, authorityPath string) (*keys, error) {
	a, err := authority(name, authorityPath)
	if err != nil {
		return nil, err
	}
	k := &keys{
		name:       name,
		path:       path,
		bundlePath: bundlePath,
		authority:  a,
		crlURL:     crlURL,
		retarget:   make(chan struct{}, 1),
	}
	if err := k.issue(time.Now()); err != nil {
		return nil, err
	}
	return k, nil
}

func authority(name, path string) (*cast.Authority, error) {
	if raw, err := os.ReadFile(path); err == nil {
		a, err := cast.LoadAuthority(raw, name)
		if err == nil {
			return a, nil
		}
		slog.Warn("making a new cast authority", "err", err)
	}
	a, err := cast.NewAuthority(name)
	if err != nil {
		return nil, err
	}
	data, err := a.Marshal()
	if err == nil {
		err = writeKept(path, data)
	}
	if err != nil {
		slog.Warn("keeping the cast authority", "err", err)
	}
	return a, nil
}

func writeKept(path string, data []byte) error {
	tmp := path + ".new"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (k *keys) Current() *cast.Credentials {
	if h := k.held.Load(); h != nil {
		return h.creds
	}
	return nil
}

func (k *keys) CRL() []byte {
	if c := k.crl.Load(); c != nil {
		return *c
	}
	return nil
}

func (k *keys) revocations(ctx context.Context) {
	for {
		wait := crlEvery
		body, err := ask(ctx, k.crlURL, oracleMost)
		switch {
		case err != nil:
			slog.Warn("fetching the cast revocation list", "err", err)
			wait = retry
		case len(body) == 0:
			slog.Warn("the cast revocation list came back empty")
			wait = retry
		default:
			k.crl.Store(&body)
			slog.Debug("cast revocation list", "bytes", len(body))
		}

		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}
	}
}

func (k *keys) Retarget() {
	select {
	case k.retarget <- struct{}{}:
	default:
	}
}

func (k *keys) set(c *cast.Credentials, from string) {
	k.held.Store(&held{creds: c, from: from})
	slog.Info(
		"cast credentials",
		"remote",
		c.Remote,
		"notBefore",
		c.NotBefore,
		"notAfter",
		c.NotAfter,
	)
	if k.changed != nil {
		k.changed(c)
	}
}

func (k *keys) issue(now time.Time) error {
	c, err := k.authority.Issue(k.name, now)
	if err != nil {
		return err
	}
	k.set(c, "")
	return nil
}

func (k *keys) run(ctx context.Context) {
	synced := make(chan struct{}, 1)
	defer k.synced(func() {
		select {
		case synced <- struct{}{}:
		default:
		}
	})()

	if c, from := k.load(k.oracle()); c != nil {
		k.set(c, from)
	}
	go k.revocations(ctx)

	for {
		t := time.NewTimer(k.step(ctx, k.bundle(), k.oracle()))
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		case <-k.retarget:
			t.Stop()
		case <-synced:
			t.Stop()
		}
	}
}

func (k *keys) step(ctx context.Context, bundle, oracle string) time.Duration {
	if bundle != "" {
		if wait, ok := k.fromBundle(ctx, bundle, time.Now()); ok {
			return wait
		}
	}
	wait := k.fromOracle(ctx, oracle)
	if k.dry.url == bundle && !k.dry.until.IsZero() {
		wait = min(wait, time.Until(k.dry.until))
	}
	return wait
}

func (k *keys) fromBundle(ctx context.Context, url string, now time.Time) (time.Duration, bool) {
	if h := k.held.Load(); h.creds.Remote && h.from == url && now.Before(h.creds.NotAfter) {
		return h.creds.NotAfter.Sub(now), true
	}
	if k.dry.url == url && (k.dry.until.IsZero() || now.Before(k.dry.until)) {
		return 0, false
	}

	b, err := k.loadBundle(ctx, url)
	if err != nil {
		slog.Warn("no cast bundle", "err", err)
		k.dry = dryBundle{url: url}
		return 0, false
	}
	c, next, err := b.Credentials(now)
	if err != nil {
		slog.Warn("reading the cast bundle", "err", err)
		k.dry = dryBundle{url: url}
		return 0, false
	}
	if c == nil {
		slog.Warn("the cast bundle has no certificate for now", "next", next)
		k.dry = dryBundle{url: url, until: next}
		return 0, false
	}
	k.set(c, url)
	return c.NotAfter.Sub(now), true
}

func (k *keys) loadBundle(ctx context.Context, url string) (*cast.Bundle, error) {
	if c, ok := readKept(k.bundlePath); ok && c.URL == url {
		return cast.ReadBundle(c.Answer)
	}
	if k.tried == url {
		return nil, fmt.Errorf("the cast bundle from %s did not download", url)
	}
	k.tried = url
	if err := os.Remove(k.bundlePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		slog.Warn("removing the kept cast bundle", "err", err)
	}
	return k.fetchBundle(ctx, url)
}

func (k *keys) fromOracle(ctx context.Context, url string) time.Duration {
	now := time.Now()
	h := k.held.Load()

	if url == "" {
		if h.creds.Remote || !now.Before(h.creds.NotAfter) {
			if err := k.issue(now); err != nil {
				slog.Warn("making cast credentials", "err", err)
				return retry
			}
			h = k.held.Load()
		}
		return time.Until(h.creds.NotAfter)
	}

	if h.creds.Remote && h.from == url && now.Before(h.creds.NotAfter) {
		return time.Until(h.creds.NotAfter)
	}
	if !h.creds.Remote && !now.Before(h.creds.NotAfter) {
		if err := k.issue(now); err != nil {
			slog.Warn("making cast credentials", "err", err)
		}
	}

	body, err := ask(ctx, url, oracleMost)
	if err != nil {
		slog.Warn("asking the cast oracle", "err", err)
		return retry
	}
	c, theirs, err := cast.ParseOracle(body)
	if err != nil {
		slog.Warn("reading the cast oracle", "err", err)
		return retry
	}

	if d := now.Sub(theirs); d > drift || d < -drift {
		slog.Warn("the clock disagrees with the cast oracle", "ours", now, "theirs", theirs)
	}
	if c.NotBefore.After(theirs) || !theirs.Before(c.NotAfter) {
		slog.Warn("the cast oracle's certificate is not current",
			"notBefore", c.NotBefore, "notAfter", c.NotAfter, "now", theirs)
		return retry
	}

	k.set(c, url)
	if err := saveCached(k.path, url, body); err != nil {
		slog.Warn("keeping the cast oracle's answer", "err", err)
	}
	return c.NotAfter.Sub(theirs)
}

func (k *keys) fetchBundle(ctx context.Context, url string) (*cast.Bundle, error) {
	body, err := ask(ctx, url, bundleMost)
	if err != nil {
		return nil, err
	}
	b, err := cast.ReadBundle(body)
	if err != nil {
		return nil, err
	}
	kept, err := json.Marshal(b)
	if err == nil {
		err = saveCached(k.bundlePath, url, kept)
	}
	if err != nil {
		slog.Warn("keeping the cast bundle", "err", err)
	}
	return b, nil
}

func ask(ctx context.Context, url string, most int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := fetch.Client(30 * time.Second).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", url, resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, most))
}

func (k *keys) load(url string) (*cast.Credentials, string) {
	if url == "" {
		return nil, ""
	}
	answer := readCached(k.path, url)
	if answer == nil {
		return nil, ""
	}
	creds, _, err := cast.ParseOracle(answer)
	if err != nil {
		slog.Warn("reading the kept cast oracle answer", "err", err)
		return nil, ""
	}
	return creds, url
}

func readCached(path, url string) []byte {
	c, ok := readKept(path)
	if !ok || c.URL != url {
		return nil
	}
	return c.Answer
}

func readKept(path string) (cached, bool) {
	var c cached
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, &c) != nil {
		return cached{}, false
	}
	return c, true
}

func saveCached(path, url string, answer []byte) error {
	data, err := json.Marshal(cached{URL: url, Answer: answer})
	if err != nil {
		return err
	}

	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)

	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

type kept struct{ path string }

func store(dir string) func(app string) cast.Kept {
	return func(app string) cast.Kept { return kept{path: filepath.Join(dir, app+".json")} }
}

func (k kept) Load(v any) bool {
	data, err := os.ReadFile(k.path)
	return err == nil && json.Unmarshal(data, v) == nil
}

func (k kept) Save(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(k.path), 0o700); err != nil {
		return err
	}
	return writeKept(k.path, data)
}

func (k kept) Clear() error {
	if err := os.Remove(k.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
