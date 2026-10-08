package boot

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
)

var sizes = []struct {
	name string
	w, h int
}{
	{"landscape", 1920, 1200},
	{"portrait", 1200, 1920},
}

func coming() []startup.Progress {
	return []startup.Progress{
		{Name: "wifi", Doing: "looking for the network"},
		{Name: "dhcp", Doing: "asking for an address", Done: true},
	}
}

func raster(w, h int, progress []startup.Progress) *ui.Image {
	img := ui.NewImage(w, h, theme.Default().Surface)
	Raster(img, theme.Default(), progress, "1.0", "Skip")
	return img
}

func same(a, b *ui.Image, in ui.Rect) bool {
	for y := in.Y; y < in.Y+in.H; y++ {
		for x := in.X; x < in.X+in.W; x++ {
			if a.At(x, y) != b.At(x, y) {
				return false
			}
		}
	}
	return true
}

func TestSplitDividesTheLongEdge(t *testing.T) {
	for _, s := range sizes {
		logo, list := Split(s.w, s.h)

		if logo.W <= 0 || logo.H <= 0 || list.W <= 0 || list.H <= 0 {
			t.Fatalf("%s: split gave an empty area: logo %+v list %+v", s.name, logo, list)
		}

		if s.w >= s.h {
			if logo.X+logo.W != list.X {
				t.Errorf("%s: the list starts at %d, want the logo's right edge %d",
					s.name, list.X, logo.X+logo.W)
			}
			if logo.H != s.h || list.H != s.h {
				t.Errorf("%s: a side by side split should use the full height", s.name)
			}
			continue
		}

		if logo.Y+logo.H != list.Y {
			t.Errorf("%s: the list starts at %d, want the logo's bottom edge %d",
				s.name, list.Y, logo.Y+logo.H)
		}
		if logo.W != s.w || list.W != s.w {
			t.Errorf("%s: a stacked split should use the full width", s.name)
		}
	}
}

func TestRasterFitsBothWaysUp(t *testing.T) {
	for _, s := range sizes {
		raster(s.w, s.h, coming())
	}
}

func TestRasterWritesTheList(t *testing.T) {
	_, list := Split(1920, 1200)
	if same(raster(1920, 1200, nil), raster(1920, 1200, coming()), list) {
		t.Error("the list area is identical with and without a list to draw")
	}
}

func TestDoneAndWaitingLookDifferent(t *testing.T) {
	waiting := raster(1920, 1200, []startup.Progress{{Name: "wifi", Doing: "looking"}})
	done := raster(1920, 1200, []startup.Progress{{Name: "wifi", Doing: "looking", Done: true}})

	_, list := Split(1920, 1200)
	if same(waiting, done, list) {
		t.Error("a component that is up is drawn the same as one that is not")
	}
}

func TestVersionStaysInItsCorner(t *testing.T) {
	for _, s := range sizes {
		img := ui.NewImage(s.w, s.h, theme.Default().Surface)
		blank := ui.NewImage(s.w, s.h, theme.Default().Surface)

		Version(img, theme.Default(), "1.0")

		kept := ui.Rect{X: s.w / 2, Y: s.h * 7 / 8, W: s.w / 2, H: s.h / 8}
		for y := range s.h {
			for x := range s.w {
				if kept.Contains(x, y) {
					continue
				}
				if img.At(x, y) != blank.At(x, y) {
					t.Fatalf("%s: the build is drawn at %d,%d, outside its corner", s.name, x, y)
				}
			}
		}
	}
}
