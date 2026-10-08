package brightness

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestManualModeLeavesTheBacklightAlone(t *testing.T) {
	a := NewAuto(1)
	screen := schema.Screen{Mode: schema.ModeManual, Backlight: NeutralBias}

	if step := a.Follow(300, screen); step.Apply {
		t.Errorf("manual mode set the backlight to %d", step.Level)
	}
	if lux, ok := a.Ambient(); !ok || lux != 300 {
		t.Errorf("ambient is %v %v, want 300 true", lux, ok)
	}
}

func TestASteadyRoomIsAppliedOnceAndThenLeft(t *testing.T) {
	a := NewAuto(1)
	screen := schema.Screen{Mode: schema.ModeAuto, Backlight: NeutralBias}

	first := a.Follow(300, screen)
	if !first.Apply || first.Level != Brightness(300, NeutralBias) {
		t.Fatalf("first step %+v, want the curve applied", first)
	}
	if again := a.Follow(300, screen); again.Apply {
		t.Errorf("an unchanged room applied %d again", again.Level)
	}
}

func TestLeavingManualModeAppliesAgain(t *testing.T) {
	a := NewAuto(1)
	auto := schema.Screen{Mode: schema.ModeAuto, Backlight: NeutralBias}

	a.Follow(300, auto)
	a.Follow(300, schema.Screen{Mode: schema.ModeManual})

	if step := a.Follow(300, auto); !step.Apply {
		t.Error("returning to auto did not reapply the level")
	}
}
