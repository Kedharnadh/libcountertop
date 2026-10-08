package sendspin

import (
	"sync"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type memory struct {
	mu sync.Mutex
	s  schema.Sendspin
}

func (m *memory) Read() schema.Sendspin {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.s
}

func (m *memory) Update(change func(*schema.Sendspin)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	change(&m.s)
	return nil
}

func TestEnablingTellsBothSides(t *testing.T) {
	m := &memory{}
	p := New(Dependencies{Settings: m})

	for _, want := range []bool{true, false, true} {
		p.SetEnabled(want)

		if got := m.Read().Enabled; got != want {
			t.Errorf("the setting says %v, want %v", got, want)
		}
		if got := p.Entities()[0].(*esphome.Switch).Get(); got != want {
			t.Errorf("the entity says %v, want %v", got, want)
		}
	}
}

func TestTheCommandFromHomeAssistantIsSaved(t *testing.T) {
	m := &memory{}
	p := New(Dependencies{Settings: m})

	p.Entities()[0].(*esphome.Switch).OnCommand(true)
	if !m.Read().Enabled {
		t.Error("turning it on from Home Assistant did not save")
	}

	p.Entities()[0].(*esphome.Switch).OnCommand(false)
	if m.Read().Enabled {
		t.Error("turning it off from Home Assistant did not save")
	}
}

func TestItIsOffOnAFreshDevice(t *testing.T) {
	if schema.DefaultSendspin().Enabled {
		t.Error("a device nobody has set up is listening for a server")
	}
}
