package vision

import (
	"bytes"
	"errors"
	"image/jpeg"
	"testing"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/camera/session"
)

func red(w, h int) []byte {
	rgba := make([]byte, w*h*4)
	for i := 0; i < len(rgba); i += 4 {
		rgba[i], rgba[i+1], rgba[i+2], rgba[i+3] = 220, 30, 30, 255
	}
	return rgba
}

func newVision(covered bool) *Vision {
	return New(Options{
		Covered: func() bool { return covered },
		Picture: func(time.Duration) (session.Picture, error) {
			return session.Picture{RGBA: red(8, 8), Width: 8, Height: 8}, nil
		},
	})
}

// ESPHome's CameraImageRequest has no key.
func TestTheCameraIsOneEntityAndItTakesPictures(t *testing.T) {
	ents := newVision(false).Entities()
	if len(ents) != 1 {
		t.Fatalf("%d entities, want the one camera", len(ents))
	}

	cam, ok := ents[0].(*esphome.Camera)
	if !ok {
		t.Fatalf("the entity is %T, want a camera", ents[0])
	}
	if cam.ObjectID == "" {
		t.Error("the camera has no object id")
	}
	if cam.Image == nil {
		t.Fatal("the camera has no Image")
	}
	if out, err := cam.Image(); err != nil || len(out) == 0 {
		t.Errorf("a picture came back as %d bytes, %v", len(out), err)
	}
}

func TestACoveredCameraTakesNoPicture(t *testing.T) {
	if _, err := newVision(true).Still(); !errors.Is(err, ErrCovered) {
		t.Errorf("err = %v, want %v", err, ErrCovered)
	}
}

func TestAPictureEncodesWithItsColour(t *testing.T) {
	w, h := 8, 8
	rgba := red(w, h)
	out, err := encode(session.Picture{RGBA: rgba, Width: w, Height: h})
	if err != nil {
		t.Fatal(err)
	}
	img, err := jpeg.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(4, 4).RGBA()
	if r>>8 < 150 || g>>8 > 90 || b>>8 > 90 {
		t.Errorf("a red picture came back as %d %d %d", r>>8, g>>8, b>>8)
	}
	if _, err := encode(session.Picture{RGBA: rgba[:3], Width: w, Height: h}); err == nil {
		t.Error("a short picture encoded")
	}
}
