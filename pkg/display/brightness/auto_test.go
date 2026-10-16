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

func TestNightPinsTheFloorAndDayRestoresAuto(t *testing.T) {
	a := NewAuto(1)
	auto := schema.Screen{Mode: schema.ModeAuto, Backlight: NeutralBias}
	day := a.Follow(300, auto)

	a.Night(schema.NightDim)
	if step := a.Follow(300, auto); !step.Apply || step.Level != Floor {
		t.Fatalf("night step %+v, want the floor applied", step)
	}
	if again := a.Follow(300, auto); again.Apply {
		t.Errorf("a steady night applied %d again", again.Level)
	}

	a.Night(schema.NightNone)
	if step := a.Follow(300, auto); !step.Apply || step.Level != day.Level {
		t.Errorf("day step %+v, want %d back", step, day.Level)
	}
}

func TestNightOffDarkensTheBacklight(t *testing.T) {
	a := NewAuto(1)
	a.Night(schema.NightOff)
	if step := a.Follow(300, schema.Screen{Mode: schema.ModeAuto}); !step.Apply || step.Level != 0 {
		t.Errorf("night off step %+v, want 0 applied", step)
	}
}

func TestNightKeepsFollowingTheRoom(t *testing.T) {
	a := NewAuto(1)
	auto := schema.Screen{Mode: schema.ModeAuto, Backlight: NeutralBias}
	a.Night(schema.NightClock)
	for range 100 {
		a.Follow(800, auto)
	}
	if lux, ok := a.Ambient(); !ok || lux < 790 {
		t.Errorf("ambient during night is %v %v, want near 800", lux, ok)
	}

	a.Night(schema.NightNone)
	if step := a.Follow(800, auto); !step.Apply || step.Level != Brightness(800, NeutralBias) {
		t.Errorf("day step %+v, want the bright-room level", step)
	}
}

func TestNightInManualModeRestoresTheChosenBacklight(t *testing.T) {
	a := NewAuto(1)
	manual := schema.Screen{Mode: schema.ModeManual, Backlight: 60}

	a.Night(schema.NightDim)
	if step := a.Follow(300, manual); !step.Apply || step.Level != Floor {
		t.Fatalf("manual night step %+v, want the floor", step)
	}
	a.Night(schema.NightNone)
	if step := a.Follow(300, manual); !step.Apply || step.Level != 60 {
		t.Errorf("manual day step %+v, want 60 back", step)
	}
	if again := a.Follow(300, manual); again.Apply {
		t.Errorf("manual mode applied %d after restoring", again.Level)
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

func TestAWakeLightRisesThroughNightAndHandsBack(t *testing.T) {
	a := NewAuto(1)
	manual := schema.Screen{Mode: schema.ModeManual, Backlight: 80}
	a.Night(schema.NightOff)
	share, rising := 0.0, true
	a.Rising(func() (float64, bool) { return share, rising })

	if step := a.Follow(1, manual); !step.Apply || step.Level != Floor {
		t.Fatalf("start of the wake light %+v, want the floor", step)
	}
	share = 0.5
	half := a.Follow(1, manual)
	if !half.Apply || half.Level <= Floor || half.Level >= 80 {
		t.Errorf("halfway %+v, want between the floor and 80", half)
	}
	share = 1
	if step := a.Follow(1, manual); step.Level != 80 {
		t.Errorf("end of the wake light %+v, want 80", step)
	}

	rising = false
	a.Night(schema.NightNone)
	if step := a.Follow(1, manual); !step.Apply || step.Level != 80 {
		t.Errorf("after the wake light %+v, want the manual 80 reapplied", step)
	}
}
