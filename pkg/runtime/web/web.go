package web

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/hook"
	netaddress "github.com/ygelfand/libcountertop/pkg/network/address"
	"github.com/ygelfand/libcountertop/pkg/runtime/status"
)

const (
	shutdown = 2 * time.Second
	watch    = 5 * time.Second
)

const AdoptURL = "https://my.home-assistant.io/redirect/config_flow_start/?domain=esphome"

type Options struct {
	Port    int
	APIPort int
	Adopted func() bool
	Page    func() status.Page
}

type Server struct {
	opts Options

	mu     sync.Mutex
	shown  string
	routes map[string]http.Handler
	peers  map[string]http.Handler
	secure int
	tls    func() *tls.Config

	Offered hook.Hook[string]
}

func New(opts Options) *Server {
	return &Server{
		opts:   opts,
		routes: map[string]http.Handler{},
		peers:  map[string]http.Handler{},
	}
}

func (s *Server) Name() string { return "web" }

func (s *Server) Offering() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shown
}

func (s *Server) Handle(pattern string, h http.Handler) {
	s.mu.Lock()
	s.routes[pattern] = h
	s.mu.Unlock()
}

func (s *Server) HandlePeer(pattern string, h http.Handler) {
	s.mu.Lock()
	s.peers[pattern] = h
	s.mu.Unlock()
}

func (s *Server) Secure(port int, cfg func() *tls.Config) {
	s.mu.Lock()
	s.secure, s.tls = port, cfg
	s.mu.Unlock()
}

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/", status.Handler(s.opts.Page))
	peers := http.NewServeMux()
	s.mu.Lock()
	for pattern, h := range s.routes {
		mux.Handle(pattern, h)
	}
	for pattern, h := range s.peers {
		peers.Handle(pattern, h)
	}
	secure, cfg := s.secure, s.tls
	s.mu.Unlock()

	servers := []*http.Server{{
		Addr:              ":" + strconv.Itoa(s.opts.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}}
	if secure != 0 && cfg != nil {
		servers = append(servers, &http.Server{
			Addr:              ":" + strconv.Itoa(secure),
			Handler:           peers,
			TLSConfig:         cfg(),
			ReadHeaderTimeout: 5 * time.Second,
		})
	}

	done := make(chan error, len(servers))
	for _, srv := range servers {
		go func() {
			var err error
			if srv.TLSConfig != nil {
				slog.Info("serving peers", "port", secure)
				err = srv.ListenAndServeTLS("", "")
			} else {
				slog.Info("serving pages", "port", s.opts.Port)
				err = srv.ListenAndServe()
			}
			if errors.Is(err, http.ErrServerClosed) {
				err = nil
			}
			done <- err
		}()
	}

	t := time.NewTicker(watch)
	defer t.Stop()

	for {
		s.offer()

		select {
		case err := <-done:
			closeAll(servers)
			return err
		case <-ctx.Done():
			s.hide()

			stop, cancel := context.WithTimeout(context.Background(), shutdown)
			defer cancel()
			var errs []error
			for _, srv := range servers {
				errs = append(errs, srv.Shutdown(stop))
			}
			return errors.Join(errs...)
		case <-t.C:
		}
	}
}

func closeAll(servers []*http.Server) {
	for _, srv := range servers {
		_ = srv.Close()
	}
}

func (s *Server) offer() {
	if s.opts.Adopted() {
		s.hide()
		return
	}

	ip := address()
	if ip == "" {
		return
	}
	addr := net.JoinHostPort(ip, strconv.Itoa(s.opts.APIPort))

	s.mu.Lock()
	if s.shown == addr {
		s.mu.Unlock()
		return
	}
	s.shown = addr
	s.mu.Unlock()
	slog.Info("waiting to be adopted", "at", addr)
	s.Offered.Emit(addr)
}

func (s *Server) hide() {
	s.mu.Lock()
	if s.shown == "" {
		s.mu.Unlock()
		return
	}
	s.shown = ""
	s.mu.Unlock()
	s.Offered.Emit("")
}

func address() string {
	for _, ip := range netaddress.Addresses() {
		if ip.To4() != nil {
			return ip.String()
		}
	}
	return ""
}
