package settings

import (
	"slices"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (p *BasicPages) Display() *shell.Page {
	return &shell.Page{
		Title: say.T("display.title"),
		Build: func() ([]menu.Row, []func(int)) {
			cfg := p.state()
			auto := cfg.Screen.Mode == config.ModeAuto

			rows := []menu.Row{
				{
					Label: say.T("display.brightness"),
					Kind:  menu.Slider,
					Level: cfg.Screen.Backlight,
				},
				{Label: say.T("display.auto"), Kind: menu.Toggle, On: auto},
				{
					Glyph: gogui.IconPalette,
					Label: say.T("display.theme"),
					Kind:  menu.Chevron,
					Value: ThemeSays(cfg.Screen.Theme),
				},
				{
					Label: say.T("display.style"),
					Kind:  menu.Chevron,
					Value: say.T("style." + style.ByName(cfg.Screen.Style).Name),
				},
				{
					Label: say.T("display.size"),
					Kind:  menu.Slider,
					Level: SizeLevel(cfg.UISize),
					Snap:  SizeSnap,
					Value: say.T("uisize." + cfg.UISize),
				},
				{
					Glyph: gogui.IconClock,
					Label: say.T("display.clock"),
					Kind:  menu.Chevron,
					Value: cfg.Clock.Face.Label(),
				},
				{
					Glyph: gogui.IconMoon,
					Label: say.T("display.idle"),
					Kind:  menu.Chevron,
					Value: cfg.Idle.After.Label(),
				},
				{
					Glyph: gogui.IconPicture,
					Label: say.T("display.poster"),
					Kind:  menu.Chevron,
					Value: cfg.PosterLabel,
				},
				{
					Label: say.T("controls.dock"),
					Kind:  menu.Chevron,
					Value: cfg.Screen.Drawer.Label(),
				},
				{
					Label: say.T("controls.volume"),
					Kind:  menu.Chevron,
					Value: cfg.Screen.Volume.Label(),
				},
				{
					Glyph: gogui.IconEye,
					Label: say.T("display.marks"),
					Kind:  menu.Toggle,
					On:    cfg.Screen.Marks,
				},
			}
			acts := []func(int){
				p.options.Screen.SetBacklight,
				func(int) { p.toggleAuto(auto) },
				p.open(func() shell.View { return p.Theme() }),
				p.open(func() shell.View { return p.Style() }),
				func(level int) { p.setSize(SizeOf(level)) },
				p.open(func() shell.View { return p.options.Clocks.Page() }),
				p.open(func() shell.View { return p.options.Idle.Page() }),
				p.open(func() shell.View { return p.Poster() }),
				p.open(func() shell.View { return p.Dock() }),
				p.open(func() shell.View { return p.VolumeEdge() }),
				func(int) { p.options.Privacy.SetMarks(!cfg.Screen.Marks) },
			}
			if p.options.PresencePage != nil {
				rows = append(
					rows,
					menu.Row{
						Glyph: gogui.IconEye,
						Label: say.T("display.presence"),
						Kind:  menu.Chevron,
					},
				)
				acts = append(acts, p.open(p.options.PresencePage))
			}
			return rows, acts
		},
	}
}
func SizeOf(level int) string {
	all := config.ScreenSizes()
	last := len(all) - 1
	return all[min(max((level*last+50)/100, 0), last)]
}
func SizeLevel(name string) int {
	all := config.ScreenSizes()
	return max(slices.Index(all, name), 0) * 100 / (len(all) - 1)
}
func SizeSnap(level int) int { return SizeLevel(SizeOf(level)) }
