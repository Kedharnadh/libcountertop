package homeassistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
	"google.golang.org/protobuf/proto"

	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	probeWait = 5 * time.Second
	syncWait  = 30 * time.Second
)

type sender interface {
	Send(proto.Message) error
}

type reply struct {
	data []byte
	err  error
}

type Settings struct {
	Home storage.Section[schema.Home]
	API  storage.Section[schema.API]
}
type Controller struct {
	settings  Settings
	ctx       context.Context
	stop      context.CancelFunc
	workers   sync.WaitGroup
	closed    bool
	probeStop context.CancelFunc

	Changed hook.Hook[Access]

	mu      sync.Mutex
	to      sender
	access  Access
	next    uint32
	waiting map[uint32]chan reply
	born    time.Time
}

func New(settings Settings) *Controller {
	ctx, stop := context.WithCancel(context.Background())
	return &Controller{
		settings: settings,
		ctx:      ctx,
		stop:     stop,
		waiting:  map[uint32]chan reply{},
		born:     time.Now(),
	}
}
func (h *Controller) Run(ctx context.Context) error { <-ctx.Done(); h.Close(); return nil }
func (h *Controller) Close() {
	h.mu.Lock()
	h.closed = true
	h.stop()
	if h.probeStop != nil {
		h.probeStop()
	}
	h.to = nil
	h.failPending()
	h.mu.Unlock()
	h.workers.Wait()
}
func (h *Controller) failPending() {
	for id, ch := range h.waiting {
		select {
		case ch <- reply{err: ErrNotConnected}:
		default:
		}
		delete(h.waiting, id)
	}
}

func (h *Controller) Name() string { return "homeassistant" }

func (h *Controller) Handle(_ context.Context, conn *esphome.Conn, msg proto.Message) error {
	switch m := msg.(type) {
	case *api.SubscribeHomeassistantServicesRequest:
		h.mu.Lock()
		if h.closed {
			h.mu.Unlock()
			return nil
		}
		if h.to != conn {
			h.failPending()
		}
		h.to = conn
		h.mu.Unlock()
		h.reprobe(conn)
	case *api.HomeassistantActionResponse:
		h.mu.Lock()
		ch := h.waiting[m.GetCallId()]
		delete(h.waiting, m.GetCallId())
		h.mu.Unlock()
		if ch == nil {
			return nil
		}
		if !m.GetSuccess() {
			ch <- reply{err: fmt.Errorf("homeassistant: %s", m.GetErrorMessage())}
			return nil
		}
		ch <- reply{data: m.GetResponseData()}
	}
	return nil
}

func (h *Controller) Startup() startup.Progress {
	return h.startup(h.settings.API.Read().Adopted, time.Now())
}

func (h *Controller) startup(adopted bool, now time.Time) startup.Progress {
	if !adopted {
		return startup.Progress{Done: true, Doing: "not adopted"}
	}
	if !h.settings.Home.Read().Enabled {
		return startup.Progress{Done: true, Doing: "home control off"}
	}
	switch h.Access() {
	case Allowed:
		return startup.Progress{Done: true, Doing: "synced"}
	case Refused:
		return startup.Progress{Failed: true, Doing: "actions not allowed"}
	}
	if now.Sub(h.born) > syncWait {
		return startup.Progress{Failed: true, Doing: "Home Assistant has not connected"}
	}
	if h.Connected() {
		return startup.Progress{Doing: "checking access"}
	}
	return startup.Progress{Doing: "waiting for Home Assistant"}
}

func (h *Controller) Access() Access {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.access
}

func (h *Controller) Connected() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.to != nil
}

func (h *Controller) Probe() {
	h.mu.Lock()
	conn := h.to
	h.mu.Unlock()
	if conn != nil {
		h.reprobe(conn)
	}
}

func (h *Controller) reprobe(conn sender) {
	if !h.settings.Home.Read().Enabled {
		return
	}
	ready := make(chan struct{})
	h.mu.Lock()
	current := !h.closed && h.to == conn
	if current {
		h.access = Unknown
		if h.probeStop != nil {
			h.probeStop()
		}
		ctx, stop := context.WithTimeout(h.ctx, probeWait)
		h.probeStop = stop
		h.workers.Add(1)
		go func() {
			defer h.workers.Done()
			defer stop()
			select {
			case <-ready:
				h.probe(ctx, conn)
			case <-ctx.Done():
			}
		}()
	}
	h.mu.Unlock()
	if !current {
		return
	}
	h.Changed.Emit(Unknown)
	close(ready)
}

func (h *Controller) probe(ctx context.Context, conn sender) {
	_, err := h.call(ctx, carrier, carrierData(), nil, "ok")
	switch {
	case err == nil:
		h.settle(conn, Allowed)
	case errors.Is(err, context.DeadlineExceeded):
		slog.Warn(
			"Home Assistant does not answer actions from this device; enable them in the ESPHome integration's options",
		)
		h.settle(conn, Refused)
	}
}

func (h *Controller) Enable(on bool) {
	if on {
		h.Probe()
		return
	}
	h.mu.Lock()
	moved := h.access != Unknown
	h.access = Unknown
	h.mu.Unlock()
	if moved {
		h.Changed.Emit(Unknown)
	}
}

func (h *Controller) settle(conn sender, access Access) {
	h.mu.Lock()
	moved := h.to == conn && h.access != access
	if moved {
		h.access = access
	}
	h.mu.Unlock()
	if moved {
		h.Changed.Emit(access)
	}
}

func (h *Controller) lost(conn sender) {
	h.mu.Lock()
	gone := h.to == conn
	if gone {
		h.to = nil
		h.access = Unknown
		if h.probeStop != nil {
			h.probeStop()
		}
		h.failPending()
	}
	h.mu.Unlock()
	if gone {
		h.Changed.Emit(Unknown)
	}
}

func (h *Controller) Call(ctx context.Context, action string, data map[string]string) error {
	_, err := h.call(ctx, action, data, nil, "")
	return err
}

func (h *Controller) Perform(
	ctx context.Context,
	action string,
	data, templates map[string]string,
) error {
	_, err := h.call(ctx, action, data, templates, "")
	return err
}

func (h *Controller) Template(ctx context.Context, tmpl string) (json.RawMessage, error) {
	raw, err := h.call(ctx, carrier, carrierData(), nil, tmpl)
	if err != nil {
		return nil, err
	}
	var out struct {
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("homeassistant: the response: %w", err)
	}
	return out.Response, nil
}

const carrier = "recorder.get_statistics"

func carrierData() map[string]string {
	return map[string]string{
		"statistic_ids": "countertop.none",
		"start_time":    time.Now().UTC().Format(time.RFC3339),
		"period":        "hour",
		"types":         "state",
	}
}

func (h *Controller) call(
	ctx context.Context,
	action string,
	data, templates map[string]string,
	tmpl string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ch := make(chan reply, 1)
	h.mu.Lock()
	to, access := h.to, h.access
	h.next++
	id := h.next
	if to != nil {
		h.waiting[id] = ch
	}
	h.mu.Unlock()
	if to == nil {
		return nil, ErrNotConnected
	}
	if access == Refused {
		h.mu.Lock()
		delete(h.waiting, id)
		h.mu.Unlock()
		return nil, ErrNotAllowed
	}
	defer func() {
		h.mu.Lock()
		delete(h.waiting, id)
		h.mu.Unlock()
	}()

	fields := make([]*api.HomeassistantServiceMap, 0, len(data))
	for k, v := range data {
		fields = append(fields, &api.HomeassistantServiceMap{Key: k, Value: v})
	}
	shaped := make([]*api.HomeassistantServiceMap, 0, len(templates))
	for k, v := range templates {
		shaped = append(shaped, &api.HomeassistantServiceMap{Key: k, Value: v})
	}
	req := &api.HomeassistantActionRequest{
		Service:          action,
		Data:             fields,
		DataTemplate:     shaped,
		CallId:           id,
		WantsResponse:    tmpl != "",
		ResponseTemplate: tmpl,
	}
	if err := to.Send(req); err != nil {
		h.lost(to)
		return nil, err
	}

	select {
	case r := <-ch:
		return r.data, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-h.ctx.Done():
		return nil, ErrNotConnected
	}
}

func (h *Controller) Watch(changed func(Access)) func() { return h.Changed.Listen(changed) }
