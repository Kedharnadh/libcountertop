package screens

import (
	"image"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/idleview"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/display/visualview"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

func (a *App) idleScreen(v *idleview.View) *Screen {
	return &Screen{
		Clear: true,
		View:  v,
		Build: func(w *gogui.Window) gogui.View { return a.idle(w, v) },
	}
}

func (a *App) idle(w *gogui.Window, v *idleview.View) gogui.View {
	vw, vh := w.WindowSize()
	cfg := a.o.Settings()
	pal := a.ui.Palette()
	slots := v.Keep(vw, vh)
	areas := idleview.Areas(vw, vh, len(slots))
	box := dashboard.Place(cfg.Idle.Position, cfg.Idle.Align, cfg.Idle.Size, vw, vh)

	var layers []gogui.View
	if len(slots) == 0 {
		layers = append(
			layers,
			widgets.Placed(
				ui.Rect{W: vw, H: vh},
				gogui.Column(
					gogui.ContainerCfg{
						Sizing:  gogui.FillFill,
						Padding: gogui.NoPadding,
						Color:   color(pal.Background),
					},
				),
			),
		)
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
	}
	if build, ok := clockview.Of(cfg.Idle.Face); ok {
		r := idleview.Reading(
			idleview.Settings{Screen: cfg.Screen, Clock: cfg.Clock, Idle: cfg.Idle},
			time.Now(),
		)
		tones := make([]visual.Traits, len(slots))
		for n, slot := range slots {
			tones[n] = visual.Kind(slot.Kind).Traits()
		}
		for _, p := range idleview.Pieces(pal, tones, areas, box) {
			clock := build(w, box, r, cfg.Clock.Ink.Over(p.Palette))
			if p.Clip.W == 0 {
				layers = append(layers, widgets.Placed(box, clock))
				continue
			}
			inner := ui.Rect{X: box.X - p.Clip.X, Y: box.Y - p.Clip.Y, W: box.W, H: box.H}
			layers = append(layers, widgets.Placed(p.Clip, gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFill,
				Padding: gogui.NoPadding,
				Clip:    true,
				Content: []gogui.View{widgets.Placed(inner, clock)},
			})))
		}
	}
	if wc := cfg.Weather; wc.Idle {
		if v := a.weatherLayer(w, vw, vh, 0, pal); v != nil {
			layers = append(layers, v)
		}
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:      "idle",
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		Content: layers,
		OnClick: func(e gogui.EventCtx) {
			a.o.Shell.Remove(v)
			e.Consume()
			e.Window.InvalidateLayout()
		},
	})
}

func (a *App) visualScreen(v *visualview.View) *Screen {
	return &Screen{
		Clear: true,
		View:  v,
		Build: func(w *gogui.Window) gogui.View {
			vw, vh := w.WindowSize()
			v.Keep(vw, vh)
			return gogui.Column(gogui.ContainerCfg{
				ID:      "visual",
				Sizing:  gogui.FillFill,
				Padding: gogui.NoPadding,
				OnClick: func(e gogui.EventCtx) {
					a.o.Shell.Close()
					e.Consume()
					e.Window.InvalidateLayout()
				},
			})
		},
	}
}

func (a *App) seeThrough(on bool) {
	a.mu.Lock()
	r := a.r
	a.mu.Unlock()
	if r == nil {
		return
	}
	if on {
		r.SetClear(&gogui.ColorTransparent)
	} else {
		r.SetClear(nil)
	}
}
