package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"

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
					v.drag(e.Window, l, ev.GestureDX, width)
				}
			default:
				v.pager.Lift(e.Window, 0)
			}
		case gogui.GestureSwipe:
			if -ev.VelocityY > swipeSpeed && -ev.VelocityY > abs(ev.VelocityX) {
				v.pager.Lift(e.Window, 0)
				v.move(e.Window, func() { v.home(l) })
			} else {
				v.pager.Lift(e.Window, ev.VelocityX)
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
