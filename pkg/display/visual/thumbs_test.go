package visual

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
	"testing/fstest"
)

func jpegOf(t *testing.T, w, h int) []byte {
	t.Helper()
	var b bytes.Buffer
	if err := jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, w, h)), nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestThumbsReadTheirArtworkAndFitIt(t *testing.T) {
	th := NewThumbs(fstest.MapFS{
		"thumbs/orb.jpg":          {Data: jpegOf(t, 32, 18)},
		"thumbs/orb-portrait.jpg": {Data: jpegOf(t, 18, 32)},
	}, "thumbs")

	if img := th.Thumbnail(Orb, false); img == nil {
		t.Fatal("no landscape thumbnail")
	}
	if img := th.Thumbnail(Orb, true); img == nil {
		t.Fatal("no portrait thumbnail")
	}
	if img := th.Thumbnail(Halo, false); img != nil {
		t.Error("a missing thumbnail decoded")
	}

	fit := th.Fit(Orb, 10, 20)
	if fit == nil || fit.Bounds().Dx() != 10 || fit.Bounds().Dy() != 20 {
		t.Fatalf("fit = %v, want 10x20", fit)
	}
	if th.Fit(Orb, 10, 20) != fit {
		t.Error("a second fit at the same size was not cached")
	}
	if th.Fit(Orb, 0, 20) != nil {
		t.Error("an empty fit returned a picture")
	}
}
