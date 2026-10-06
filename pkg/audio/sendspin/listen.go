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

// listener accepts the servers that dial in, one at a time. The spec ranks competing servers by
// declared activity; until that is implemented the first to arrive holds the room.
type Listener struct {
	out *Output
	bg  Arbitration

	// player is what the room shows Home Assistant. Written to from the accept goroutine and from the
	// session, so it has to tolerate that.
	callbacks Callbacks
	identity  func() Identity

	mu   sync.Mutex
	busy bool
}

func NewListener(o *Output, bg Arbitration, identity func() Identity, callbacks Callbacks) *Listener {
	return &Listener{out: o, bg: bg, identity: identity, callbacks: callbacks}
}

// serve holds the port until ctx ends.
func (l *Listener) Serve(ctx context.Context, name string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ln, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(Port)))
	if err != nil {
		return err
	}

	up := websocket.Upgrader{
		// Any origin: a server dialing in is not a browser, and there is nothing here a page could
		// reach that the network cannot already.
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

		// For as long as the server is connected, not just while audio is arriving: a paused group is
		// one this room can still be told to play again.
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

	// Closing the server unblocks Serve and hangs up on whatever is connected, which is what stopping
	// means here: the room is leaving the group, not pausing.
	go func() {
		<-ctx.Done()
		_ = srv.Close()
	}()

	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// take admits one server and turns away the rest.
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
