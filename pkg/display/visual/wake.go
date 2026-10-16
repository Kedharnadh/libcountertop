package visual

import (
	_ "embed"
	"image"
	"image/color"
	"math"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

const (
	WakeSunrise Kind = "wake-sunrise"
	WakeWhite   Kind = "wake-white"
	WakeEmber   Kind = "wake-ember"
)

//go:embed wakewhite.glsl
var wakeWhiteShader string

func init() {
	register(WakeWhite, func() Visual {
		return &wake{shader: wakeWhiteShader, amount: 0.35, radius: 0.06}
	})
}

type wake struct {
	st     stage
	shader string
	amount float32
	radius float32

	label string
	w, h  int
}

func (v *wake) Shade(g GL, fresh bool, in ui.Rect, x Input) error {
	if !v.st.measure(in) {
		return nil
	}
	if fresh {
		if err := g.Program(v.shader, Light); err != nil {
			return err
		}
	}
	if fresh || x.Label != v.label || v.w != v.st.w || v.h != v.st.h {
		if err := g.Texture(0, v.face(x.Label)); err != nil {
			return err
		}
		v.label, v.w, v.h = x.Label, v.st.w, v.st.h
	}
	share := math.Min(math.Max(x.Dawn, 0), 1)
	vals := []float32{
		float32(math.Mod(x.Now.Seconds(), 3600)),
		float32(share),
		float32(v.st.w) / float32(v.st.h),
	}
	return g.Values(vals, v.amount, v.radius, 2)
}

func (v *wake) face(clock string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, v.st.w, v.st.h))
	W, H := float64(v.st.w), float64(v.st.h)
	bold, _ := faces()
	size := math.Min(W, H) * 0.24
	letterWithin(img, bold, size, color.NRGBA{255, 255, 255, 255}, clock, W/2, H*0.17, W*0.8)
	return img
}
