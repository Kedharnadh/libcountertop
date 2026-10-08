package cameracmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/camera/live"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type memory struct{ cam schema.Camera }

func (m *memory) Read() schema.Camera { return m.cam }

func (m *memory) Update(change func(*schema.Camera)) error {
	change(&m.cam)
	return nil
}

var _ storage.Section[schema.Camera] = (*memory)(nil)

func camera(m *memory) *live.Camera {
	return live.New(live.Options{
		Vendor: live.Vendor{
			EVMin: -1, EVMax: 1,
			Scenes: []string{"auto"}, Balances: []string{"auto"}, Effects: []string{"none"},
			Levels: true, Rates: []int{15, 30},
		},
		Streams: live.Streams{Main: []string{"1280x720"}, MainDefault: "1280x720"},
		Saved:   m.Read,
		Writer:  func() schema.CameraWriter { return schema.NewCameraWriter(m) },
	})
}

func row(t *testing.T, cam *live.Camera, name string) func(string) error {
	t.Helper()
	for _, r := range Rows(cam) {
		if r.Name == name {
			return r.Write
		}
	}
	t.Fatalf("no %s row", name)
	return nil
}

func TestABadValueIsRefusedBeforeItIsSaved(t *testing.T) {
	m := &memory{cam: schema.DefaultCamera()}
	if err := row(t, camera(m), "camera.ev")("9"); err == nil {
		t.Error("an exposure out of range was accepted")
	}
	if len(m.cam.Settings) != 0 {
		t.Errorf("saved %v anyway", m.cam.Settings)
	}
}

func TestAGoodValueIsSavedAndReadBack(t *testing.T) {
	m := &memory{cam: schema.DefaultCamera()}
	cam := camera(m)
	if err := row(t, cam, "camera.ev")("1"); err != nil {
		t.Fatal(err)
	}
	for _, r := range Rows(cam) {
		if r.Name == "camera.ev" && r.Field == "Camera.Settings" && r.Read(m.cam) != "1" {
			t.Errorf("reads back %q", r.Read(m.cam))
		}
	}
}

func TestAStillIsWrittenWhereAsked(t *testing.T) {
	at := filepath.Join(t.TempDir(), "still.jpg")
	out, err := Still(func() ([]byte, error) { return []byte("jpeg"), nil }, []string{at})
	if err != nil || !strings.Contains(out, at) {
		t.Fatalf("%q, %v", out, err)
	}
	if got, _ := os.ReadFile(at); string(got) != "jpeg" {
		t.Errorf("wrote %q", got)
	}
	if _, err := Still(
		func() ([]byte, error) { return nil, errors.New("no") },
		[]string{at},
	); err == nil {
		t.Error("a failed picture was reported as taken")
	}
}
