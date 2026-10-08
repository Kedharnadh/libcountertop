package clockface

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func every() []Kind {
	out := make([]Kind, 0, len(registered))
	for name := range registered {
		out = append(out, name)
	}
	return out
}

func TestEveryFaceStaysInsideTheBoxItIsGiven(t *testing.T) {
	palette := theme.Default()

	const pad = 60
	w, h := 1200, 1920
	in := ui.Rect{X: pad, Y: pad, W: w - 2*pad, H: h - 2*pad}

	readings := []Reading{
		Read(time.Date(2026, 9, 21, 14, 37, 0, 0, time.UTC), true),
		Read(time.Date(2026, 9, 21, 9, 5, 0, 0, time.UTC), false),
		Read(time.Date(2026, 9, 21, 23, 59, 0, 0, time.UTC), false),
		Read(time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), true).Undated(),
	}

	for _, name := range every() {
		for _, r := range readings {
			blank := ui.NewImage(w, h, palette.Background)
			img := ui.NewImage(w, h, palette.Background)
			ui.Fill(img, palette.Background)

			Of(name).Draw(img, in, r, palette)

			if painted := ui.Changed(blank, img); !in.Holds(painted) {
				t.Errorf("%s drawing %q%s painted %+v, outside the box %+v",
					name, r.Time, r.Suffix, painted, in)
			}
		}
	}
}

func TestEveryFaceStaysInsideAnAwkwardBox(t *testing.T) {
	palette := theme.Default()
	r := Read(time.Date(2026, 9, 21, 10, 8, 0, 0, time.UTC), false)

	for _, box := range []ui.Rect{
		{X: 40, Y: 40, W: 300, H: 1200},
		{X: 40, Y: 40, W: 1600, H: 240},
		{X: 40, Y: 40, W: 200, H: 200},
	} {
		w, h := box.X+box.W+40, box.Y+box.H+40

		for _, name := range every() {
			blank := ui.NewImage(w, h, palette.Background)
			img := ui.NewImage(w, h, palette.Background)
			ui.Fill(img, palette.Background)

			Of(name).Draw(img, box, r, palette)

			if painted := ui.Changed(blank, img); !box.Holds(painted) {
				t.Errorf("%s in %+v painted %+v, outside it", name, box, painted)
			}
		}
	}
}
