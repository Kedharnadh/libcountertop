package poster

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type (
	Setting  = setting.Setting[config.Poster]
	Table    = setting.Table[config.Poster]
	Controls = setting.Controls[config.Poster]
)

const Group setting.Group = "Poster"

func NewTable() *Table {
	var t *Table
	t = setting.NewTable(
		"poster",
		[]setting.Group{Group},
		rows(func() *Table { return t }),
		setting.Messages{Text: say.T, Missing: say.Missing},
	)
	return t
}

func rows(table func() *Table) []Setting {
	text := func(name, icon string, kind setting.Kind, field func(*config.Poster) *string) Setting {
		return Setting{
			Name: name, Group: Group, Icon: icon, Kind: kind,
			Read:  func(c *config.Poster) string { return *field(c) },
			Write: func(c *config.Poster, v string) error { *field(c) = v; return nil },
		}
	}
	return []Setting{
		{
			Name: "on", Group: Group, ID: "poster", Icon: "mdi:image-frame",
			Kind: setting.Toggle,
			Read: func(c *config.Poster) string { return setting.OnOff(c.Enabled) },
			Write: func(c *config.Poster, v string) error {
				on, ok := setting.Boolean(v)
				if !ok {
					return table().MustRow("on").Bad(v, "on or off")
				}
				c.Enabled = on
				return nil
			},
		},
		{
			Name: "every", Group: Group, Icon: "mdi:image-sync",
			Kind: setting.Choice, Options: everies(),
			Read: func(c *config.Poster) string { return string(c.Every) },
			Write: func(c *config.Poster, v string) error {
				e := config.PosterEvery(v)
				if !slices.Contains(config.PosterEveries(), e) {
					return table().MustRow("every").Bad(v, "")
				}
				c.Every = e
				return nil
			},
		},
		text("server", "mdi:server", setting.Text,
			func(c *config.Poster) *string { return &c.Server }),
		text("key", "mdi:key", setting.Secret,
			func(c *config.Poster) *string { return &c.Key }),
		text("albums", "mdi:image-album", setting.Text,
			func(c *config.Poster) *string { return &c.Albums }),
		text("tags", "mdi:tag-multiple", setting.Text,
			func(c *config.Poster) *string { return &c.Tags }),
	}
}

func everies() []setting.Option {
	all := config.PosterEveries()
	out := make([]setting.Option, len(all))
	for i, e := range all {
		out[i] = setting.Option{Value: string(e), Key: string(e)}
	}
	return out
}
