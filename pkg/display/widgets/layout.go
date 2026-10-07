package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

// LivePreview tracks camera readiness for one GUI session.
type LivePreview struct{ ready bool }

func (p *LivePreview) View(w *gogui.Window, starting string, watch func(ui.Rect) bool) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	bw, bh, sizing := float32(vw)*0.45, float32(vh)*0.6, gogui.FixedFixed
	if vh > vw {
		bw, bh, sizing = 0, float32(vh)*0.3, gogui.FillFixed
	}
	var content []gogui.View
	fill := gogui.Color{}
	if !p.ready {
		content = append(content, gogui.Label(starting, secondary()))
		fill = t.Cfg.ColorPanel
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:      "preview",
		Width:   bw,
		Height:  bh,
		Sizing:  sizing,
		Color:   fill,
		Radius:  gogui.RadiusMedium,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: content,
		AmendLayout: func(e gogui.EventCtx) {
			s := e.Layout.Shape
			p.ready = watch(ui.Rect{X: int(s.X), Y: int(s.Y), W: int(s.Width), H: int(s.Height)})
		},
	})
}

// Beside arranges a preview beside a list or above it in portrait mode.
func Beside(w *gogui.Window, aside, list gogui.View) gogui.View {
	vw, vh := w.WindowSize()
	cfg := gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding, Spacing: gogui.SpacingLarge, Content: []gogui.View{aside, list}}
	if vh > vw {
		return gogui.Column(cfg)
	}
	return gogui.Row(cfg)
}

// Gestures prevents sliders from dragging their surrounding scroll view.
// It belongs to one GUI session and runs on the UI loop.
type Gestures struct {
	gripped bool
	Release func(gogui.EventCtx)
}

func panning(e gogui.EventCtx) (gogui.GesturePhase, bool) {
	if e.Event == nil || e.Event.GestureType != gogui.GesturePan {
		return 0, false
	}
	return e.Event.GesturePhase, true
}

func (g *Gestures) Grip(v gogui.View) gogui.View {
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{v},
		OnGesture: func(e gogui.EventCtx) {
			if phase, ok := panning(e); ok {
				g.gripped = phase == gogui.GesturePhaseBegan || phase == gogui.GesturePhaseChanged
				e.Consume()
			}
		},
	})
}

func (g *Gestures) HoldStill(e gogui.EventCtx) {
	phase, ok := panning(e)
	if ok {
		g.Release(e)
	}
	if !ok || !g.gripped {
		return
	}
	if phase != gogui.GesturePhaseBegan && phase != gogui.GesturePhaseChanged {
		g.gripped = false
	}
	e.Consume()
}
