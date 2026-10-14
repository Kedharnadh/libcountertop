package apps

import (
	"strings"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type (
	Setting = setting.Setting[schema.Apps]
	Table   = setting.Table[schema.Apps]
)

const Group setting.Group = "Apps"

func NewTable() *Table {
	rows := []Setting{
		ids(Setting{Name: "order", Group: Group, ID: "app_order", Icon: "mdi:sort", NoEntity: true},
			func(c *schema.Apps) *[]string { return &c.Order }),
		ids(
			Setting{
				Name:     "dock",
				Group:    Group,
				ID:       "app_dock",
				Icon:     "mdi:dock-right",
				NoEntity: true,
			},
			func(c *schema.Apps) *[]string { return &c.Dock },
		),
	}
	return setting.NewTable("apps", []setting.Group{Group}, rows,
		setting.Messages{Text: say.T, Missing: say.Missing})
}

func ids(s Setting, field func(*schema.Apps) *[]string) Setting {
	s.Kind = setting.Text
	s.Read = func(c *schema.Apps) string { return strings.Join(*field(c), ",") }
	s.Write = func(c *schema.Apps, v string) error {
		var out []string
		for id := range strings.SplitSeq(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				out = append(out, id)
			}
		}
		*field(c) = out
		return nil
	}
	return s
}
