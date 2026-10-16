package widgets

import (
	"testing"

	gogui "github.com/go-gui-org/go-gui/gui"
)

func TestAFlickOffASliderLetsThePageScrollAgain(t *testing.T) {
	g := Gestures{Release: func(gogui.EventCtx) {}}
	hold := func(e *gogui.Event) bool {
		g.HoldStill(gogui.EventCtx{Event: e})
		return e.IsHandled
	}
	pan := func(phase gogui.GesturePhase) *gogui.Event {
		return &gogui.Event{GestureType: gogui.GesturePan, GesturePhase: phase}
	}

	g.gripped = true
	if !hold(pan(gogui.GesturePhaseChanged)) {
		t.Fatal("the page scrolled while a slider was being dragged")
	}
	hold(&gogui.Event{GestureType: gogui.GestureSwipe})
	g.Lift()
	if hold(pan(gogui.GesturePhaseChanged)) {
		t.Error("after the finger lifted, the page still refused to scroll")
	}
}
