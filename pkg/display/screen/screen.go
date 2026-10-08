package screen

import (
	"log/slog"
	"sync"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Options struct {
	Settings  storage.Section[schema.Screen]
	DeviceID  uint32
	Backlight func(level int) error
	UISize    func() string
}

type Screen struct {
	Themed hook.Hook[theme.Theme]

	Restyled hook.Hook[schema.Screen]

	options Options
	table   *Table
	knobs   *setting.Controls[schema.Screen]
	level   *esphome.Sensor

	mu      sync.Mutex
	current theme.Theme

	lit int
}

func New(options Options) *Screen {
	s := &Screen{options: options, lit: -1, current: theme.Default()}
	s.table = NewTable(options.UISize)
	s.knobs = &setting.Controls[schema.Screen]{
		Table:  s.table,
		Device: options.DeviceID,
		Read:   options.Settings.Read,
		Save:   s.save,
	}
	s.level = &esphome.Sensor{
		Base: esphome.Base{
			ObjectID: "screen_brightness",
			DeviceID: options.DeviceID,
			Name:     "Screen brightness",
			Icon:     "mdi:brightness-percent",
			Category: esphome.CategoryDiagnostic,
		},
		Unit:       "%",
		StateClass: esphome.StateClassMeasurement,
	}
	return s
}

func (s *Screen) Name() string { return "screen" }

func (s *Screen) Table() *Table { return s.table }

func (s *Screen) Settings() *setting.Controls[schema.Screen] { return s.knobs }

func (s *Screen) Entities() []esphome.Entity {
	return append(s.knobs.Entities(), s.level)
}

func (s *Screen) knob(name string) esphome.Entity { return s.knobs.Entity(name) }

func (s *Screen) Restore(c schema.Screen) {
	if c.Mode != schema.ModeAuto {
		s.applyBacklight(c.Backlight)
	}

	s.Use(c.Theme)
	s.knobs.PublishFrom(c)
}

func (s *Screen) Theme() theme.Theme {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *Screen) Use(name string) {
	at := s.options.Settings.Read()
	t, ok := theme.ByName(style.Theme(at.Style, name))
	if !ok {
		slog.Warn("no such theme", "theme", name)
		return
	}

	s.mu.Lock()
	changed := s.current.Name != t.Name
	s.current = t
	s.mu.Unlock()

	at.Theme = t.Name
	s.knobs.PublishFrom(at)

	if changed {
		s.Themed.Emit(t)
	}
}

func (s *Screen) SetBacklight(level int) {
	settled := max(0, min(level, 100))
	if s.options.Settings.Read().Mode != schema.ModeAuto {
		settled = s.applyBacklight(settled)
	}

	if err := s.options.Settings.Update(
		func(c *schema.Screen) { c.Backlight = settled },
	); err != nil {
		slog.Error("saving a setting failed", "setting", "backlight", "err", err)
	}
	s.knobs.Publish()
}

func (s *Screen) Lit(percent int) {
	s.mu.Lock()
	same := s.lit == percent
	s.lit = percent
	s.mu.Unlock()

	if !same {
		s.level.Set(float32(percent))
	}
}

func (s *Screen) applyBacklight(level int) int {
	level = max(0, min(level, 100))

	if err := s.options.Backlight(level); err != nil {
		slog.Error("setting the backlight failed", "level", level, "err", err)
		return s.options.Settings.Read().Backlight
	}

	s.Lit(level)
	return level
}

func (s *Screen) save(row Setting, value string) error {
	now := s.options.Settings.Read()
	if err := row.Write(&now, value); err != nil {
		return err
	}
	if err := s.options.Settings.Update(func(c *schema.Screen) { *c = now }); err != nil {
		return err
	}
	if now.Language != "" && now.Language != say.Chosen() {
		say.Use(now.Language)
	}

	if now.Mode == schema.ModeManual {
		s.applyBacklight(now.Backlight)
	}
	s.Use(now.Theme)
	s.Restyled.Emit(now)
	return nil
}

func (s *Screen) Set(name, value string) error {
	row, err := s.table.Row(name)
	if err != nil {
		return err
	}
	if err := s.save(row, value); err != nil {
		return err
	}
	s.knobs.Publish()
	return nil
}

func (s *Screen) SetDrawer(edge schema.Edge) {
	if err := s.options.Settings.Update(func(c *schema.Screen) { c.Drawer = edge }); err != nil {
		slog.Error("saving a setting failed", "setting", "drawer", "err", err)
	}
	s.knobs.Publish()
}

func (s *Screen) SetMode(mode schema.ScreenMode) {
	if err := s.options.Settings.Update(func(c *schema.Screen) { c.Mode = mode }); err != nil {
		slog.Error("saving a setting failed", "setting", "auto", "err", err)
	}

	if mode == schema.ModeManual {
		s.applyBacklight(s.options.Settings.Read().Backlight)
	}
	s.knobs.Publish()
}
