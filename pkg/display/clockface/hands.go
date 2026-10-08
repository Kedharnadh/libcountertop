package clockface

import (
	"math"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func hand(cx, cy, length, width int, deg float64) []ui.Point {
	dx, dy := heading(deg)
	px, py := -dy, dx

	tipX, tipY := float64(cx)+dx*float64(length), float64(cy)+dy*float64(length)

	tail := float64(length) * handTail
	tailX, tailY := float64(cx)-dx*tail, float64(cy)-dy*tail

	half := float64(width) / 2
	tip := half * 0.55

	return []ui.Point{
		{X: int(tailX + px*half), Y: int(tailY + py*half)},
		{X: int(tipX + px*tip), Y: int(tipY + py*tip)},
		{X: int(tipX - px*tip), Y: int(tipY - py*tip)},
		{X: int(tailX - px*half), Y: int(tailY - py*half)},
	}
}

func numerals(s ui.Surface, cx, cy, dial int, palette theme.Theme) {
	font := ui.MustLoad(ui.Bold, max(int(float64(dial)*numeralOfDial), 1))
	radius := float64(dial) * numeralRadius

	digits := font.Ascent() - font.Descent()

	for i, text := range []string{"12", "3", "6", "9"} {
		dx, dy := heading(float64(i) * 90)

		x := float64(cx) + radius*dx
		y := float64(cy) + radius*dy

		w, _ := font.Measure(text)
		ui.DrawText(s, font, int(x)-w/2, int(y)-digits/2-(font.Ascent()-digits),
			palette.Text, palette.Background, text)
	}
}

func heading(deg float64) (dx, dy float64) {
	rad := deg * math.Pi / 180
	return math.Sin(rad), -math.Cos(rad)
}
