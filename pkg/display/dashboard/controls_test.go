package dashboard

import (
	"errors"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"
)

func TestSaveFailurePublishesActualState(t *testing.T) {
	state := State{Face: "Plain", Date: true}
	draws := 0
	c := newControls(controlOptions{Read: func() State { return state }, Redraw: func() { draws++ }})
	c.Restore(state)
	c.Commit("clock_face", func() error { return errors.New("write failed") })
	if c.Face.Get() != "Plain" || draws != 0 {
		t.Fatalf("failed write changed entity=%q redraws=%d", c.Face.Get(), draws)
	}
	c.Commit("clock_face", func() error { state.Face = "Analog"; return errors.New("directory sync failed after rename") })
	if c.Face.Get() != "Analog" || draws != 1 {
		t.Fatalf("visible saved state lost: entity=%q redraws=%d", c.Face.Get(), draws)
	}
}

type faceChoice string

func (f faceChoice) Label() string { return string(f) }

func TestChoiceCommandSavesAndKeepsEntityIdentity(t *testing.T) {
	state := State{Face: "Plain"}
	c := newControls(controlOptions{DeviceID: 1, Read: func() State { return state }, Face: Choices([]faceChoice{"Plain", "Analog"}, func(v faceChoice) error { state.Face = v.Label(); return nil })})
	c.Restore(state)
	c.Face.OnCommand("Analog")
	if state.Face != "Analog" || c.Face.Get() != "Analog" {
		t.Fatal("select command did not save and publish")
	}
	c.Face.OnCommand("Sundial")
	if state.Face != "Analog" {
		t.Fatal("unknown label reached writer")
	}
	ids := []string{"clock_format", "clock_face", "clock_position", "clock_size", "clock_color", "clock_date"}
	for i, entity := range c.Entities() {
		var base *esphome.Base
		switch e := entity.(type) {
		case *esphome.Select:
			base = &e.Base
		case *esphome.Switch:
			base = &e.Base
		}
		if base == nil || base.ObjectID != ids[i] || base.DeviceID != 1 {
			t.Fatalf("entity identity %d changed: %+v", i, base)
		}
	}
}
