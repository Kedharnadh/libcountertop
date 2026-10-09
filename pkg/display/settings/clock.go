package settings

import (
	"time"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/previews"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type ClockState struct {
	Screen config.Screen
	Clock  config.Clock
}
type ClockController interface {
	Placement
	SetFace(config.Face)
	SetInk(config.Ink)
	SetDate(bool)
}
type ClockDependencies struct {
	Preferences Preferences
	Shell       *shell.Shell
	Controller  ClockController
	Previews    previews.Kit
}
type ClockPages struct{ options ClockDependencies }

func NewClockPages(options ClockDependencies) *ClockPages { return &ClockPages{options: options} }

func (p *ClockPages) open(
	view shell.View,
) func(int) {
	return func(int) { p.options.Shell.Push(view) }
}
func (p *ClockPages) Page() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.title"),
		Build: func() ([]menu.Row, []func(int)) {
			cfg := p.options.Preferences.ClockState()

			rows := []menu.Row{
				{
					Label: say.T("clock.face"),
					Kind:  menu.Chevron,
					Value: cfg.Clock.Face.Label(),
				},
			}
			taps := []func(int){
				p.open(p.Faces(say.T("clock.face"),
					func() config.Face { return p.options.Preferences.ClockState().Clock.Face },
					p.options.Controller.SetFace)),
			}
			placed, placing := Placing{
				Positions: config.Positions(),
				Aligns:    config.Aligns(),
				Read: func() config.Placement {
					return p.options.Preferences.ClockState().Clock.Placement
				},
				Set:     p.options.Controller,
				Preview: p.placed,
			}.Rows(p.options.Shell)
			rows = append(append(rows, placed...),
				menu.Row{
					Label: say.T("clock.color"),
					Kind:  menu.Chevron,
					Value: cfg.Clock.Ink.Label(),
				},
				menu.Row{Label: say.T("clock.date"), Kind: menu.Toggle, On: cfg.Clock.Date},
			)
			taps = append(append(taps, placing...),
				p.open(p.Colors()),
				func(int) { p.options.Controller.SetDate(!cfg.Clock.Date) },
			)
			return rows, taps
		},
	}
}

func (p *ClockPages) Faces(
	title string,
	now func() config.Face,
	use func(config.Face),
) *shell.Page {
	return &shell.Page{
		Title: title,
		Tiles: func() ([]menu.Cell, []func(int)) {
			chosen := now()

			cells := make([]menu.Cell, 0, len(config.Faces()))
			acts := make([]func(int), 0, len(config.Faces()))

			for _, f := range config.Faces() {
				cells = append(cells, menu.Cell{
					Label:   f.Label(),
					Chosen:  f == chosen,
					Preview: p.options.Previews.ClockFace(f),
				})
				acts = append(acts, picks(use, f))
			}
			return cells, acts
		},
	}
}

func picks(use func(config.Face), f config.Face) func(int) {
	return func(int) { use(f) }
}

func (p *ClockPages) Colors() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.color.title"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Preferences.ClockState().Clock.Ink

			rows := make([]menu.Row, 0, len(config.Inks()))
			acts := make([]func(int), 0, len(config.Inks()))

			for _, ink := range config.Inks() {
				rows = append(rows, menu.Row{
					Label:   ink.Label(),
					Chosen:  ink == now,
					Preview: p.inking(ink),
				})
				acts = append(acts, p.useInk(ink))
			}
			return rows, acts
		},
	}
}

func (p *ClockPages) useInk(ink config.Ink) func(int) {
	return func(int) { p.options.Controller.SetInk(ink) }
}

func (p *ClockPages) placed(at config.Placement) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		cfg := p.options.Preferences.ClockState()
		reading := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)

		within := dashboard.Place(at.Position, at.Align, at.Size, box.W, box.H)
		within.X += box.X
		within.Y += box.Y

		face.Of(cfg.Clock.Face).Draw(s, within, reading.Undated(), cfg.Clock.Ink.Over(palette))
	}
}

func (p *ClockPages) inking(ink config.Ink) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		cfg := p.options.Preferences.ClockState()
		reading := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)

		face.Of(cfg.Clock.Face).Draw(s, at, reading.Undated(), ink.Over(palette))
	}
}
