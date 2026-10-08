package settings

import (
	"image"
	"slices"
	"time"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
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
type Placement interface {
	SetPosition(config.Position)
	SetAlign(config.Align)
	SetSize(config.Size)
}
type IdleSlots interface {
	SetKind(int, string)
	SetSource(int, config.Source)
}
type IdleClock interface {
	SetAfter(config.Delay)
	SetFace(config.Face)
}
type IdleActions interface {
	IdleClock
	IdleSlots
	Placement
}
type WeatherCatalog interface {
	Fetch()
	Offered() ([]ha.Entity, error, bool)
}
type WeatherEntity interface {
	Title(string) string
	SetEntity(string)
}
type WeatherPlaces interface {
	SetDashboard(bool)
	SetIdle(bool)
}
type WeatherRendering interface {
	SetAnimate(bool)
	SetThemed(bool)
	SetLook(config.WeatherLook)
}
type WeatherActions interface {
	WeatherCatalog
	WeatherEntity
	WeatherPlaces
	WeatherRendering
	Placement
}
type MediaIdle interface{ SetIdle(config.Delay) }
type VisualLabel interface{ SetLabel(string) }
type IdleDependencies struct {
	Preferences Preferences
	Shell       *shell.Shell
	Idle        IdleActions
	Weather     WeatherActions
	Media       MediaIdle
	Visuals     VisualLabel
	Thumbnail   func(visual.Kind, int, int) *image.RGBA
}
type IdlePages struct{ options IdleDependencies }

func NewIdlePages(o IdleDependencies) *IdlePages { return &IdlePages{options: o} }

func (p *IdlePages) open(
	page shell.View,
) func(int) {
	return func(int) { p.options.Shell.Push(page) }
}
func (p *IdlePages) makePage(page func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(page()) }
}

func (p *IdlePages) visualPicker(
	title, empty string,
	none func(),
	chosen func() string,
	pick func(visual.Kind),
) *shell.Page {
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
		Build: func() ([]menu.Row, []func(int)) {
			c := p.options.Preferences.IdleState().Idle
			i := p.options.Idle
			return []menu.Row{
					{Label: say.T("idle.after"), Kind: menu.Chevron, Value: c.After.Label()},
					{
						Label: say.T("idle.media"),
						Hint:  say.T("idle.media.hint"),
						Kind:  menu.Slider,
						Level: MediaLevel(c.Media),
						Snap:  MediaSnap,
						Value: c.Media.Label(),
					},
					{
						Label: say.T("idle.visual1"),
						Kind:  menu.Chevron,
						Value: IdleKindLabel(c.First.Kind),
					},
					{
						Label: say.T("idle.source1"),
						Kind:  menu.Chevron,
						Value: c.First.Source.Label(),
					},
					{
						Label: say.T("idle.visual2"),
						Kind:  menu.Chevron,
						Value: IdleKindLabel(c.Second.Kind),
					},
					{
						Label: say.T("idle.source2"),
						Kind:  menu.Chevron,
						Value: c.Second.Source.Label(),
					},
					{Label: say.T("idle.face"), Kind: menu.Chevron, Value: c.Face.Label()},
					{
						Label: say.T("idle.vertical"),
						Kind:  menu.Chevron,
						Value: c.Position.Label(),
					},
					{Label: say.T("idle.horizontal"), Kind: menu.Chevron, Value: c.Align.Label()},
					{Label: say.T("idle.size"), Kind: menu.Chevron, Value: c.Size.Label()},
					{
						Label: say.T("idle.label"),
						Kind:  menu.Field,
						Value: p.options.Preferences.IdleState().Visual.Label,
						Save:  p.options.Visuals.SetLabel,
					},
				}, []func(int){
					p.open(Choose(
						say.T("idle.after.title"),
						config.Delays(),
						func() config.Delay { return p.options.Preferences.IdleState().Idle.After },
						i.SetAfter,
						nil,
					)),
					func(level int) { p.options.Media.SetIdle(config.MediaDelays()[MediaIndex(level)]) },
					p.makePage(func() shell.View { return p.Visual(0) }),
					p.makePage(func() shell.View { return p.Source(0) }),
					p.makePage(func() shell.View { return p.Visual(1) }),
					p.makePage(func() shell.View { return p.Source(1) }),
					p.open(Choose(
						say.T("idle.face"),
						config.IdleFaces(),
						func() config.Face { return p.options.Preferences.IdleState().Idle.Face },
						i.SetFace,
						p.FacePreview,
					)),
					p.open(Choose(
						say.T("idle.vertical"),
						config.IdlePositions(),
						func() config.Position { return p.options.Preferences.IdleState().Idle.Position },
						i.SetPosition,
						func(position config.Position) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Position = position })
						},
					)),
					p.open(Choose(
						say.T("idle.horizontal"),
						config.IdleAligns(),
						func() config.Align { return p.options.Preferences.IdleState().Idle.Align },
						i.SetAlign,
						func(a config.Align) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Align = a })
						},
					)),
					p.open(Choose(
						say.T("idle.size"),
						config.Sizes(),
						func() config.Size { return p.options.Preferences.IdleState().Idle.Size },
						i.SetSize,
						func(s config.Size) func(ui.Surface, ui.Rect, theme.Theme) {
							return p.Placed(func(c *config.Idle) { c.Size = s })
						},
					)),
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
	c := p.options.Preferences.IdleState().Idle
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
		cfg := p.options.Preferences.IdleState()
		r := face.Read(time.Now(), cfg.Screen.Hours == config.TwentyFourHour)
		face.Of(f).Draw(s, at, r.Undated(), cfg.Clock.Ink.Over(palette))
	}
}

func (p *IdlePages) Placed(change func(*config.Idle)) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, box ui.Rect, palette theme.Theme) {
		cfg := p.options.Preferences.IdleState()
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
		Build: func() ([]menu.Row, []func(int)) {
			c := p.options.Preferences.IdleState().Weather
			return []menu.Row{
					{
						Label: say.T("weather.entity"),
						Kind:  menu.Chevron,
						Value: p.options.Weather.Title(c.Entity),
					},
					{Label: say.T("weather.dashboard"), Kind: menu.Toggle, On: c.Dashboard},
					{Label: say.T("weather.idle"), Kind: menu.Toggle, On: c.Idle},
					{Label: say.T("weather.look"), Kind: menu.Chevron, Value: c.Look.Label()},
					{Label: say.T("weather.animate"), Kind: menu.Toggle, On: c.Animate},
					{
						Label: say.T("weather.themed"),
						Hint:  say.T("weather.themed.hint"),
						Kind:  menu.Toggle,
						On:    c.Themed,
					},
					{
						Label: say.T("idle.vertical"),
						Kind:  menu.Chevron,
						Value: c.Position.Label(),
					},
					{Label: say.T("idle.horizontal"), Kind: menu.Chevron, Value: c.Align.Label()},
					{Label: say.T("weather.size"), Kind: menu.Chevron, Value: c.Size.Label()},
				}, []func(int){
					func(int) {
						p.options.Weather.Fetch()
						p.options.Shell.Push(p.WeatherEntities())
					},
					func(int) { p.options.Weather.SetDashboard(!c.Dashboard) },
					func(int) { p.options.Weather.SetIdle(!c.Idle) },
					p.makePage(func() shell.View { return p.WeatherLooks() }),
					func(int) { p.options.Weather.SetAnimate(!c.Animate) },
					func(int) { p.options.Weather.SetThemed(!c.Themed) },
					p.open(Choose(
						say.T("idle.vertical"),
						config.Positions(),
						func() config.Position { return p.options.Preferences.IdleState().Weather.Position },
						p.options.Weather.SetPosition,
						nil,
					)),
					p.open(Choose(
						say.T("idle.horizontal"),
						config.Aligns(),
						func() config.Align { return p.options.Preferences.IdleState().Weather.Align },
						p.options.Weather.SetAlign,
						nil,
					)),
					p.open(Choose(
						say.T("weather.size"),
						config.Sizes(),
						func() config.Size { return p.options.Preferences.IdleState().Weather.Size },
						p.options.Weather.SetSize,
						nil,
					)),
				}
		},
	}
}

func (p *IdlePages) WeatherEntities() *shell.Page {
	return &shell.Page{
		Title: say.T("weather.entity"),
		Build: func() ([]menu.Row, []func(int)) {
			list, err, fetched := p.options.Weather.Offered()
			switch {
			case !fetched:
				return []menu.Row{{Label: say.T("home.loading")}}, []func(int){nil}
			case err != nil:
				return []menu.Row{
						{Label: say.T("home.failed"), Hint: err.Error()},
						{Label: say.T("home.retry")},
					},
					[]func(int){
						nil,
						func(int) { p.options.Weather.Fetch() },
					}
			case len(list) == 0:
				return []menu.Row{{Label: say.T("weather.none")}}, []func(int){nil}
			}
			now := p.options.Preferences.IdleState().Weather.Entity
			rows := make([]menu.Row, 0, len(list))
			taps := make([]func(int), 0, len(list))
			for _, e := range list {
				rows = append(rows, menu.Row{Label: e.Name, Hint: e.ID, Chosen: e.ID == now})
				taps = append(taps, func(int) { p.options.Weather.SetEntity(e.ID) })
			}
			return rows, taps
		},
	}
}

func (p *IdlePages) WeatherLooks() *shell.Page {
	return &shell.Page{
		Title: say.T("weather.look"),
		Tiles: func() ([]menu.Cell, []func(int)) {
			now := p.options.Preferences.IdleState().Weather.Look
			var cells []menu.Cell
			var taps []func(int)
			for _, l := range config.WeatherLooks() {
				cells = append(cells, menu.Cell{Label: l.Label(), Chosen: l == now, Weather: l})
				taps = append(taps, func(int) { p.options.Weather.SetLook(l) })
			}
			return cells, taps
		},
	}
}
