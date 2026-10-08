package live

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type memory struct{ cam schema.Camera }

func (m *memory) Read() schema.Camera { return m.cam }

func (m *memory) Update(change func(*schema.Camera)) error {
	change(&m.cam)
	return nil
}

func (m *memory) camera() *Camera {
	m.cam = schema.DefaultCamera()
	return New(Options{
		Vendor:  levelled,
		Streams: Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
		Saved:   m.Read,
		Writer:  func() schema.CameraWriter { return schema.NewCameraWriter(m) },
	})
}

func TestABadValueIsRefusedBeforeItIsSaved(t *testing.T) {
	var m memory
	if err := m.camera().Controls().Change("ev", "9"); err == nil {
		t.Error("an exposure out of range was accepted")
	}
	if len(m.cam.Settings) != 0 {
		t.Errorf("saved %v anyway", m.cam.Settings)
	}
}

func TestAGoodValueIsSavedAndReadBack(t *testing.T) {
	var m memory
	knobs := m.camera().Controls()
	if err := knobs.Change("ev", "1"); err != nil {
		t.Fatal(err)
	}
	if got, err := knobs.Value("ev"); err != nil || got != "1" {
		t.Errorf("reads back %q, %v", got, err)
	}
	if m.cam.Settings["ev"] != "1" {
		t.Errorf("saved %v", m.cam.Settings)
	}
}
