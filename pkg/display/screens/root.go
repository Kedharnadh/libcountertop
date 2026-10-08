package screens

import (
	"image"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (a *App) root(w *gogui.Window) gogui.View {
	covering, above := a.nav.Showing(a.o.Shell.Views())
	if a.fullLook() || a.o.Stage.StagedFull() {
		a.seeThrough(true)
		layers := []gogui.View{
			gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding}),
		}
		for _, o := range a.nav.Overlays() {
			if v := o.Build(w); v != nil {
				layers = append(layers, widgets.Floating(o.Priority+1, v))
			}
		}
		return gogui.Column(
			gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding, Content: layers},
		)
	}
	a.seeThrough(covering != nil && covering.Clear)
	base := a.dashboard(w)
	if addr := a.o.Offering(); addr != "" {
		base = a.onboard(w, addr)
	}
	switch {
	case covering == nil:
	case covering.Title == "":
		base = covering.Build(w)
	default:
		base = a.page(w, covering)
	}
	layers := []gogui.View{base}
	for _, s := range above {
		layers = append(layers, widgets.Floating(0, s.Build(w)))
	}
	for _, o := range a.nav.Overlays() {
		if v := o.Build(w); v != nil {
			layers = append(layers, widgets.Floating(o.Priority+1, v))
		}
	}
	if a.ui.Editor.Typing(w) {
		if v := a.keyboard(w); v != nil {
			layers = append(
				layers,
				gogui.Column(
					gogui.ContainerCfg{
						Float:       true,
						FloatZIndex: 1001,
						Sizing:      gogui.FillFill,
						Padding:     gogui.NoPadding,
						Content:     []gogui.View{v},
					},
				),
			)
		}
	}
	return gogui.Column(
		gogui.ContainerCfg{Sizing: gogui.FillFill, Padding: gogui.NoPadding, Content: layers},
	)
}

func (a *App) dashboard(w *gogui.Window) gogui.View {
	vw, vh := w.WindowSize()
	cfg := a.o.Settings()
	pal := a.ui.Palette()
	ink := cfg.Clock.Ink.Over(pal)
	r := face.Read(time.Now(), cfg.Screen.Hours == schema.TwentyFourHour)
	if !cfg.Clock.Date {
		r = r.Undated()
	}
	box := dashboard.Box(cfg.Clock.Position, cfg.Clock.Size, vw, vh)
	tabs := a.o.Tabs.List()
	showing, tabbed := a.o.Tabs.Showing()
	kind, drawn := a.boards[showing.Kind]
	margin := int(reach() * 0.5)
	strip := ui.Rect{X: margin, Y: margin, W: vw - 2*margin, H: int(reach() * 2)}
	if len(tabs) > 0 && box.Y < strip.Y+strip.H {
		shift := strip.Y + strip.H - box.Y
		box.Y += shift
		box.H = max(box.H-shift, 1)
	}

	var layers []gogui.View
	if backdrop, behind := a.o.Poster.
		Backdrop(
			vw,
			vh,
			image.Rect(box.X, box.Y, box.X+box.W, box.Y+box.H),
			pal.Background,
		); backdrop != nil {
		layers = append(
			layers,
			widgets.Placed(
				ui.Rect{W: vw, H: vh},
				widgets.Picture(clockview.ImageSource("poster/"+behind, backdrop), vw, vh),
			),
		)
	}

	switch build, ok := clockview.Of(cfg.Clock.Face); {
	case tabbed && drawn:
		below := strip.Y + strip.H + margin
		layers = append(
			layers,
			kind.Draw(
				ui.Rect{X: margin, Y: below, W: vw - 2*margin, H: vh - below - margin},
				showing,
				pal,
			),
		)
	case ok:
		layers = append(layers, widgets.Placed(box, build(w, box, r, ink)))
	}
	if wc := cfg.Weather; wc.Dashboard && (!tabbed || !drawn) {
		top := float32(0)
		if len(tabs) > 0 {
			top = float32(strip.Y + strip.H)
		}
		if v := a.weatherLayer(w, vw, vh, top, pal); v != nil {
			layers = append(layers, v)
		}
	}
	if len(tabs) > 0 {
		layers = append(layers, a.tabStrip(strip, tabs, showing.Key))
	}

	frame := gogui.ContainerCfg{
		ID:      "dashboard",
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		Content: layers,
	}
	dash := gogui.Column(frame)
	return gogui.Column(
		gogui.ContainerCfg{
			Sizing:  gogui.FillFill,
			Padding: gogui.NoPadding,
			Color:   color(pal.Background),
			Content: []gogui.View{dash},
		},
	)
}

func (a *App) page(w *gogui.Window, p *Screen) gogui.View {
	return a.ui.Toolkit.PageFrame(
		w,
		widgets.FrameOptions{
			Title:   p.Title,
			Fixed:   p.Fixed,
			Build:   p.Build,
			Back:    func() { a.o.Shell.Pop() },
			Gesture: a.ui.HoldStill,
			Room:    a.ui.Editor.Room(w),
		},
	)
}

func color(c theme.Color) gogui.Color { return gogui.RGB(c.R, c.G, c.B) }
