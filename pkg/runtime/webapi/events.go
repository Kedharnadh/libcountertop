package webapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	esphome "github.com/ygelfand/go-esphome-device"
)

const (
	queueMost = 256
	pingEvery = 30 * time.Second
	writeWait = 10 * time.Second
	readMost  = 4 << 10
)

type event struct {
	Type   string          `json:"type"`
	State  *esphome.State  `json:"state,omitempty"`
	States []esphome.State `json:"states,omitempty"`
}

var upgrader = websocket.Upgrader{}

func (a *API) events(w http.ResponseWriter, r *http.Request) {
	if a.o.States == nil {
		http.Error(w, "no states yet", http.StatusServiceUnavailable)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	queue := make(chan esphome.State, queueMost)
	full := make(chan struct{})
	stop := a.o.States.Listen(func(s esphome.State) {
		select {
		case queue <- s:
		default:
			select {
			case <-full:
			default:
				close(full)
			}
		}
	})
	defer stop()

	gone := make(chan struct{})
	conn.SetReadLimit(readMost)
	go func() {
		defer close(gone)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	if !a.send(conn, event{Type: "states", States: a.o.States.States()}) {
		return
	}
	ping := time.NewTicker(pingEvery)
	defer ping.Stop()
	for {
		select {
		case s := <-queue:
			if !a.send(conn, event{Type: "state", State: &s}) {
				return
			}
		case <-ping.C:
			if err := conn.WriteControl(
				websocket.PingMessage, nil, time.Now().Add(writeWait),
			); err != nil {
				return
			}
		case <-full:
			slog.Info("closing a slow events client", "from", remote(r))
			return
		case <-gone:
			return
		}
	}
}

func (a *API) send(conn *websocket.Conn, e event) bool {
	_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
	return conn.WriteJSON(e) == nil
}
