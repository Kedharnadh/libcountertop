package clockface

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

var screens = []struct {
	name string
	w, h int
}{
	{"landscape", 1920, 1200},
	{"portrait", 1200, 1920},
}

func TestEveryOfferedFaceIsDrawable(t *testing.T) {
	for _, name := range Kinds() {
		if _, ok := registered[name]; !ok {
			t.Errorf("%q is offered in the settings and nothing draws it", name)
		}
	}
}

func TestEveryDrawableFaceIsOffered(t *testing.T) {
	for name := range registered {
		var found bool
		for _, offered := range Kinds() {
			found = found || offered == name
		}
		if !found {
			t.Errorf("%q is drawn and is not offered in the settings", name)
		}
	}
}

func TestAnUnknownNameFallsBackRatherThanPanicking(t *testing.T) {
	if Of("sundial") == nil {
		t.Fatal("an unknown face has nothing to draw it")
	}
	if Of("sundial") != registered[Default] {
		t.Error("an unknown face is not the default")
	}
}

func TestEveryFaceStaysInTheBoxItIsGiven(t *testing.T) {
	palette := theme.Default()
	edge := theme.Color{R: 255, G: 0, B: 255}

	for _, name := range Kinds() {
		for _, s := range screens {
			for _, twentyFour := range []bool{true, false} {
				t.Run(string(name)+" "+s.name, func(t *testing.T) {
					img := ui.NewImage(s.w, s.h, edge)

					in := ui.Rect{X: s.w / 8, Y: s.h / 8, W: s.w * 3 / 4, H: s.h * 3 / 4}
					ui.FillRect(img, in, palette.Background)

					Of(name).Draw(img, in, Read(at(12, 34), twentyFour), palette)

					var drew int
					for y := range s.h {
						for x := range s.w {
							got := img.At(x, y)

							if in.Contains(x, y) {
								if got != palette.Background {
									drew++
								}
								continue
							}
							if got != edge {
								t.Fatalf("painted at %d,%d, outside the %v it was given", x, y, in)
							}
						}
					}
					if drew == 0 {
						t.Error("the face drew nothing")
					}
				})
			}
		}
	}
}

func TestEveryFaceDrawsInACorner(t *testing.T) {
	palette := theme.Default()

	for _, name := range Kinds() {
		img := ui.NewImage(1200, 1920, palette.Background)
		in := ui.Rect{X: 600, Y: 0, W: 600, H: 480}

		Of(name).Draw(img, in, Read(at(12, 34), true), palette)

		var drew int
		for y := in.Y; y < in.Y+in.H; y++ {
			for x := in.X; x < in.X+in.W; x++ {
				if img.At(x, y) != palette.Background {
					drew++
				}
			}
		}
		if drew == 0 {
			t.Errorf("%q drew nothing in a corner", name)
		}
	}
}
