package control

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
)

type Dispatch func(context.Context, []string) (string, error)

type Server struct {
	mu       sync.Mutex
	workers  sync.WaitGroup
	address  string
	dispatch Dispatch
	listener net.Listener
	clients  map[net.Conn]context.CancelFunc
}

func NewServer(address string, dispatch Dispatch) *Server {
	return &Server{
		address:  address,
		dispatch: dispatch,
		clients:  make(map[net.Conn]context.CancelFunc),
	}
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

func (s *Server) Wait() { s.workers.Wait() }

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

// Framing: output lines start "|", error lines "!", and a bare ok or error ends the reply.
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
		if reply(conn, out, err) != nil {
			return
		}
	}
	if err := in.Err(); err != nil && ctx.Err() == nil {
		_ = reply(conn, "", fmt.Errorf("reading command: %w", err))
	}
}

func reply(w io.Writer, out string, failed error) error {
	var b strings.Builder
	if out = strings.TrimRight(out, "\n"); out != "" {
		for line := range strings.SplitSeq(out, "\n") {
			b.WriteString(markOutput + line + "\n")
		}
	}
	if failed == nil {
		b.WriteString(endOK + "\n")
	} else {
		for line := range strings.SplitSeq(failed.Error(), "\n") {
			b.WriteString(markError + line + "\n")
		}
		b.WriteString(endError + "\n")
	}
	_, err := io.WriteString(w, b.String())
	return err
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
