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
					Glyph: gogui.IconHome,
					Label: say.T("display.dashboard"),
					Kind:  menu.Chevron,
					Value: p.dashboardMode().Label(),
				},
				{
					Glyph: gogui.IconMoon,
					Label: say.T("display.idle"),
					Kind:  menu.Chevron,
					Value: cfg.Idle.After.Label(),
				},
				SegmentRow(say.T("controls.dock"), config.Edges(), cfg.Screen.Drawer),
				SegmentRow(say.T("controls.volume"), config.Sides(), cfg.Screen.Volume),
				SegmentRow(say.T("display.alerts"), config.AlertStyles(), cfg.Screen.Alert),
				{
					Label: say.T("display.timer"),
					Kind:  menu.Chevron,
					Value: cfg.Screen.Timer.Label(),
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
				p.open(func() shell.View { return p.DashboardPage() }),
				p.open(func() shell.View { return p.options.Idle.Page() }),
				SegmentPick(config.Edges(), p.options.Screen.SetDrawer),
				SegmentPick(config.Sides(), p.setVolumeEdge),
				SegmentPick(config.AlertStyles(), p.setAlert),
				p.open(func() shell.View {
					return Choose(
						say.T("display.timer"),
						config.TimerStyles(),
						func() config.TimerStyle { return p.state().Screen.Timer },
						p.setTimer,
						nil,
					)
				}),
				func(int) { p.options.Privacy.SetMarks(!cfg.Screen.Marks) },
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
