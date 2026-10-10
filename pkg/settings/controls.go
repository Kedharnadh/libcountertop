package setting

import (
	"errors"
	"log/slog"
	"strconv"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"
	"github.com/ygelfand/go-esphome-device/api"
)

const textMost = 255

type Controls[T any] struct {
	Table  *Table[T]
	Device uint32
	Want   func(Group) bool

	Read func() T
	Save func(s Setting[T], value string) error

	once  sync.Once
	built []esphome.Entity
	rows  []Setting[T]
}

func (c *Controls[T]) Entities() []esphome.Entity {
	c.once.Do(c.build)
	return c.built
}

func (c *Controls[T]) Entity(name string) esphome.Entity {
	c.once.Do(c.build)

	for i, s := range c.rows {
		if s.Name == name {
			return c.built[i]
		}
	}
	return nil
}

func (c *Controls[T]) Publish() { c.PublishFrom(c.Read()) }

func (c *Controls[T]) PublishFrom(at T) {
	c.once.Do(c.build)

	for i, s := range c.rows {
		switch e := c.built[i].(type) {
		case *esphome.Switch:
			e.Set(s.On(&at))
		case *esphome.Number:
			e.Set(float32(s.Level(&at)))
		case *esphome.Select:
			e.Set(s.LabelOf(s.Read(&at)))
		case *esphome.Text:
			e.Set(s.Read(&at))
		}
	}
}

func (c *Controls[T]) build() {
	at := c.Read()

	for _, g := range c.Table.Groups() {
		if c.Want != nil && !c.Want(g) {
			continue
		}
		for _, s := range c.Table.In(g) {
			e := c.entity(s, &at)
			if e == nil {
				continue
			}
			c.built = append(c.built, e)
			c.rows = append(c.rows, s)
		}
	}
}

// esphome.Base carries a mutex and must not be copied.
func (c *Controls[T]) entity(s Setting[T], at *T) esphome.Entity {
	id, name, icon, device := s.Object(c.Table.Domain), s.Named(), s.Icon, c.Device

	switch s.Kind {
	case Toggle:
		e := &esphome.Switch{Base: esphome.Base{
			ObjectID: id, DeviceID: device, Name: name, Icon: icon,
			Category: esphome.CategoryConfig,
		}}
		e.Set(s.On(at))
		e.OnCommand = func(on bool) { c.put(s, OnOff(on)) }
		return e

	case Number:
		mode := esphome.NumberBox
		if s.Slider {
			mode = esphome.NumberSlider
		}
		e := &esphome.Number{
			Base: esphome.Base{
				ObjectID: id, DeviceID: device, Name: name, Icon: icon,
				Category: esphome.CategoryConfig,
			},
			Min: float64(s.Min), Max: float64(s.Max), Step: 1,
			Unit: s.Unit,
			Mode: mode,
		}
		e.Set(float32(s.Level(at)))
		e.OnCommand = func(v float32) { c.put(s, strconv.Itoa(int(v))) }
		return e

	case Text, Secret:
		e := &esphome.Text{
			Base: esphome.Base{
				ObjectID: id, DeviceID: device, Name: name, Icon: icon,
				Category: esphome.CategoryConfig,
			},
			MaxLength: textMost,
		}
		if s.Kind == Secret {
			e.Mode = api.TextMode_TEXT_MODE_PASSWORD
		}
		e.Set(s.Read(at))
		e.OnCommand = func(v string) { c.change(s, v) }
		return e

	case Choice:
		e := &esphome.Select{
			Base: esphome.Base{
				ObjectID: id, DeviceID: device, Name: name, Icon: icon,
				Category: esphome.CategoryConfig,
			},
			Options: s.Labels(),
		}
		e.Set(s.LabelOf(s.Read(at)))
		e.OnCommand = func(label string) { c.put(s, s.ValueOf(label)) }
		return e
	}
	return nil
}

func (c *Controls[T]) put(s Setting[T], v string) {
	if v != "" {
		c.change(s, v)
	}
}

func (c *Controls[T]) change(s Setting[T], v string) {
	if err := c.Save(s, v); err != nil {
		slog.Error("the setting could not be saved", "setting", s.Name, "err", err)
		return
	}
	c.Publish()
}

func (c *Controls[T]) Domain() string { return c.Table.Domain }

func (c *Controls[T]) Names() []string {
	rows := c.Table.Rows()
	out := make([]string, len(rows))
	for i, s := range rows {
		out[i] = s.Name
	}
	return out
}

func (c *Controls[T]) Value(name string) (string, error) {
	s, err := c.Table.Row(name)
	if err != nil {
		return "", err
	}
	at := c.Read()
	return s.Shows(&at), nil
}

func (c *Controls[T]) Change(name, value string) error {
	s, err := c.Table.Row(name)
	if err != nil {
		return err
	}
	if s.Kind == Choice {
		if v := s.ValueOf(value); v != "" {
			value = v
		}
	}
	probe := c.Read()
	if err := s.Write(&probe, value); err != nil {
		return err
	}
	if err := c.Save(s, value); err != nil {
		return err
	}
	c.Publish()
	return nil
}

func (c *Controls[T]) Export() map[string]string {
	at := c.Read()
	out := map[string]string{}
	for _, s := range c.Table.Rows() {
		if s.Portable() {
			out[s.Name] = s.Read(&at)
		}
	}
	return out
}

func (c *Controls[T]) Import(values map[string]string) (int, error) {
	var (
		applied int
		errs    []error
	)
	for name, value := range values {
		s, ok := c.Table.Find(name)
		if !ok || !s.Portable() {
			continue
		}
		probe := c.Read()
		if s.Read(&probe) == value {
			continue
		}
		if err := s.Write(&probe, value); err != nil {
			errs = append(errs, err)
			continue
		}
		if err := c.Save(s, value); err != nil {
			errs = append(errs, err)
			continue
		}
		applied++
	}
	if applied > 0 {
		c.Publish()
	}
	return applied, errors.Join(errs...)
}
