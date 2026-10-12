package calendar

import (
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const entitiesMost = 255

type (
	Setting = setting.Setting[schema.Calendar]
	Table   = setting.Table[schema.Calendar]
)

const Group setting.Group = "Calendar"

func NewTable() *Table {
	return setting.NewTable("calendar", []setting.Group{Group}, []Setting{
		setting.Words(
			Setting{Name: "entities", Group: Group, Icon: "mdi:calendar-multiple", Local: true},
			entitiesMost,
			func(c *schema.Calendar) *string { return &c.Entities },
		),
		setting.Pick(
			Setting{Name: "span", Group: Group, Icon: "mdi:calendar-range", Local: true},
			schema.CalendarSpans(),
			func(c *schema.Calendar) *schema.CalendarSpan { return &c.Span },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func (c *Calendar) Name() string { return "calendar" }

func (c *Calendar) Entities() []esphome.Entity { return c.knobs.Entities() }

func (c *Calendar) Settings() *setting.Controls[schema.Calendar] { return c.knobs }

func (c *Calendar) Publish(s schema.Calendar) { c.knobs.PublishFrom(s) }
