package message

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/hook"
)

const (
	DefaultSeconds = 15
	MaxSeconds     = 3600
)

type Messages struct {
	Changed hook.Hook[bool]

	mu         sync.Mutex
	timer      *time.Timer
	current    Message
	generation uint64
	took       int
}

func (m *Messages) Took(id int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.took == id+1
}

func (m *Messages) Current() (Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.current, m.timer != nil
}

func New() *Messages                              { return &Messages{} }
func (m *Messages) Run(ctx context.Context) error { <-ctx.Done(); m.Close(); return nil }
func (m *Messages) Close()                        { m.Hide() }

func (m *Messages) Dismiss(id int) {
	m.mu.Lock()
	showing := m.timer != nil
	m.took = 0
	if showing {
		m.took = id + 1
	}
	m.mu.Unlock()

	if !showing {
		return
	}
	slog.Info("message dismissed")
	m.Hide()
}

func (m *Messages) Name() string { return "messages" }

func (m *Messages) Actions() []*esphome.Action {
	return []*esphome.Action{
		{
			Name: "show_message",
			Args: []esphome.Arg{
				{Name: "body", Type: esphome.ArgString},
				{Name: "title", Type: esphome.ArgString},
				{Name: "tone", Type: esphome.ArgString},
				{Name: "seconds", Type: esphome.ArgInt},
			},
			Run: func(c esphome.Call) (any, error) {
				body := c.String("body")
				if body == "" {
					return nil, fmt.Errorf("show_message needs a body")
				}

				m.Show(Message{
					Title: c.String("title"),
					Body:  body,
					Tone:  tone(c.String("tone")),
				}, Hold(c.Int("seconds")))
				return nil, nil
			},
		},
		{
			Name: "clear_message",
			Run:  func(esphome.Call) (any, error) { m.Hide(); return nil, nil },
		},
	}
}

func (m *Messages) Show(msg Message, hold time.Duration) {
	m.mu.Lock()
	m.current = msg
	m.generation++
	generation := m.generation
	if m.timer != nil {
		m.timer.Stop()
	}
	m.timer = time.AfterFunc(hold, func() { m.expire(generation) })
	m.mu.Unlock()
	slog.Info("message", "title", msg.Title, "tone", msg.Tone, "for", hold)
	m.Changed.Emit(true)
}

func (m *Messages) Info(title, body string, hold time.Duration) {
	m.Show(Message{Title: title, Body: body, Tone: ToneInfo}, hold)
}

func (m *Messages) Hide() {
	m.mu.Lock()
	if m.timer == nil {
		m.mu.Unlock()
		return
	}
	m.generation++
	m.timer.Stop()
	m.timer = nil
	m.mu.Unlock()
	m.Changed.Emit(false)
}

func tone(name string) Tone {
	for _, t := range Tones() {
		if string(t) == name {
			return t
		}
	}
	return ToneInfo
}

func Hold(v int) time.Duration {
	switch {
	case v <= 0:
		v = DefaultSeconds
	case v > MaxSeconds:
		v = MaxSeconds
	}
	return time.Duration(v) * time.Second
}

func (m *Messages) expire(generation uint64) {
	m.mu.Lock()
	if m.timer == nil || m.generation != generation {
		m.mu.Unlock()
		return
	}
	m.timer = nil
	m.mu.Unlock()
	m.Changed.Emit(false)
}
