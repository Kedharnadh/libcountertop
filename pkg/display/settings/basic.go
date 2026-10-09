package settings

import (
	"log/slog"
	"slices"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
	tz "github.com/ygelfand/libcountertop/pkg/timezone"
)

type BasicState struct {
	Screen              config.Screen
	Clock               config.Clock
	Idle                config.Idle
	Visual              config.Visual
	PosterLabel, UISize string
}
type ScreenControl interface {
	Set(string, string) error
	SetDrawer(config.Edge)
	SetBacklight(int)
	SetMode(config.ScreenMode)
	Use(string)
}
type PosterControl interface {
	SetEnabled(bool)
	Next()
	SetEvery(config.PosterEvery)
}
type VisualControl interface {
	SetMaxFPS(int)
	SetSeed(int)
}
type PrivacyControl interface{ SetMarks(bool) }
type ADBControl interface {
	ADB() bool
	SetADB(bool)
}
type ZoneControl interface {
	ChosenZone() string
	SetZone(string)
}
type HoursControl interface{ SetHours(config.HourFormat) }
type BasicDependencies struct {
	Preferences                         Preferences
	Shell                               *shell.Shell
	Screen                              ScreenControl
	Poster                              PosterControl
	Visuals                             VisualControl
	Privacy                             PrivacyControl
	Access                              ADBControl
	Zone                                ZoneControl
	Dashboard                           HoursControl
	Clocks                              *ClockPages
	Idle                                *IdlePages
	UISize                              func() string
	PowerPage, VisualPage, PresencePage func() shell.View
}
type BasicPages struct{ options BasicDependencies }

func NewBasicPages(o BasicDependencies) *BasicPages { return &BasicPages{options: o} }
func (p *BasicPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(makePage()) }
}
func (p *BasicPages) useTheme(name string) func(int) { return func(int) { p.setTheme(name) } }
func (p *BasicPages) useZone(name string) func(int) {
	return func(int) { p.options.Zone.SetZone(name) }
}

const (
	swatchPad = 0.12
	swatchGap = 0.07
)

func (p *BasicPages) Debug() *shell.Page {
	return &shell.Page{
		Title: say.T("debug.title"),
		Build: func() ([]menu.Row, []func(int)) {
			fps, seed := p.state().Visual.MaxFPS, p.state().Visual.Seed
			adb := p.options.Access.ADB()
			return []menu.Row{
					{
						Glyph: gogui.IconTerminal,
						Label: say.T("system.adb"),
						Hint:  say.T("system.adb.hint"),
						Kind:  menu.Toggle,
						On:    adb,
					},
					{Label: say.T("debug.visuals"), Kind: menu.Chevron},
					{
						Label: say.T("debug.fps"),
						Kind:  menu.Slider,
						Level: FpsLevel(fps),
						Snap:  FpsSnap,
						Value: FpsSays(fps),
					},
					{
						Label: say.T("debug.seed"),
						Kind:  menu.Slider,
						Level: SeedLevel(seed),
						Value: SeedSays(seed),
					},
				}, []func(int){
					func(int) { p.options.Access.SetADB(!adb) },
					p.open(p.options.VisualPage),
					func(level int) { p.options.Visuals.SetMaxFPS(config.MaxFPSSteps[FpsIndex(level)]) },
					func(level int) { p.options.Visuals.SetSeed(SeedOf(level)) },
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
		Tiles: func() ([]menu.Cell, []func(int)) {
			now := style.ByName(p.state().Screen.Style).Name
			var cells []menu.Cell
			var taps []func(int)
			for _, name := range style.Names() {
				cells = append(
					cells,
					menu.Cell{Label: say.T("style." + name), Chosen: name == now, Style: name},
				)
				taps = append(taps, func(int) {
					if err := p.setStyle(name); err != nil {
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

		Tiles: func() ([]menu.Cell, []func(int)) {
			now := p.state().Screen.Theme
			follow, _ := theme.ByName(style.Theme(p.state().Screen.Style, theme.Follow))

			cells := []menu.Cell{
				{
					Label:   say.T("theme.default"),
					Chosen:  now == theme.Follow,
					Palette: &follow,
				},
			}
			acts := []func(int){p.useTheme(theme.Follow)}

			for _, t := range theme.All {
				cells = append(cells, menu.Cell{
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

		ui.Border(s, at, max(at.H/40, 1), t.Text.Blend(t.Background, 0.7))

		on := []theme.Color{t.Surface, t.Text, t.Accent, t.Accent2}

		pad := int(float64(at.H) * swatchPad)
		gap := int(float64(at.H) * swatchGap)

		wide := (at.W - pad*2 - gap*(len(on)-1)) / len(on)
		side := min(at.H-pad*2, wide)

		x := at.X + (at.W-side*len(on)-gap*(len(on)-1))/2
		y := at.Y + (at.H-side)/2

		for i, c := range on {
			ui.FillRect(s, ui.Rect{X: x + i*(side+gap), Y: y, W: side, H: side}, c)
		}
	}
}

func ThemeSays(name string) string {
	if name == theme.Follow {
		return say.T("theme.default")
	}
	return name
}

func (p *BasicPages) Language() *shell.Page {
	return &shell.Page{
		Title: say.T("settings.language"),
		Build: func() ([]menu.Row, []func(int)) {
			now := say.Chosen()

			var (
				rows []menu.Row
				acts []func(int)
			)
			for _, tag := range say.Languages() {
				rows = append(rows, menu.Row{Label: say.Name(tag), Chosen: tag == now})
				acts = append(acts, func(int) { p.setLanguage(tag) })
			}
			return rows, acts
		},
	}
}

func (p *BasicPages) Zone() *shell.Page {
	return &shell.Page{
		Title: say.T("zone.title"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Zone.ChosenZone()

			rows := []menu.Row{{Label: "Home Assistant", Chosen: now == ""}}
			acts := []func(int){p.useZone("")}

			for _, z := range tz.In("") {
				rows = append(rows, menu.Row{Label: z.Name, Chosen: z.Name == now})
				acts = append(acts, p.useZone(z.Name))
			}

			for _, region := range tz.Regions() {
				rows = append(rows, menu.Row{
					Label: region, Kind: menu.Chevron, Value: ChosenIn(region, now),
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
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Zone.ChosenZone()

			var (
				rows []menu.Row
				acts []func(int)
			)
			for _, z := range tz.In(region) {
				rows = append(rows, menu.Row{
					Label: z.Name[len(region)+1:], Chosen: z.Name == now,
				})
				acts = append(acts, p.useZone(z.Name))
			}
			return rows, acts
		},
	}
}

func (p *BasicPages) state() BasicState { return p.options.Preferences.Basic(p.options.UISize()) }
func (p *BasicPages) setTheme(name string) {
	if err := p.options.Preferences.Screen.Update(
		func(c *config.Screen) { c.Theme = name },
	); err == nil {
		p.options.Screen.Use(name)
	}
}
func (p *BasicPages) setLanguage(tag string) {
	saveLanguage(p.options.Preferences.Screen, p.options.Shell, tag)
}
func (p *BasicPages) setKeyboard(v config.KeyboardSize) {
	saveKeyboard(p.options.Preferences.Screen, p.options.Shell, v)
}

func saveLanguage(section storage.Section[config.Screen], stack *shell.Shell, tag string) {
	say.Use(tag)
	saveScreen(section, stack, "screen.language", func(c *config.Screen) { c.Language = tag })
}

func saveKeyboard(
	section storage.Section[config.Screen],
	stack *shell.Shell,
	v config.KeyboardSize,
) {
	saveScreen(section, stack, "screen.keyboard", func(c *config.Screen) { c.Keyboard = v })
}
func (p *BasicPages) setVolumeEdge(v config.Edge) {
	saveScreen(
		p.options.Preferences.Screen,
		p.options.Shell,
		"screen.volume",
		func(c *config.Screen) { c.Volume = v },
	)
}
func (p *BasicPages) setStyle(v string) error { return p.options.Screen.Set("style", v) }
func (p *BasicPages) setSize(v string) {
	if err := p.options.Screen.Set("size", v); err != nil {
		slog.Error("saving the display size failed", "err", err)
	}
}
func (p *BasicPages) toggleAuto(was bool) {
	mode := config.ModeAuto
	if was {
		mode = config.ModeManual
	}
	p.options.Screen.SetMode(mode)
}
func (p *BasicPages) toggleHours(was config.HourFormat) {
	hours := config.TwentyFourHour
	if was == hours {
		hours = config.TwelveHour
	}
	p.options.Dashboard.SetHours(hours)
}
