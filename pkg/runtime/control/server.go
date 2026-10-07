// Package control provides the local device harness transport and command tree.
package control

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

type Dispatch func(context.Context, []string) (string, error)

// Server owns a listener and the clients accepted during that listener's lifetime.
type Server struct {
	mu       sync.Mutex
	workers  sync.WaitGroup
	address  string
	dispatch Dispatch
	listener net.Listener
	clients  map[net.Conn]context.CancelFunc
}

func NewServer(address string, dispatch Dispatch) *Server {
	return &Server{address: address, dispatch: dispatch, clients: make(map[net.Conn]context.CancelFunc)}
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	l, err := net.Listen("unix", s.address)
	if err != nil {
		return fmt.Errorf("control: %w", err)
	}
	s.listener = l
	slog.Info("the control socket is open to root and adb shell", "address", s.address)
	s.workers.Add(1)
	safe.Go("control accept", func() { defer s.workers.Done(); s.accept(l) })
	return nil
}

// Wait joins stopped clients and the accept loop. Call after Close without a concurrent Start.
func (s *Server) Wait() { s.workers.Wait() }

// Close revokes existing connections as well as releasing the listening address.
func (s *Server) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
		slog.Info("the control socket is closed", "address", s.address)
	}
	for conn, cancel := range s.clients {
		cancel()
		_ = conn.Close()
	}
}

func (s *Server) accept(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		if uid, allowed := Caller(conn); !allowed {
			slog.Warn("control socket refused a caller", "uid", uid)
			_ = conn.Close()
			continue
		}
		s.mu.Lock()
		if s.listener != l {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		s.clients[conn] = cancel
		s.workers.Add(1)
		s.mu.Unlock()
		safe.Go("control client", func() {
			defer s.workers.Done()
			defer func() {
				cancel()
				s.mu.Lock()
				delete(s.clients, conn)
				s.mu.Unlock()
			}()
			Serve(ctx, conn, s.dispatch)
		})
	}
}

// Serve preserves partial output before a terminal error. Empty lines are harmless.
func Serve(ctx context.Context, conn net.Conn, dispatch Dispatch) {
	defer conn.Close()
	in := bufio.NewScanner(conn)
	in.Buffer(make([]byte, 4096), 1<<20)
	for in.Scan() {
		if ctx.Err() != nil {
			return
		}
		args, parseErr := arguments(in.Text())
		var out string
		err := parseErr
		if err == nil && len(args) > 0 {
			out, err = dispatch(ctx, args)
		}
		if out != "" {
			if _, writeErr := fmt.Fprintln(conn, strings.TrimRight(out, "\n")); writeErr != nil {
				return
			}
		}
		if err != nil {
			if _, writeErr := fmt.Fprintf(conn, "error: %v\n", err); writeErr != nil {
				return
			}
			continue
		}
		if _, err := fmt.Fprintln(conn, "ok"); err != nil {
			return
		}
	}
	if err := in.Err(); err != nil && ctx.Err() == nil {
		_, _ = fmt.Fprintf(conn, "error: reading command: %v\n", err)
	}
}

func arguments(line string) ([]string, error) {
	if strings.HasPrefix(strings.TrimSpace(line), "[") {
		var args []string
		if err := json.Unmarshal([]byte(line), &args); err != nil {
			return nil, fmt.Errorf("command arguments: %w", err)
		}
		return args, nil
	}
	return strings.Fields(line), nil
}
