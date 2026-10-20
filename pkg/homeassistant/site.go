package homeassistant

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/libp2p/zeroconf/v2"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

const (
	siteService = "_home-assistant._tcp"
	siteDomain  = "local."
	siteListen  = 4 * time.Second
	siteEvery   = 5 * time.Minute
)

type Site struct {
	Override func() string

	mu    sync.Mutex
	found string
}

func (s *Site) Name() string { return "home assistant site" }

func (s *Site) URL() string {
	if s.Override != nil {
		if u := strings.TrimRight(strings.TrimSpace(s.Override()), "/"); u != "" {
			return u
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.found
}

func (s *Site) Run(ctx context.Context) error {
	for {
		if u := s.browse(ctx); u != "" {
			s.mu.Lock()
			changed := s.found != u
			s.found = u
			s.mu.Unlock()
			if changed {
				slog.Info("Home Assistant found", "url", u)
			}
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(siteEvery):
		}
	}
}

func (s *Site) browse(ctx context.Context) string {
	ctx, cancel := context.WithTimeout(ctx, siteListen)
	defer cancel()
	entries := make(chan *zeroconf.ServiceEntry, 4)
	done := make(chan struct{})
	safe.Go("home assistant browse", func() {
		defer close(done)
		if err := zeroconf.Browse(ctx, siteService, siteDomain, entries); err != nil {
			slog.Debug("home assistant browse", "err", err)
		}
	})
	var found string
	for {
		select {
		case <-ctx.Done():
			<-done
			return found
		case e, ok := <-entries:
			if !ok {
				<-done
				return found
			}
			if found == "" {
				found = siteURL(e)
			}
		}
	}
}

func siteURL(e *zeroconf.ServiceEntry) string {
	txt := map[string]string{}
	for _, kv := range e.Text {
		k, v, _ := strings.Cut(kv, "=")
		txt[strings.ToLower(k)] = v
	}
	for _, k := range []string{"internal_url", "base_url"} {
		if u := strings.TrimRight(txt[k], "/"); u != "" {
			return u
		}
	}
	if len(e.AddrIPv4) == 0 {
		return ""
	}
	return "http://" + net.JoinHostPort(e.AddrIPv4[0].String(), strconv.Itoa(e.Port))
}
