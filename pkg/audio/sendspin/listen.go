package sendspin

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const Port = 8928

// The spec ranks competing servers by declared activity.
type Listener struct {
	out *Output
	bg  Arbitration

	callbacks Callbacks
	identity  func() Identity

	mu   sync.Mutex
	busy bool
}

func NewListener(
	o *Output,
	bg Arbitration,
	identity func() Identity,
	callbacks Callbacks,
) *Listener {
	return &Listener{out: o, bg: bg, identity: identity, callbacks: callbacks}
}

func (l *Listener) Serve(ctx context.Context, name string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(Port)))
	if err != nil {
		return err
	}

	up := websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool { return true },
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if !l.take() {
			http.Error(w, "already connected", http.StatusConflict)
			return
		}
		defer l.give()

		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("sendspin upgrade failed", "from", r.RemoteAddr, "err", err)
			return
		}
		defer conn.Close()

		slog.Info("sendspin server connected", "from", r.RemoteAddr)
		l.callbacks.state(stateJoined)
		defer l.callbacks.state(stateWaiting)

		s := newSession(conn, l.out, l.bg, name, l.identity(), l.callbacks)
		l.callbacks.connection(s)
		defer l.callbacks.connection(nil)

		if err := s.run(ctx); err != nil {
			slog.Warn("sendspin session ended", "from", r.RemoteAddr, "err", err)
		} else {
			slog.Info("sendspin server disconnected", "from", r.RemoteAddr)
		}
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (l *Listener) take() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.busy {
		return false
	}
	l.busy = true
	return true
}

func (l *Listener) give() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.busy = false
}
