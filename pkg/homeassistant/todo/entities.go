package todo

import (
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const entityMost = 128

type (
	Setting = setting.Setting[schema.Todo]
	Table   = setting.Table[schema.Todo]
)

const Group setting.Group = "Todo"

func NewTable() *Table {
	return setting.NewTable("todo", []setting.Group{Group}, []Setting{
		setting.Words(
			Setting{Name: "entity", Group: Group, Icon: "mdi:cart-outline", Local: true},
			entityMost,
			func(c *schema.Todo) *string { return &c.Entity },
		),
		setting.Flag(
			Setting{
				Name:  "completed",
				Group: Group,
				Icon:  "mdi:checkbox-marked-outline",
				Local: true,
			},
			func(c *schema.Todo) *bool { return &c.Completed },
		),
	}, setting.Messages{Text: say.T, Missing: say.Missing})
}

func (t *Todo) Name() string { return "todo" }

func (t *Todo) Entities() []esphome.Entity { return t.knobs.Entities() }

func (t *Todo) Settings() *setting.Controls[schema.Todo] { return t.knobs }

func (t *Todo) Publish(c schema.Todo) { t.knobs.PublishFrom(c) }

func (t *Todo) store(s Setting, v string) error {
	if err := setting.Store(t.opts.Settings, s, v); err != nil {
		return err
	}
	t.opts.Redraw()
	return nil
}
