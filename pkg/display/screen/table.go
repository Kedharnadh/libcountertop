package screen

import (
	"slices"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type (
	Setting = setting.Setting[schema.Screen]
	Table   = setting.Table[schema.Screen]
)

const Panel setting.Group = "Panel"

func NewTable(uiSize func() string) *Table {
	var t *Table
	t = setting.NewTable(
		"screen",
		[]setting.Group{Panel},
		rows(func() *Table { return t }, uiSize),
		setting.Messages{Text: say.T, Missing: say.Missing},
	)
	return t
}

func rows(table func() *Table, uiSize func() string) []Setting {
	return []Setting{
		{
			Name:  "backlight",
			Group: Panel,
			ID:    "backlight", Icon: "mdi:brightness-6",
			Kind: setting.Number, Min: 0, Max: 100, Unit: "%", Slider: true,
			Read: func(s *schema.Screen) string { return strconv.Itoa(s.Backlight) },
			Write: func(s *schema.Screen, v string) error {
				n, err := table().MustRow("backlight").Number(v)
				if err != nil {
					return err
				}
				s.Backlight = n
				return nil
			},
		},
		{
			Name:  "auto",
			Group: Panel,
			ID:    "screen_mode",
			Icon:  "mdi:brightness-auto",
			Kind:  setting.Toggle,
			Read: func(s *schema.Screen) string {
				return setting.OnOff(s.Mode == schema.ModeAuto)
			},
			Write: func(s *schema.Screen, v string) error {
				on, ok := setting.Boolean(v)
				if !ok {
					return table().MustRow("auto").Bad(v, "on or off")
				}
				s.Mode = schema.ModeManual
				if on {
					s.Mode = schema.ModeAuto
				}
				return nil
			},
		},
		{
			Name:  "style",
			Group: Panel,
			ID:    "screen_style", Icon: "mdi:shape-outline",
			Kind: setting.Choice, Options: plain(style.Names()),
			Read: func(s *schema.Screen) string {
				if s.Style == "" {
					return style.Standard
				}
				return s.Style
			},
			Write: func(s *schema.Screen, v string) error {
				if !slices.Contains(style.Names(), v) {
					return table().MustRow("style").Bad(v, "")
				}
				s.Style = v
				return nil
			},
		},
		{
			Name:  "size",
			Group: Panel,
			ID:    "screen_size", Icon: "mdi:resize",
			Kind: setting.Choice, Options: plain(schema.ScreenSizes()),
			Read: func(s *schema.Screen) string {
				if s.Size == "" {
					return uiSize()
				}
				return s.Size
			},
			Write: func(s *schema.Screen, v string) error {
				if !slices.Contains(schema.ScreenSizes(), v) {
					return table().MustRow("size").Bad(v, "")
				}
				s.Size = v
				return nil
			},
		},
		{
			Name:  "theme",
			Group: Panel,
			ID:    "theme", Icon: "mdi:palette",
			Kind: setting.Choice, Options: themes(),
			Read: func(s *schema.Screen) string { return s.Theme },
			Write: func(s *schema.Screen, v string) error {
				if _, ok := theme.ByName(v); !ok && v != style.ThemeDefault {
					return table().MustRow("theme").Bad(v, "")
				}
				s.Theme = v
				return nil
			},
		},
		{
			Name:  "drawer",
			Group: Panel,
			ID:    "drawer_edge", Icon: "mdi:gesture-swipe-horizontal",
			Kind: setting.Choice, Options: edges(),
			Read: func(s *schema.Screen) string { return string(s.Drawer) },
			Write: func(s *schema.Screen, v string) error {
				for _, e := range schema.Edges() {
					if string(e) == v {
						s.Drawer = e
						return nil
					}
				}
				return table().MustRow("drawer").Bad(v, "")
			},
		},
	}
}

func plain(values []string) []setting.Option {
	out := make([]setting.Option, 0, len(values))
	for _, v := range values {
		out = append(out, setting.Option{Value: v, Label: v})
	}
	return out
}

func themes() []setting.Option {
	out := []setting.Option{{Value: style.ThemeDefault, Label: style.ThemeDefault}}
	for _, name := range theme.Names() {
		out = append(out, setting.Option{Value: name, Label: name})
	}
	return out
}

func edges() []setting.Option {
	out := make([]setting.Option, 0, len(schema.Edges()))
	for _, e := range schema.Edges() {
		out = append(out, setting.Option{Value: string(e), Label: e.Label()})
	}
	return out
}
