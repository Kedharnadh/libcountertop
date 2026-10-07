package dashboard

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"
)

type Choice struct {
	Labels []string
	Save   func(string) error
}

// Choices binds translated enum labels to a product's typed config writer.
func Choices[T interface{ Label() string }](values []T, save func(T) error) Choice {
	values = slices.Clone(values)
	labels := make([]string, len(values))
	for i, value := range values {
		labels[i] = value.Label()
	}
	return Choice{Labels: labels, Save: func(label string) error {
		for _, value := range values {
			if value.Label() == label {
				return save(value)
			}
		}
		return fmt.Errorf("unknown dashboard choice %q", label)
	}}
}

type State struct {
	Hours, Face, Position, Size, Ink string
	Date                             bool
}

type ControlOptions struct {
	DeviceID                         uint32
	Hours, Face, Position, Size, Ink Choice
	Date                             func(bool) error
	Read                             func() State
	Redraw                           func()
}

// Controls owns the stable dashboard entity identities and their command/save/publication flow.
type Controls struct {
	mu                                sync.Mutex
	options                           ControlOptions
	Format, Face, Position, Size, Ink *esphome.Select
	Date                              *esphome.Switch
}

func NewControls(options ControlOptions) *Controls {
	c := &Controls{options: options}
	selects := []struct {
		id, name, icon string
		choice         Choice
		target         **esphome.Select
	}{
		{"clock_format", "Clock format", "mdi:clock-outline", options.Hours, &c.Format},
		{"clock_face", "Clock face", "mdi:clock-digital", options.Face, &c.Face},
		{"clock_position", "Clock position", "mdi:align-vertical-center", options.Position, &c.Position},
		{"clock_size", "Clock size", "mdi:format-size", options.Size, &c.Size},
		{"clock_color", "Clock color", "mdi:palette-outline", options.Ink, &c.Ink},
	}
	for _, binding := range selects {
		entity := &esphome.Select{Base: esphome.Base{ObjectID: binding.id, Name: binding.name, Icon: binding.icon, Category: esphome.CategoryConfig, DeviceID: options.DeviceID}, Options: slices.Clone(binding.choice.Labels)}
		entity.OnCommand = func(label string) {
			if !slices.Contains(entity.Options, label) {
				return
			}
			c.Commit(entity.ObjectID, func() error { return binding.choice.Save(label) })
		}
		*binding.target = entity
	}
	c.Date = &esphome.Switch{Base: esphome.Base{ObjectID: "clock_date", Name: "Clock date", Icon: "mdi:calendar-blank-outline", Category: esphome.CategoryConfig, DeviceID: options.DeviceID}}
	c.Date.OnCommand = func(on bool) { c.Commit(c.Date.ObjectID, func() error { return options.Date(on) }) }
	return c
}

func (c *Controls) Entities() []esphome.Entity {
	return []esphome.Entity{c.Format, c.Face, c.Position, c.Size, c.Ink, c.Date}
}

func (c *Controls) Restore(state State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.publish(state)
}

func (c *Controls) publish(state State) {
	c.Format.Set(state.Hours)
	c.Face.Set(state.Face)
	c.Position.Set(state.Position)
	c.Size.Set(state.Size)
	c.Ink.Set(state.Ink)
	c.Date.Set(state.Date)
}

// Commit publishes a fresh active snapshot, including when a save reports a post-rename error.
// Persistence runs outside the publication lock; concurrent writers cannot publish an old snapshot
// after a newer one because Read happens within that lock.
func (c *Controls) Commit(setting string, save func() error) {
	before := c.options.Read()
	err := save()
	c.mu.Lock()
	after := c.options.Read()
	c.publish(after)
	c.mu.Unlock()
	if before != after && c.options.Redraw != nil {
		c.options.Redraw()
	}
	if err != nil {
		slog.Error("saving a setting failed", "setting", setting, "err", err)
	}
}
