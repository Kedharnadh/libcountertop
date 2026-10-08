package widgets

import (
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const swipeSpeed = 300

func (v *View) gesture(l tessera.Layout) func(gogui.EventCtx) {
	return func(e gogui.EventCtx) {
		ev := e.Event
		v.mu.Lock()
		width := v.width
		v.mu.Unlock()
		switch ev.GestureType {
		case gogui.GesturePan:
			switch ev.GesturePhase {
			case gogui.GesturePhaseBegan, gogui.GesturePhaseChanged:
				if abs(ev.GestureDX) >= abs(ev.GestureDY) {
					v.drag(l, ev.GestureDX, width)
				}
			default:
				v.release(e.Window, l, width, 0)
			}
		case gogui.GestureSwipe:
			if -ev.VelocityY > swipeSpeed && -ev.VelocityY > abs(ev.VelocityX) {
				v.move(e.Window, func() { v.home(l) })
			} else {
				v.release(e.Window, l, width, ev.VelocityX)
			}
		default:
			return
		}
		e.Consume()
		e.Window.InvalidateLayout()
	}
}

func abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

func (v *View) dots(at ui.Rect, shown []int, page int) gogui.View {
	t := gogui.CurrentTheme().Cfg
	side := style.Reach() * 0.3
	var dots []gogui.View
	for i, p := range shown {
		fill := t.ColorTextSecondary.WithOpacity(0.4)
		if p == page {
			fill = t.ColorAccent
		}
		dots = append(dots, gogui.Row(gogui.ContainerCfg{
			ID:      fmt.Sprintf("tessera-dot-%d", i),
			Width:   side,
			Height:  side,
			Sizing:  gogui.FixedFixed,
			Radius:  gogui.RadiusLarge,
			Color:   fill,
			Padding: gogui.NoPadding,
		}))
	}
	return widgets.Placed(at, gogui.Row(gogui.ContainerCfg{
		Width:   float32(at.W),
		Height:  float32(at.H),
		Sizing:  gogui.FixedFixed,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Spacing: gogui.SpacingMedium,
		Padding: gogui.NoPadding,
		Content: dots,
	}))
}
