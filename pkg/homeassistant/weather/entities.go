package weather

import (
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const entityMost = 128

type (
	Setting = setting.Setting[schema.Weather]
	Table   = setting.Table[schema.Weather]
)

const Group setting.Group = "Weather"

func NewTable() *Table {
	flag := func(name, id, icon string, field func(*schema.Weather) *bool) Setting {
		return setting.Flag(Setting{Name: name, Group: Group, ID: id, Icon: icon}, field)
	}
	return setting.NewTable("weather", []setting.Group{Group}, []Setting{
		setting.Words(Setting{Name: "entity", Group: Group, Icon: "mdi:weather-partly-cloudy"},
			entityMost, func(c *schema.Weather) *string { return &c.Entity }),
		setting.Pick(Setting{Name: "look", Group: Group, Icon: "mdi:palette-outline"},
			schema.WeatherLooks(), func(c *schema.Weather) *schema.WeatherLook { return &c.Look }),
		flag("dashboard", "", "mdi:view-dashboard-outline",
			func(c *schema.Weather) *bool { return &c.Dashboard }),
		flag("idle", "weather_screensaver", "mdi:monitor-star",
			func(c *schema.Weather) *bool { return &c.Idle }),
		flag("animate", "", "mdi:animation-play-outline",
			func(c *schema.Weather) *bool { return &c.Animate }),
		flag("themed", "", "mdi:palette",
			func(c *schema.Weather) *bool { return &c.Themed }),
		setting.Pick(Setting{Name: "position", Group: Group, Icon: "mdi:arrow-up-down"},
			schema.Positions(), func(c *schema.Weather) *schema.Position { return &c.Position }),
		setting.Pick(Setting{Name: "align", Group: Group, Icon: "mdi:arrow-left-right"},
			schema.Aligns(), func(c *schema.Weather) *schema.Align { return &c.Align }),
		setting.Pick(Setting{Name: "size", Group: Group, Icon: "mdi:resize"},
			schema.Sizes(), func(c *schema.Weather) *schema.Size { return &c.Size }),
		seeded(setting.Pick(Setting{Name: "saver.look", Group: Group, Icon: "mdi:palette-outline"},
			schema.WeatherLooks(),
			func(c *schema.Weather) *schema.WeatherLook { return &c.SeedSaver().Look })),
		seeded(flag("saver.animate", "", "mdi:animation-play-outline",
			func(c *schema.Weather) *bool { return &c.SeedSaver().Animate })),
		seeded(flag("saver.themed", "", "mdi:palette",
			func(c *schema.Weather) *bool { return &c.SeedSaver().Themed })),
		seeded(
			setting.Pick(Setting{Name: "saver.position", Group: Group, Icon: "mdi:arrow-up-down"},
				schema.Positions(),
				func(c *schema.Weather) *schema.Position { return &c.SeedSaver().Position }),
		),
		seeded(
			setting.Pick(Setting{Name: "saver.align", Group: Group, Icon: "mdi:arrow-left-right"},
				schema.Aligns(),
				func(c *schema.Weather) *schema.Align { return &c.SeedSaver().Align }),
		),
		seeded(setting.Pick(Setting{Name: "saver.size", Group: Group, Icon: "mdi:resize"},
			schema.Sizes(),
			func(c *schema.Weather) *schema.Size { return &c.SeedSaver().Size })),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func seeded(s Setting) Setting {
	read := s.Read
	s.Read = func(c *schema.Weather) string {
		probe := *c
		return read(&probe)
	}
	return s
}

func (w *Weather) Name() string { return "weather" }

func (w *Weather) Entities() []esphome.Entity { return w.knobs.Entities() }

func (w *Weather) Settings() *setting.Controls[schema.Weather] { return w.knobs }

func (w *Weather) Publish(c schema.Weather) { w.knobs.PublishFrom(c) }

func (w *Weather) store(s Setting, v string) error {
	if err := setting.Store(w.opts.Settings, s, v); err != nil {
		return err
	}
	w.opts.Redraw()
	return nil
}
