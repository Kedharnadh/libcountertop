package ui

import "testing"

func TestARoundedRectangleStaysInsideItself(t *testing.T) {
	for _, size := range [][2]int{
		{10, 168}, {168, 10}, {20, 20}, {21, 21}, {40, 8}, {8, 40}, {2, 2}, {1, 9}, {393, 168},
	} {
		w, h := size[0], size[1]

		for _, radius := range []int{1, 2, 3, 5, h / 2, w / 2, max(w, h)} {
			if radius <= 0 {
				continue
			}

			const pad = 8
			img := NewImage(w+2*pad, h+2*pad, black)
			blank := NewImage(w+2*pad, h+2*pad, black)

			r := Rect{X: pad, Y: pad, W: w, H: h}
			FillRounded(img, r, radius, white)

			if painted := Changed(blank, img); !r.Holds(painted) {
				t.Errorf("%dx%d radius %d painted %+v, outside %+v", w, h, radius, painted, r)
			}
		}
	}
}

func TestARoundedRectangleStillFillsItsMiddle(t *testing.T) {
	img := NewImage(60, 60, black)
	FillRounded(img, Rect{X: 10, Y: 10, W: 40, H: 40}, 20, white)

	if got := img.At(30, 30); got != white {
		t.Errorf("the middle is %v, want it filled", got)
	}
}
