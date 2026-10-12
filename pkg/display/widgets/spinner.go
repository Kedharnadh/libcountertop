package widgets

import (
	"math"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/interaction"
)

const (
	spinDots = 12
	SpinStep = 100 * time.Millisecond
)

func Spinner(vw, vh int, since time.Duration) gogui.View {
	r := float32(min(vw, vh)) / 14
	dot := max(r/5, 3)
	lead := int(since/SpinStep) % spinDots
	return gogui.DrawCanvas(gogui.DrawCanvasCfg{
		Width: float32(vw), Height: float32(vh),
		Version: uint64(lead),
		OnDraw: func(dc *gogui.DrawContext) {
			cx, cy := float32(vw)/2, float32(vh)/2
			for i := range spinDots {
				a := 2 * math.Pi * float64(i) / spinDots
				x := cx + r*float32(math.Sin(a))
				y := cy - r*float32(math.Cos(a))
				age := (lead - i + spinDots) % spinDots
				v := uint8(0xf0 - age*0xb0/spinDots)
				dc.FilledPolygon(
					interaction.Disc(x, y, dot),
					gogui.Color{R: v, G: v, B: v, A: 0xff},
				)
			}
		},
	})
}
