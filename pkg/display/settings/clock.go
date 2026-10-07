package settings

import (
	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"time"
)

type ClockState struct {
	Screen config.Screen
	Clock  config.Clock
}
type ClockOptions struct {
	Read     func() ClockState
	Push     func(shell.View)
	Face     func(config.Face)
	Position func(config.Position)
	Size     func(config.Size)
	Ink      func(config.Ink)
	Date     func(bool)
}
type ClockPages struct{ options ClockOptions }

func NewClockPages(options ClockOptions) *ClockPages { return &ClockPages{options: options} }
func (p *ClockPages) open(view shell.View) func(int) { return func(int) { p.options.Push(view) } }
func (p *ClockPages) Page() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.title"),
		Build: func() ([]widget.Row, []func(int)) {
			cfg := p.options.Read()

			return []widget.Row{
					{Label: say.T("clock.face"), Kind: widget.Chevron, Value: cfg.Clock.Face.Label()},
					{Label: say.T("clock.position"), Kind: widget.Chevron,
						Value: cfg.Clock.Position.Label()},
					{Label: say.T("clock.size"), Kind: widget.Chevron, Value: cfg.Clock.Size.Label()},
					{Label: say.T("clock.color"), Kind: widget.Chevron, Value: cfg.Clock.Ink.Label()},
					{Label: say.T("clock.date"), Kind: widget.Toggle, On: cfg.Clock.Date},
				}, []func(int){
					p.open(p.Faces(say.T("clock.face"),
						func() config.Face { return p.options.Read().Clock.Face },
						p.options.Face)),
					p.open(p.Positions()),
					p.open(p.Sizes()),
					p.open(p.Colors()),
					func(int) { p.options.Date(!cfg.Clock.Date) },
				}
		},
	}
}

func (p *ClockPages) Faces(title string, now func() config.Face, use func(config.Face)) *shell.Page {
	return &shell.Page{
		Title: title,
		Tiles: func() ([]widget.Cell, []func(int)) {
			chosen := now()

			cells := make([]widget.Cell, 0, len(config.Faces()))
			acts := make([]func(int), 0, len(config.Faces()))

			for _, f := range config.Faces() {
				cells = append(cells, widget.Cell{
					Label:  f.Label(),
					Chosen: f == chosen,
					Paint:  p.Preview(f),
					Face:   f,
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

func (p *ClockPages) Preview(f config.Face) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		cfg := p.options.Read()
		reading := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)

		// In the clock's own color, not the theme's. The row is previewing this device's clock, and
		// a list of gray thumbnails picking a shape for an amber clock is previewing a different one.
		face.Of(f).Draw(s, at, reading.Undated(), cfg.Clock.Ink.Over(palette))
	}
}

func (p *ClockPages) Positions() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.position.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read().Clock.Position

			rows := make([]widget.Row, 0, len(config.Positions()))
			acts := make([]func(int), 0, len(config.Positions()))

			for _, at := range config.Positions() {
				rows = append(rows, widget.Row{
					Label:   at.Label(),
					Chosen:  at == now,
					Preview: p.placing(at),
				})
				acts = append(acts, p.usePosition(at))
			}
			return rows, acts
		},
	}
}

func (p *ClockPages) usePosition(at config.Position) func(int) {
	return func(int) { p.options.Position(at) }
}

func (p *ClockPages) Sizes() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.size.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read().Clock.Size

			rows := make([]widget.Row, 0, len(config.Sizes()))
			acts := make([]func(int), 0, len(config.Sizes()))

			for _, size := range config.Sizes() {
				rows = append(rows, widget.Row{
					Label:   size.Label(),
					Chosen:  size == now,
					Preview: p.sizing(size),
				})
				acts = append(acts, p.useSize(size))
			}
			return rows, acts
		},
	}
}

func (p *ClockPages) useSize(size config.Size) func(int) {
	return func(int) { p.options.Size(size) }
}

func (p *ClockPages) Colors() *shell.Page {
	return &shell.Page{
		Title: say.T("clock.color.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read().Clock.Ink

			rows := make([]widget.Row, 0, len(config.Inks()))
			acts := make([]func(int), 0, len(config.Inks()))

			for _, ink := range config.Inks() {
				rows = append(rows, widget.Row{
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
	return func(int) { p.options.Ink(ink) }
}

func (p *ClockPages) placing(at config.Position) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		p.laid(s, box, palette, at, p.options.Read().Clock.Size)
	}
}

func (p *ClockPages) sizing(size config.Size) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		p.laid(s, box, palette, p.options.Read().Clock.Position, size)
	}
}

func (p *ClockPages) laid(s ui.Surface, box ui.Rect, palette theme.Theme, at config.Position, size config.Size) {
	cfg := p.options.Read()
	reading := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)

	within := dashboard.Box(at, size, box.W, box.H)
	within.X += box.X
	within.Y += box.Y

	face.Of(cfg.Clock.Face).Draw(s, within, reading.Undated(), cfg.Clock.Ink.Over(palette))
}

func (p *ClockPages) inking(ink config.Ink) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		cfg := p.options.Read()
		reading := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)

		face.Of(cfg.Clock.Face).Draw(s, at, reading.Undated(), ink.Over(palette))
	}
}
