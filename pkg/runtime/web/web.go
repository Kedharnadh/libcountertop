package web

import (
	"context"
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
	srv  *http.Server

	mu     sync.Mutex
	shown  string
	routes map[string]http.Handler

	Offered hook.Hook[string]
}

func New(opts Options) *Server {
	return &Server{opts: opts, routes: map[string]http.Handler{}}
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

func (s *Server) Run(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.Handle("/", status.Handler(s.opts.Page))
	s.mu.Lock()
	for pattern, h := range s.routes {
		mux.Handle(pattern, h)
	}
	s.mu.Unlock()

	s.srv = &http.Server{
		Addr:              ":" + strconv.Itoa(s.opts.Port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	done := make(chan error, 1)
	go func() {
		slog.Info("serving pages", "port", s.opts.Port)

		err := s.srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		done <- err
	}()

	t := time.NewTicker(watch)
	defer t.Stop()

	for {
		s.offer()

		select {
		case err := <-done:
			return err
		case <-ctx.Done():
			s.hide()

			stop, cancel := context.WithTimeout(context.Background(), shutdown)
			defer cancel()
			return s.srv.Shutdown(stop)
		case <-t.C:
		}
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
