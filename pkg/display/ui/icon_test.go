package ui

import (
	"testing"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func TestFillPolygonTriangle(t *testing.T) {
	img := NewImage(40, 40, black)

	FillPolygon(img, []Point{{X: 0, Y: 39}, {X: 20, Y: 0}, {X: 39, Y: 39}}, white)

	n := drawn(img, 40, 40, black)
	if n < 680 || n > 860 {
		t.Errorf("a triangle in a 40x40 square filled %d pixels, want about 760", n)
	}
	if img.At(0, 0) != black || img.At(39, 0) != black {
		t.Error("the top corners are filled, and a triangle apex up should leave them")
	}
	if img.At(20, 38) != white {
		t.Error("the middle of the base is not filled")
	}
}

func TestFillPolygonClips(t *testing.T) {
	img := NewImage(20, 20, black)

	FillPolygon(img, []Point{{X: -50, Y: -50}, {X: 70, Y: -50}, {X: 10, Y: 70}}, white)

	if drawn(img, 20, 20, black) == 0 {
		t.Error("a shape covering the surface drew nothing")
	}
}

func TestFillPolygonNeedsThreePoints(t *testing.T) {
	img := NewImage(20, 20, black)

	FillPolygon(img, []Point{{X: 0, Y: 0}, {X: 19, Y: 19}}, white)

	if n := drawn(img, 20, 20, black); n != 0 {
		t.Errorf("a line filled %d pixels", n)
	}
}

func drawn(img *Image, w, h int, ground theme.Color) int {
	var n int
	for y := range h {
		for x := range w {
			if img.At(x, y) != ground {
				n++
			}
		}
	}
	return n
}

func TestDrawIconPaints(t *testing.T) {
	img := NewImage(64, 64, black)

	DrawIcon(img, icons.AVVolumeUp, Rect{W: 64, H: 64}, white, black)

	if n := drawn(img, 64, 64, black); n == 0 {
		t.Error("the icon drew nothing")
	}
}

func TestIconsDiffer(t *testing.T) {
	up := NewImage(64, 64, black)
	DrawIcon(up, icons.AVVolumeUp, Rect{W: 64, H: 64}, white, black)

	off := NewImage(64, 64, black)
	DrawIcon(off, icons.AVVolumeOff, Rect{W: 64, H: 64}, white, black)

	for y := range 64 {
		for x := range 64 {
			if up.At(x, y) != off.At(x, y) {
				return
			}
		}
	}
	t.Error("two different icons drew the same picture")
}

func TestDrawIconUsesTheColorItIsGiven(t *testing.T) {
	img := NewImage(64, 64, black)
	want := theme.Color{R: 0x00, G: 0x80, B: 0xf0}

	DrawIcon(img, icons.AVVolumeUp, Rect{W: 64, H: 64}, want, black)

	for y := range 64 {
		for x := range 64 {
			if img.At(x, y) == want {
				return
			}
		}
	}
	t.Error("no pixel of the icon is the color it was asked for")
}

func TestDrawIconBlendsAgainstTheGround(t *testing.T) {
	img := NewImage(64, 64, black)
	DrawIcon(img, icons.AVVolumeUp, Rect{W: 64, H: 64}, white, black)

	for y := range 64 {
		for x := range 64 {
			c := img.At(x, y)
			if c.R != c.G || c.G != c.B {
				t.Fatalf("%d,%d is %v, want a gray between black and white", x, y, c)
			}
		}
	}
}

func TestDrawIconScales(t *testing.T) {
	for _, size := range []int{16, 48, 200} {
		img := NewImage(size, size, black)
		DrawIcon(img, icons.AVVolumeUp, Rect{W: size, H: size}, white, black)

		if n := drawn(img, size, size, black); n == 0 {
			t.Errorf("at %dpx the icon drew nothing", size)
		}
	}
}

func TestDrawIconCentresInAWideRect(t *testing.T) {
	img := NewImage(200, 60, black)

	DrawIcon(img, icons.AVVolumeUp, Rect{W: 200, H: 60}, white, black)

	for y := range 60 {
		for x := range 200 {
			if x >= 70 && x < 130 {
				continue
			}
			if img.At(x, y) != black {
				t.Fatalf("the icon drew at %d,%d, outside the square it was fitted to", x, y)
			}
		}
	}
}

func TestDrawIconClips(t *testing.T) {
	img := NewImage(32, 32, black)

	DrawIcon(img, icons.AVVolumeUp, Rect{X: -20, Y: -20, W: 120, H: 120}, white, black)
	DrawIcon(img, icons.AVVolumeUp, Rect{X: 20, Y: 20, W: 120, H: 120}, white, black)
}

func TestDrawIconInNoSpace(t *testing.T) {
	img := NewImage(32, 32, black)

	DrawIcon(img, icons.AVVolumeUp, Rect{W: 0, H: 0}, white, black)
	DrawIcon(img, icons.AVVolumeUp, Rect{W: -5, H: 10}, white, black)

	if n := drawn(img, 32, 32, black); n != 0 {
		t.Errorf("an icon with no room drew %d pixels", n)
	}
}
