package settings

import (
	gogui "github.com/go-gui-org/go-gui/gui"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	zonepolicy "github.com/ygelfand/libcountertop/pkg/settings/timezone"
)

func (p *BasicPages) Poster() *shell.Page {
	return &shell.Page{
		Title: say.T("display.poster"),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := p.options.Poster()
			return []widget.Row{
					{Label: say.T("poster.show"), Kind: widget.Toggle, On: cfg.Enabled},
					{Label: say.T("poster.next"), Dim: !cfg.Enabled},
					{Label: say.T("poster.every"), Kind: widget.Chevron, Value: cfg.Every.Label(), Dim: !cfg.Enabled},
				}, []func(int){
					func(int) { p.options.PosterEnabled(!cfg.Enabled) },
					func(int) { p.options.PosterNext() },
					p.open(func() shell.View { return p.PosterEvery() }),
				}
		},
	}
}
func (p *BasicPages) PosterEvery() *shell.Page {
	return &shell.Page{
		Title: say.T("poster.every"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Poster().Every
			var rows []widget.Row
			var acts []func(int)
			for _, e := range config.PosterEveries() {
				rows = append(rows, widget.Row{Label: e.Label(), Chosen: e == now})
				acts = append(acts, func(int) { p.options.SetPosterEvery(e) })
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
func (p *BasicPages) Dock() *shell.Page {
	return &shell.Page{
		Title: say.T("dock.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read().Screen.Drawer

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, e := range config.Edges() {
				rows = append(rows, widget.Row{Label: e.Label(), Chosen: e == now})
				acts = append(acts, func(int) { p.options.SetDock(e) })
			}
			return rows, acts
		},
	}
}
func (p *BasicPages) VolumeEdge() *shell.Page {
	return &shell.Page{
		Title: say.T("volume.location.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read().Screen.Volume

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, e := range config.Sides() {
				rows = append(rows, widget.Row{Label: e.Label(), Chosen: e == now})
				acts = append(acts, func(int) { p.options.SetVolumeEdge(e) })
			}
			return rows, acts
		},
	}
}
func (p *BasicPages) System() *shell.Page {
	return &shell.Page{Title: say.T("system.title"), Build: func() ([]widget.Row, []func(int)) {
		c := p.options.Read()
		return []widget.Row{{Glyph: gogui.IconTextSize, Label: say.T("settings.language"), Kind: widget.Chevron, Value: say.Name(say.Chosen())}, {Glyph: gogui.IconClock, Label: say.T("datetime.title"), Kind: widget.Chevron, Value: zonepolicy.Label(p.options.Zone())}, {Glyph: gogui.IconKeyboard, Label: say.T("keyboard.size"), Kind: widget.Chevron, Value: c.Screen.Keyboard.Label()}, {Glyph: gogui.IconSync, Label: say.T("settings.power"), Kind: widget.Chevron}}, []func(int){p.open(func() shell.View { return p.Language() }), p.open(func() shell.View { return p.DateTime() }), p.open(func() shell.View { return p.Keyboard() }), p.open(p.options.PowerPage)}
	}}
}
func (p *BasicPages) DateTime() *shell.Page {
	return &shell.Page{Title: say.T("datetime.title"), Build: func() ([]widget.Row, []func(int)) {
		c := p.options.Read()
		return []widget.Row{{Label: say.T("clock.zone"), Kind: widget.Chevron, Value: zonepolicy.Label(p.options.Zone())}, {Label: say.T("clock.hours"), Kind: widget.Toggle, On: c.Screen.Hours == config.TwentyFourHour}}, []func(int){p.open(func() shell.View { return p.Zone() }), func(int) { p.options.ToggleHours(c.Screen.Hours) }}
	}}
}
func (p *BasicPages) Keyboard() *shell.Page {
	return &shell.Page{Title: say.T("keyboard.size"), Build: func() ([]widget.Row, []func(int)) {
		now := p.options.Read().Screen.Keyboard
		rows := []widget.Row{}
		acts := []func(int){}
		for _, k := range config.KeyboardSizes() {
			rows = append(rows, widget.Row{Label: k.Label(), Chosen: k == now})
			acts = append(acts, func(int) { p.options.SetKeyboard(k) })
		}
		return rows, acts
	}}
}
