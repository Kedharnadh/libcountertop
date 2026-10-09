package volume

import (
	"log/slog"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Output interface {
	SetLevel(schema.Stream, int)
	Mute(schema.Stream, bool)
	Muted(schema.Stream) bool
}

type MixerOptions struct {
	Output  Output
	Streams []schema.Stream
	Save    func(schema.Stream, int) error
	Device  uint32
}

type Mixer struct {
	Changed Changes

	output   Output
	save     func(schema.Stream, int) error
	streams  []schema.Stream
	entities *Entities

	mu     sync.Mutex
	levels map[schema.Stream]int
}

func NewMixer(o MixerOptions) *Mixer {
	m := &Mixer{
		output:  o.Output,
		save:    o.Save,
		streams: slices.Clone(o.Streams),
		levels:  map[schema.Stream]int{},
	}
	m.entities = NewEntities(m, m.streams, o.Device)
	return m
}

func (m *Mixer) Streams() []schema.Stream { return slices.Clone(m.streams) }

func (m *Mixer) Entities() []esphome.Entity { return m.entities.Entities() }

func (m *Mixer) Number(s schema.Stream) *esphome.Number { return m.entities.Numbers[s] }

func (m *Mixer) Restore(level func(schema.Stream) int) {
	for _, s := range m.streams {
		m.hold(s, level(s))
	}
}

func (m *Mixer) Level(s schema.Stream) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.levels[s]
}

func (m *Mixer) Set(s schema.Stream, level int) {
	if !slices.Contains(m.streams, s) {
		return
	}
	level = Clamp(level)
	if m.Level(s) == level {
		return
	}
	m.hold(s, level)
	m.Changed.Emit(Change{Stream: s, Level: level})

	if m.save == nil {
		return
	}
	if err := m.save(s, level); err != nil {
		slog.Error("saving a volume failed", "stream", s, "err", err)
	}
}

func (m *Mixer) Mute(s schema.Stream, on bool) {
	if m.output.Muted(s) == on {
		return
	}
	m.output.Mute(s, on)
	m.Changed.Emit(Change{Stream: s, Level: m.Level(s)})
}

func (m *Mixer) Muted(s schema.Stream) bool { return m.output.Muted(s) }

func (m *Mixer) Watch(s schema.Stream, changed func(int)) func() {
	return m.Changed.Watch(s, changed)
}

func (m *Mixer) hold(s schema.Stream, level int) {
	level = Clamp(level)

	m.mu.Lock()
	m.levels[s] = level
	m.mu.Unlock()

	m.output.SetLevel(s, level)
	if n := m.entities.Numbers[s]; n != nil {
		n.Set(float32(level))
	}
}
