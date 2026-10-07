package settings

import (
	"log/slog"
	"slices"

	gogui "github.com/go-gui-org/go-gui/gui"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	tz "github.com/ygelfand/libcountertop/pkg/timezone"
)

type BasicState struct {
	Screen              config.Screen
	Clock               config.Clock
	Idle                config.Idle
	Visual              config.Visual
	PosterLabel, UISize string
}
type BasicOptions struct {
	Poster                 func() config.Poster
	PosterEnabled          func(bool)
	PosterNext             func()
	SetPosterEvery         func(config.PosterEvery)
	SetDock, SetVolumeEdge func(config.Edge)
	SetKeyboard            func(config.KeyboardSize)
	ToggleHours            func(config.HourFormat)
	PowerPage              func() shell.View

	Read                                                                            func() BasicState
	Push                                                                            func(shell.View)
	ADB                                                                             func() bool
	SetADB                                                                          func(bool)
	SetFPS, SetSeed                                                                 func(int)
	SetStyle                                                                        func(string) error
	SetTheme, SetLanguage                                                           func(string)
	Zone                                                                            func() string
	SetZone                                                                         func(string)
	VisualPage, ClockPage, IdlePage, PosterPage, DockPage, VolumePage, PresencePage func() shell.View
	Brightness                                                                      func(int)
	Auto                                                                            func(bool)
	Size                                                                            func(string)
	Marks                                                                           func(bool)
}
type BasicPages struct{ options BasicOptions }

func NewBasicPages(o BasicOptions) *BasicPages { return &BasicPages{options: o} }
func (p *BasicPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Push(makePage()) }
}
func (p *BasicPages) useTheme(name string) func(int) { return func(int) { p.options.SetTheme(name) } }
func (p *BasicPages) useZone(name string) func(int)  { return func(int) { p.options.SetZone(name) } }

const (
	swatchPad = 0.12
	swatchGap = 0.07
)

func (p *BasicPages) Debug() *shell.Page {
	return &shell.Page{
		Title: say.T("debug.title"),
		Build: func() ([]widget.Row, []func(int)) {
			fps, seed := p.options.Read().Visual.MaxFPS, p.options.Read().Visual.Seed
			adb := p.options.ADB()
			return []widget.Row{
					{Glyph: gogui.IconTerminal, Label: say.T("system.adb"), Hint: say.T("system.adb.hint"), Kind: widget.Toggle, On: adb},
					{Label: say.T("debug.visuals"), Kind: widget.Chevron},
					{Label: say.T("debug.fps"), Kind: widget.Slider, Level: FpsLevel(fps), Snap: FpsSnap, Value: FpsSays(fps)},
					{Label: say.T("debug.seed"), Kind: widget.Slider, Level: SeedLevel(seed), Value: SeedSays(seed)},
				}, []func(int){
					func(int) { p.options.SetADB(!adb) },
					p.open(p.options.VisualPage),
					func(level int) { p.options.SetFPS(config.MaxFPSSteps[FpsIndex(level)]) },
					func(level int) { p.options.SetSeed(SeedOf(level)) },
				}
		},
	}
}

func FpsIndex(level int) int {
	n := len(config.MaxFPSSteps) - 1
	return min(max((level*n+50)/100, 0), n)
}

func FpsLevel(fps int) int {
	i := max(slices.Index(config.MaxFPSSteps, fps), 0)
	return i * 100 / (len(config.MaxFPSSteps) - 1)
}

func FpsSnap(level int) int { return FpsLevel(config.MaxFPSSteps[FpsIndex(level)]) }

func SeedOf(level int) int {
	if level <= 0 {
		return 0
	}
	return max(1, (min(level, 100)*visual.SeedMost+50)/100)
}

func SeedLevel(seed int) int {
	if seed <= 0 {
		return 0
	}
	return max(1, (seed*100+visual.SeedMost/2)/visual.SeedMost)
}

func SeedSays(seed int) string {
	if seed <= 0 {
		return say.T("debug.seed.random")
	}
	return say.F("debug.seed.value", map[string]any{"N": seed})
}

func FpsSays(fps int) string {
	return say.F("debug.fps.value", map[string]any{"N": fps})
}

func (p *BasicPages) Style() *shell.Page {
	return &shell.Page{
		Title: say.T("style.title"),
		Tiles: func() ([]widget.Cell, []func(int)) {
			now := style.ByName(p.options.Read().Screen.Style).Name
			var cells []widget.Cell
			var taps []func(int)
			for _, name := range style.Names() {
				cells = append(cells, widget.Cell{Label: say.T("style." + name), Chosen: name == now, Style: name})
				taps = append(taps, func(int) {
					if err := p.options.SetStyle(name); err != nil {
						slog.Error("the style could not be saved", "style", name, "err", err)
					}
				})
			}
			return cells, taps
		},
	}
}

func (p *BasicPages) Theme() *shell.Page {
	return &shell.Page{
		Title: say.T("theme.title"),

		// A grid rather than a row each: a row spends its width on a label and a small Swatch, and
		// twelve of them do not fit on a screen. A tile is the Swatch, with the name on it.
		Tiles: func() ([]widget.Cell, []func(int)) {
			now := p.options.Read().Screen.Theme
			follow, _ := theme.ByName(style.Theme(p.options.Read().Screen.Style, style.ThemeDefault))

			cells := []widget.Cell{{Label: say.T("theme.default"), Chosen: now == style.ThemeDefault, Palette: &follow}}
			acts := []func(int){p.useTheme(style.ThemeDefault)}

			for _, t := range theme.All {
				cells = append(cells, widget.Cell{
					Label:   t.Name,
					Chosen:  t.Name == now,
					Paint:   Swatch(t),
					Palette: &t,
				})
				acts = append(acts, p.useTheme(t.Name))
			}
			return cells, acts
		},
	}
}

func Swatch(t theme.Theme) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, _ theme.Theme) {
		ui.FillRect(s, at, t.Background)

		// A hairline, because a light theme's card on a light page has no edge of its own — and the
		// card being that color is the first thing the row is there to say.
		ui.Border(s, at, max(at.H/40, 1), t.Text.Blend(t.Background, 0.7))

		on := []theme.Color{t.Surface, t.Text, t.Accent, t.Accent2}

		pad := int(float64(at.H) * swatchPad)
		gap := int(float64(at.H) * swatchGap)

		// Square, so the side is whichever of the two runs out first: the height the card leaves, or
		// what is left of the width once the gaps are taken out.
		wide := (at.W - pad*2 - gap*(len(on)-1)) / len(on)
		side := min(at.H-pad*2, wide)

		// Centered as a block rather than spread to the edges, so the squares stay a group on the
		// card instead of four things pinned to its corners.
		x := at.X + (at.W-side*len(on)-gap*(len(on)-1))/2
		y := at.Y + (at.H-side)/2

		for i, c := range on {
			ui.FillRect(s, ui.Rect{X: x + i*(side+gap), Y: y, W: side, H: side}, c)
		}
	}
}

func ThemeSays(name string) string {
	if name == style.ThemeDefault {
		return say.T("theme.default")
	}
	return name
}

func (p *BasicPages) Language() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.language"),
		Build: func() ([]widget.Row, []func(int)) {
			now := say.Chosen()

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, tag := range say.Languages() {
				rows = append(rows, widget.Row{Label: say.Name(tag), Chosen: tag == now})
				acts = append(acts, func(int) { p.options.SetLanguage(tag) })
			}
			return rows, acts
		},
	}
}

func (p *BasicPages) Zone() *shell.Page {
	return &shell.Page{
		Title: say.T("zone.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Zone()

			rows := []widget.Row{{Label: "Home Assistant", Chosen: now == ""}}
			acts := []func(int){p.useZone("")}

			// Zones that are not in a region stand on their own, which today is UTC.
			for _, z := range tz.In("") {
				rows = append(rows, widget.Row{Label: z.Name, Chosen: z.Name == now})
				acts = append(acts, p.useZone(z.Name))
			}

			for _, region := range tz.Regions() {
				rows = append(rows, widget.Row{
					Label: region, Kind: widget.Chevron, Value: ChosenIn(region, now),
				})
				acts = append(acts, p.open(func() shell.View { return p.Region(region) }))
			}
			return rows, acts
		},
	}
}

func ChosenIn(region, now string) string {
	if now != "" && tz.Region(now) == region {
		return now[len(region)+1:]
	}
	return ""
}

func (p *BasicPages) Region(region string) *shell.Page {
	return &shell.Page{
		Title: region,
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Zone()

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, z := range tz.In(region) {
				rows = append(rows, widget.Row{
					Label: z.Name[len(region)+1:], Chosen: z.Name == now,
				})
				acts = append(acts, p.useZone(z.Name))
			}
			return rows, acts
		},
	}
}
