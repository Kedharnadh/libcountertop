package dashboard

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Choice struct {
	Labels []string
	Save   func(string) error
}

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

type controlOptions struct {
	DeviceID                         uint32
	Hours, Face, Position, Size, Ink Choice
	Date                             func(bool) error
	Read                             func() State
	Redraw                           func()
}

type Controls struct {
	preferences                       Preferences
	mu                                sync.Mutex
	options                           controlOptions
	Format, Face, Position, Size, Ink *esphome.Select
	Date                              *esphome.Switch
}

func newControls(options controlOptions) *Controls {
	c := &Controls{options: options}
	selects := []struct {
		id, name, icon string
		choice         Choice
		target         **esphome.Select
	}{
		{"clock_format", "Clock format", "mdi:clock-outline", options.Hours, &c.Format},
		{"clock_face", "Clock face", "mdi:clock-digital", options.Face, &c.Face},
		{
			"clock_position",
			"Clock position",
			"mdi:align-vertical-center",
			options.Position,
			&c.Position,
		},
		{"clock_size", "Clock size", "mdi:format-size", options.Size, &c.Size},
		{"clock_color", "Clock color", "mdi:palette-outline", options.Ink, &c.Ink},
	}
	for _, binding := range selects {
		entity := &esphome.Select{
			Base: esphome.Base{
				ObjectID: binding.id,
				Name:     binding.name,
				Icon:     binding.icon,
				Category: esphome.CategoryConfig,
				DeviceID: options.DeviceID,
			},
			Options: slices.Clone(binding.choice.Labels),
		}
		entity.OnCommand = func(label string) {
			if !slices.Contains(entity.Options, label) {
				return
			}
			c.Commit(entity.ObjectID, func() error { return binding.choice.Save(label) })
		}
		*binding.target = entity
	}
	c.Date = &esphome.Switch{
		Base: esphome.Base{
			ObjectID: "clock_date",
			Name:     "Clock date",
			Icon:     "mdi:calendar-blank-outline",
			Category: esphome.CategoryConfig,
			DeviceID: options.DeviceID,
		},
	}
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

type Redraw interface{ Redraw() }
type Preferences struct {
	Screen storage.Section[schema.Screen]
	Clock  storage.Section[schema.Clock]
}
type Dependencies struct {
	Preferences Preferences
	Display     Redraw
	DeviceID    uint32
}

func (p Preferences) State() State { return StateOf(p.Screen.Read(), p.Clock.Read()) }

func StateOf(s schema.Screen, c schema.Clock) State {
	return State{
		Hours:    s.Hours.Label(),
		Face:     c.Face.Label(),
		Position: c.Position.Label(),
		Size:     c.Size.Label(),
		Ink:      c.Ink.Label(),
		Date:     c.Date,
	}
}
func NewControls(d Dependencies) *Controls {
	p := d.Preferences
	c := newControls(controlOptions{
		DeviceID: d.DeviceID,
		Read:     p.State,
		Redraw:   d.Display.Redraw,
		Hours: Choices(
			schema.HourFormats(),
			func(v schema.HourFormat) error { return p.Screen.Update(func(s *schema.Screen) { s.Hours = v }) },
		),
		Face: Choices(
			schema.Faces(),
			func(v schema.Face) error { return p.Clock.Update(func(s *schema.Clock) { s.Face = v }) },
		),
		Position: Choices(
			schema.Positions(),
			func(v schema.Position) error { return p.Clock.Update(func(s *schema.Clock) { s.Position = v }) },
		),
		Size: Choices(
			schema.Sizes(),
			func(v schema.Size) error { return p.Clock.Update(func(s *schema.Clock) { s.Size = v }) },
		),
		Ink: Choices(
			schema.Inks(),
			func(v schema.Ink) error { return p.Clock.Update(func(s *schema.Clock) { s.Ink = v }) },
		),
		Date: func(v bool) error { return p.Clock.Update(func(s *schema.Clock) { s.Date = v }) },
	})
	c.preferences = p
	return c
}
func (c *Controls) SetHours(v schema.HourFormat) {
	c.Commit(
		"clock_format",
		func() error { return c.preferences.Screen.Update(func(s *schema.Screen) { s.Hours = v }) },
	)
}
func (c *Controls) SetFace(v schema.Face) {
	c.Commit(
		"clock_face",
		func() error { return c.preferences.Clock.Update(func(s *schema.Clock) { s.Face = v }) },
	)
}
func (c *Controls) SetPosition(v schema.Position) {
	c.Commit(
		"clock_position",
		func() error { return c.preferences.Clock.Update(func(s *schema.Clock) { s.Position = v }) },
	)
}
func (c *Controls) SetSize(v schema.Size) {
	c.Commit(
		"clock_size",
		func() error { return c.preferences.Clock.Update(func(s *schema.Clock) { s.Size = v }) },
	)
}
func (c *Controls) SetInk(v schema.Ink) {
	c.Commit(
		"clock_color",
		func() error { return c.preferences.Clock.Update(func(s *schema.Clock) { s.Ink = v }) },
	)
}
func (c *Controls) SetDate(v bool) {
	c.Commit(
		"clock_date",
		func() error { return c.preferences.Clock.Update(func(s *schema.Clock) { s.Date = v }) },
	)
}
