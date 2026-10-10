package profiler

import (
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

const Addr = "127.0.0.1:6060"

type Server struct {
	Addr string

	mu  sync.Mutex
	srv *http.Server
	at  string
}

func (s *Server) Open() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv != nil {
		return s.at, nil
	}

	addr := s.Addr
	if addr == "" {
		addr = Addr
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return "", err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s.srv, s.at = srv, l.Addr().String()
	safe.Go("profiler", func() {
		if err := srv.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("the profiler stopped", "err", err)
		}
	})
	slog.Warn("the profiler is open", "addr", s.at)
	return s.at, nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.srv == nil {
		return nil
	}
	err := s.srv.Close()
	s.srv, s.at = nil, ""
	slog.Info("the profiler is closed")
	return err
}

func (s *Server) Listening() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.at
}
