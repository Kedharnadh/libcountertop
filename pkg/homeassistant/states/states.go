package states

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
	"google.golang.org/protobuf/proto"

	"github.com/ygelfand/libcountertop/pkg/hook"
)

type States struct {
	mu        sync.Mutex
	following []*Value
	to        sender
}

type sender interface {
	Send(proto.Message) error
}

// A nil pointer in an interface is not a nil interface.
func wrap(conn *esphome.Conn) sender {
	if conn == nil {
		return nil
	}
	return conn
}

type Value struct {
	Entity    string
	Attribute string

	Changed  hook.Hook[string]
	Reported hook.Hook[string]

	mu    sync.Mutex
	state string
	known bool
}

func New() *States { return &States{} }

func (s *States) Name() string { return "states" }

func (s *States) Follow(entity, attribute string) *Value {
	entity = strings.TrimSpace(entity)
	if entity == "" {
		return &Value{}
	}

	s.mu.Lock()
	for _, v := range s.following {
		if v.Entity == entity && v.Attribute == attribute {
			s.mu.Unlock()
			return v
		}
	}

	v := &Value{Entity: entity, Attribute: attribute}
	s.following = append(s.following, v)
	to := s.to
	s.mu.Unlock()

	ask(to, v)
	return v
}

func (s *States) Following() []*Value {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*Value(nil), s.following...)
}

func (s *States) Handle(_ context.Context, conn *esphome.Conn, msg proto.Message) error {
	switch m := msg.(type) {
	case *api.SubscribeHomeAssistantStatesRequest:
		s.offer(wrap(conn))
	case *api.HomeAssistantStateResponse:
		s.take(m.GetEntityId(), m.GetAttribute(), m.GetState())
	}
	return nil
}

func (s *States) offer(to sender) {
	s.mu.Lock()
	s.to = to
	list := append([]*Value(nil), s.following...)
	s.mu.Unlock()

	for _, v := range list {
		ask(to, v)
	}
	slog.Debug("asked Home Assistant to report on", "entities", len(list))
}

func ask(to sender, v *Value) {
	if to == nil || v.Entity == "" {
		return
	}

	err := to.Send(&api.SubscribeHomeAssistantStateResponse{
		EntityId:  v.Entity,
		Attribute: v.Attribute,
		Once:      false,
	})
	if err != nil {
		slog.Debug("asking for a state failed", "entity", v.Entity, "err", err)
	}
}

func (s *States) take(entity, attribute, state string) {
	s.mu.Lock()
	list := append([]*Value(nil), s.following...)
	s.mu.Unlock()

	for _, v := range list {
		if v.Entity == entity && v.Attribute == attribute {
			v.set(state)
			return
		}
	}
	slog.Debug(
		"a state arrived for something nothing is following",
		"entity",
		entity,
		"attribute",
		attribute,
	)
}

func (v *Value) Get() (string, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.state, v.known
}

func (v *Value) Known() bool {
	_, ok := v.Get()
	return ok
}

// Home Assistant sends every state as a string, and "unavailable" for an unavailable entity.
func (v *Value) Float() (float64, bool) {
	s, ok := v.Get()
	if !ok {
		return 0, false
	}

	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func (v *Value) set(state string) {
	v.mu.Lock()
	same := v.known && v.state == state
	v.state, v.known = state, true
	v.mu.Unlock()

	if !same {
		v.Changed.Emit(state)
	}
	v.Reported.Emit(state)
}
