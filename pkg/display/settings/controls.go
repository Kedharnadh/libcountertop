package settings

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	zonepolicy "github.com/ygelfand/libcountertop/pkg/settings/timezone"
)

func (p *BasicPages) Poster() *shell.Page {
	return &shell.Page{
		Title: say.T("display.poster"),
		Build: func() ([]menu.Row, []func(int)) {
			cfg := p.options.Preferences.Poster.Read()
			return []menu.Row{
					{Label: say.T("poster.show"), Kind: menu.Toggle, On: cfg.Enabled},
					{Label: say.T("poster.next"), Dim: !cfg.Enabled},
					{
						Label: say.T("poster.every"),
						Kind:  menu.Chevron,
						Value: cfg.Every.Label(),
						Dim:   !cfg.Enabled,
					},
				}, []func(int){
					func(int) { p.options.Poster.SetEnabled(!cfg.Enabled) },
					func(int) { p.options.Poster.Next() },
					p.open(func() shell.View { return p.PosterEvery() }),
				}
		},
	}
}
func (p *BasicPages) PosterEvery() *shell.Page {
	return &shell.Page{
		Title: say.T("poster.every"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Preferences.Poster.Read().Every
			var rows []menu.Row
			var acts []func(int)
			for _, e := range config.PosterEveries() {
				rows = append(rows, menu.Row{Label: e.Label(), Chosen: e == now})
				acts = append(acts, func(int) { p.options.Poster.SetEvery(e) })
			}
			return rows, acts
		},
	}
}
func PosterSays(p config.Poster) string {
	if !p.Enabled {
		return say.T("poster.off")
	}
	return p.Every.Label()
}
func (p *BasicPages) System() *shell.Page {
	return &shell.Page{Title: say.T("system.title"), Build: func() ([]menu.Row, []func(int)) {
		c := p.state()
		return []menu.Row{
				{
					Glyph: gogui.IconTextSize,
					Label: say.T("settings.language"),
					Kind:  menu.Chevron,
					Value: say.Name(say.Chosen()),
				},
				{
					Glyph: gogui.IconClock,
					Label: say.T("datetime.title"),
					Kind:  menu.Chevron,
					Value: zonepolicy.Label(p.options.Zone.ChosenZone()),
				},
				{
					Glyph: gogui.IconKeyboard,
					Label: say.T("keyboard.size"),
					Kind:  menu.Chevron,
					Value: c.Screen.Keyboard.Label(),
				},
				{Glyph: gogui.IconSync, Label: say.T("settings.power"), Kind: menu.Chevron},
			}, []func(int){
				p.open(func() shell.View { return p.Language() }),
				p.open(func() shell.View { return p.DateTime() }),
				p.open(func() shell.View { return p.Keyboard() }),
				p.open(p.options.PowerPage),
			}
	}}
}
func (p *BasicPages) DateTime() *shell.Page {
	return &shell.Page{Title: say.T("datetime.title"), Build: func() ([]menu.Row, []func(int)) {
		c := p.state()
		return []menu.Row{
				{
					Label: say.T("clock.zone"),
					Kind:  menu.Chevron,
					Value: zonepolicy.Label(p.options.Zone.ChosenZone()),
				},
				{
					Label: say.T("clock.hours"),
					Kind:  menu.Toggle,
					On:    c.Screen.Hours == config.TwentyFourHour,
				},
			}, []func(int){
				p.open(func() shell.View { return p.Zone() }),
				func(int) { p.toggleHours(c.Screen.Hours) },
			}
	}}
}
func (p *BasicPages) Keyboard() *shell.Page {
	return &shell.Page{Title: say.T("keyboard.size"), Build: func() ([]menu.Row, []func(int)) {
		now := p.state().Screen.Keyboard
		rows := []menu.Row{}
		acts := []func(int){}
		for _, k := range config.KeyboardSizes() {
			rows = append(rows, menu.Row{Label: k.Label(), Chosen: k == now})
			acts = append(acts, func(int) { p.setKeyboard(k) })
		}
		return rows, acts
	}}
}
