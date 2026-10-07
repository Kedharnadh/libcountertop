package settings

import (
	"image"
	"slices"
	"time"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	ha "github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type IdleState struct {
	Screen  config.Screen
	Clock   config.Clock
	Idle    config.Idle
	Visual  config.Visual
	Weather config.Weather
}
type IdleActions interface {
	SetAfter(config.Delay)
	SetKind(int, string)
	SetSource(int, config.Source)
	SetFace(config.Face)
	SetPosition(config.Position)
	SetAlign(config.Align)
	SetSize(config.Size)
}
type WeatherActions struct {
	Title                                        func(string) string
	Fetch                                        func()
	Offered                                      func() ([]ha.Entity, error, bool)
	SetDashboard, SetIdle, SetAnimate, SetThemed func(bool)
	SetPosition                                  func(config.Position)
	SetAlign                                     func(config.Align)
	SetSize                                      func(config.Size)
	SetEntity                                    func(string)
	SetLook                                      func(config.WeatherLook)
}
type IdleOptions struct {
	Read      func() IdleState
	Push      func(shell.View)
	Idle      IdleActions
	Weather   WeatherActions
	MediaIdle func(config.Delay)
	Label     func(string)
	Thumbnail func(visual.Kind, int, int) *image.RGBA
}
type IdlePages struct{ options IdleOptions }

func NewIdlePages(o IdleOptions) *IdlePages         { return &IdlePages{options: o} }
func (p *IdlePages) open(page shell.View) func(int) { return func(int) { p.options.Push(page) } }
func (p *IdlePages) makePage(page func() shell.View) func(int) {
	return func(int) { p.options.Push(page()) }
}
func (p *IdlePages) visualPicker(title, empty string, none func(), chosen func() string, pick func(visual.Kind)) *shell.Page {
	return VisualPicker(title, empty, none, chosen, pick, p.options.Thumbnail)
}
func IdleKindLabel(kind string) string {
	if kind == "" {
		return say.T("idle.none")
	}
	return visual.Kind(kind).Label()
}
func (p *IdlePages) Page() *shell.Page {
	return &shell.Page{
		Title: say.T("idle.title"),
		Build: func() ([]widget.Row, []func(int)) {
			c := p.options.Read().Idle
			i := p.options.Idle
			return []widget.Row{
					{Label: say.T("idle.after"), Kind: widget.Chevron, Value: c.After.Label()},
					{Label: say.T("idle.media"), Hint: say.T("idle.media.hint"), Kind: widget.Slider, Level: MediaLevel(c.Media), Snap: MediaSnap, Value: c.Media.Label()},
					{Label: say.T("idle.visual1"), Kind: widget.Chevron, Value: IdleKindLabel(c.First.Kind)},
					{Label: say.T("idle.source1"), Kind: widget.Chevron, Value: c.First.Source.Label()},
					{Label: say.T("idle.visual2"), Kind: widget.Chevron, Value: IdleKindLabel(c.Second.Kind)},
					{Label: say.T("idle.source2"), Kind: widget.Chevron, Value: c.Second.Source.Label()},
					{Label: say.T("idle.face"), Kind: widget.Chevron, Value: c.Face.Label()},
					{Label: say.T("idle.vertical"), Kind: widget.Chevron, Value: c.Position.Label()},
					{Label: say.T("idle.horizontal"), Kind: widget.Chevron, Value: c.Align.Label()},
					{Label: say.T("idle.size"), Kind: widget.Chevron, Value: c.Size.Label()},
					{Label: say.T("idle.label"), Kind: widget.Field, Value: p.options.Read().Visual.Label, Save: p.options.Label},
				}, []func(int){
					p.open(Choose(say.T("idle.after.title"), config.Delays(),
						func() config.Delay { return p.options.Read().Idle.After }, i.SetAfter, nil)),
					func(level int) { p.options.MediaIdle(config.MediaDelays()[MediaIndex(level)]) },
					p.makePage(func() shell.View { return p.Visual(0) }),
					p.makePage(func() shell.View { return p.Source(0) }),
					p.makePage(func() shell.View { return p.Visual(1) }),
					p.makePage(func() shell.View { return p.Source(1) }),
					p.open(Choose(say.T("idle.face"), config.IdleFaces(),
						func() config.Face { return p.options.Read().Idle.Face }, i.SetFace, p.FacePreview)),
					p.open(Choose(say.T("idle.vertical"), config.Positions(),
						func() config.Position { return p.options.Read().Idle.Position }, i.SetPosition,
						func(position config.Position) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Position = position })
						})),
					p.open(Choose(say.T("idle.horizontal"), config.Aligns(),
						func() config.Align { return p.options.Read().Idle.Align }, i.SetAlign,
						func(a config.Align) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Align = a })
						})),
					p.open(Choose(say.T("idle.size"), config.Sizes(),
						func() config.Size { return p.options.Read().Idle.Size }, i.SetSize,
						func(s config.Size) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Size = s })
						})),
					nil,
				}
		},
	}
}

func MediaIndex(level int) int {
	n := len(config.MediaDelays()) - 1
	return min(max((level*n+50)/100, 0), n)
}

func MediaLevel(d config.Delay) int {
	i := max(slices.Index(config.MediaDelays(), d), 0)
	return i * 100 / (len(config.MediaDelays()) - 1)
}

func MediaSnap(level int) int { return MediaLevel(config.MediaDelays()[MediaIndex(level)]) }

func (p *IdlePages) SlotVisual(slot int) config.IdleVisual {
	c := p.options.Read().Idle
	if slot == 0 {
		return c.First
	}
	return c.Second
}

func (p *IdlePages) Visual(slot int) *shell.Page {
	title := say.T("idle.visual1")
	if slot == 1 {
		title = say.T("idle.visual2")
	}
	return p.visualPicker(title, say.T("idle.none"),
		func() { p.options.Idle.SetKind(slot, "") },
		func() string { return p.SlotVisual(slot).Kind },
		func(k visual.Kind) { p.options.Idle.SetKind(slot, string(k)) })
}

func (p *IdlePages) Source(slot int) *shell.Page {
	title := say.T("idle.source1")
	if slot == 1 {
		title = say.T("idle.source2")
	}
	return Choose(title, config.Sources(),
		func() config.Source { return p.SlotVisual(slot).Source },
		func(s config.Source) { p.options.Idle.SetSource(slot, s) }, nil)
}

func (p *IdlePages) FacePreview(f config.Face) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		if f == config.FaceNone {
			return
		}
		cfg := p.options.Read()
		r := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)
		face.Of(f).Draw(s, at, r.Undated(), cfg.Clock.Ink.Over(palette))
	}
}

func (p *IdlePages) Placed(change func(*config.Idle)) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		cfg := p.options.Read()
		c := cfg.Idle
		change(&c)
		if c.Face == config.FaceNone {
			return
		}
		r := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)
		within := dashboard.Place(c.Position, c.Align, c.Size, box.W, box.H)
		within.X += box.X
		within.Y += box.Y
		face.Of(c.Face).Draw(s, within, r.Undated(), cfg.Clock.Ink.Over(palette))
	}
}

func (p *IdlePages) Weather() *shell.Page {
	return &shell.Page{
		Title: say.T("weather.title"),
		Build: func() ([]widget.Row, []func(int)) {
			c := p.options.Read().Weather
			return []widget.Row{
					{Label: say.T("weather.entity"), Kind: widget.Chevron, Value: p.options.Weather.Title(c.Entity)},
					{Label: say.T("weather.dashboard"), Kind: widget.Toggle, On: c.Dashboard},
					{Label: say.T("weather.idle"), Kind: widget.Toggle, On: c.Idle},
					{Label: say.T("weather.look"), Kind: widget.Chevron, Value: c.Look.Label()},
					{Label: say.T("weather.animate"), Kind: widget.Toggle, On: c.Animate},
					{Label: say.T("weather.themed"), Hint: say.T("weather.themed.hint"), Kind: widget.Toggle, On: c.Themed},
					{Label: say.T("idle.vertical"), Kind: widget.Chevron, Value: c.Position.Label()},
					{Label: say.T("idle.horizontal"), Kind: widget.Chevron, Value: c.Align.Label()},
					{Label: say.T("weather.size"), Kind: widget.Chevron, Value: c.Size.Label()},
				}, []func(int){
					func(int) {
						p.options.Weather.Fetch()
						p.options.Push(p.WeatherEntities())
					},
					func(int) { p.options.Weather.SetDashboard(!c.Dashboard) },
					func(int) { p.options.Weather.SetIdle(!c.Idle) },
					p.makePage(func() shell.View { return p.WeatherLooks() }),
					func(int) { p.options.Weather.SetAnimate(!c.Animate) },
					func(int) { p.options.Weather.SetThemed(!c.Themed) },
					p.open(Choose(say.T("idle.vertical"), config.Positions(),
						func() config.Position { return p.options.Read().Weather.Position }, p.options.Weather.SetPosition, nil)),
					p.open(Choose(say.T("idle.horizontal"), config.Aligns(),
						func() config.Align { return p.options.Read().Weather.Align }, p.options.Weather.SetAlign, nil)),
					p.open(Choose(say.T("weather.size"), config.Sizes(),
						func() config.Size { return p.options.Read().Weather.Size }, p.options.Weather.SetSize, nil)),
				}
		},
	}
}

func (p *IdlePages) WeatherEntities() *shell.Page {
	return &shell.Page{
		Title: say.T("weather.entity"),
		Build: func() ([]widget.Row, []func(int)) {
			list, err, fetched := p.options.Weather.Offered()
			switch {
			case !fetched:
				return []widget.Row{{Label: say.T("home.loading")}}, []func(int){nil}
			case err != nil:
				return []widget.Row{{Label: say.T("home.failed"), Hint: err.Error()}, {Label: say.T("home.retry")}},
					[]func(int){nil, func(int) { p.options.Weather.Fetch() }}
			case len(list) == 0:
				return []widget.Row{{Label: say.T("weather.none")}}, []func(int){nil}
			}
			now := p.options.Read().Weather.Entity
			rows := make([]widget.Row, 0, len(list))
			taps := make([]func(int), 0, len(list))
			for _, e := range list {
				rows = append(rows, widget.Row{Label: e.Name, Hint: e.ID, Chosen: e.ID == now})
				taps = append(taps, func(int) { p.options.Weather.SetEntity(e.ID) })
			}
			return rows, taps
		},
	}
}

func (p *IdlePages) WeatherLooks() *shell.Page {
	return &shell.Page{
		Title: say.T("weather.look"),
		Tiles: func() ([]widget.Cell, []func(int)) {
			now := p.options.Read().Weather.Look
			var cells []widget.Cell
			var taps []func(int)
			for _, l := range config.WeatherLooks() {
				cells = append(cells, widget.Cell{Label: l.Label(), Chosen: l == now, Weather: l})
				taps = append(taps, func(int) { p.options.Weather.SetLook(l) })
			}
			return cells, taps
		},
	}
}
