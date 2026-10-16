package visual

import (
	_ "embed"
	"image"
	"math"
	"math/rand/v2"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

//go:embed sunrise.glsl
var sunriseShader string

const (
	sunriseStars = 160
	sunriseRays  = 11
	sunRadius    = 0.07
)

func init() {
	register(WakeSunrise, func() Visual { return &sunrise{} })
}

type rgb [3]float64

type dawnStar struct {
	x, y, size, rate, phase float64
}

type sunrise struct {
	st    stage
	stars []dawnStar
	label string
	w, h  int
}

func (v *sunrise) Shade(g GL, fresh bool, in ui.Rect, x Input) error {
	if !v.st.measure(in) {
		return nil
	}
	if fresh {
		if err := g.Program(sunriseShader, Light); err != nil {
			return err
		}
		for unit, img := range sunriseMaps() {
			if err := g.Texture(unit, img); err != nil {
				return err
			}
		}
	}
	if fresh || x.Label != v.label || v.w != v.st.w || v.h != v.st.h {
		if err := g.Texture(0, wakeFace(v.st.w, v.st.h, x.Label)); err != nil {
			return err
		}
		v.label, v.w, v.h = x.Label, v.st.w, v.st.h
		v.scatter()
	}
	s := clamp01(x.Dawn)
	t := x.Now.Seconds()
	if err := g.Points(v.points(s, t)); err != nil {
		return err
	}
	return g.Values(v.values(s, t), 0.5, 0.04, 2)
}

func (v *sunrise) scatter() {
	r := rand.New(rand.NewPCG(7, 11))
	v.stars = make([]dawnStar, sunriseStars)
	for i := range v.stars {
		v.stars[i] = dawnStar{
			x:     r.Float64(),
			y:     r.Float64() * 0.7,
			size:  1.2 + r.Float64()*1.6,
			rate:  1.2 + r.Float64()*3,
			phase: r.Float64() * 50,
		}
	}
}

func (v *sunrise) points(s, t float64) []Point {
	W, H := float64(v.st.w), float64(v.st.h)
	dim := 0.18 + 0.82*smoothstep(0, 0.3, s)
	fade := (1 - smoothstep(0.05, 0.4, s)) * dim
	pts := make([]Point, 0, len(v.stars)+sunriseRays*4)
	if fade > 0.01 {
		for _, st := range v.stars {
			tw := 0.55 + 0.45*math.Sin(t*st.rate+st.phase)
			a := float32(tw * fade * smoothstep(0.1, 0.5, 1-st.y))
			pts = append(pts, Point{
				X: float32(st.x * W), Y: float32(st.y * H), Size: float32(st.size * v.st.u),
				R: 0.85 * a, G: 0.9 * a, B: a, A: a, Round: true,
			})
		}
	}
	lit := smoothstep(0, 0.5, s)
	if lit > 0.01 {
		cx, cy := W/2, sunHeight(s)*H
		core := sunCore(s)
		for i := range sunriseRays {
			ang := float64(i)*2*math.Pi/sunriseRays + t*0.03/sunriseRays
			for k := range 4 {
				r := H * (sunRadius + 0.05 + 0.07*float64(k))
				a := 0.05 * lit * dim * (1 - float64(k)/4)
				pts = append(pts, Point{
					X:    float32(cx + math.Cos(ang)*r),
					Y:    float32(cy + math.Sin(ang)*r),
					Size: float32(H * (0.05 - 0.008*float64(k))),
					R:    float32(core[0] * a), G: float32(core[1] * a), B: float32(core[2] * a),
					A: float32(a), Round: true, Light: true,
				})
			}
		}
	}
	return pts
}

func (v *sunrise) values(s, t float64) []float32 {
	lit := smoothstep(0, 0.5, s)
	low := ramp4(
		rgb{0.012, 0.01, 0.035},
		rgb{0.3, 0.08, 0.2},
		rgb{0.85, 0.36, 0.18},
		rgb{0.95, 0.72, 0.5},
		s,
	)
	zenith := ramp4(
		rgb{0.004, 0.006, 0.03},
		rgb{0.04, 0.05, 0.16},
		rgb{0.14, 0.24, 0.5},
		rgb{0.24, 0.46, 0.8},
		s,
	)
	warm := mixRGB(rgb{1, 0.36, 0.1}, rgb{1, 0.8, 0.55}, s)
	dark := mixRGB(rgb{0.015, 0.015, 0.04}, rgb{0.3, 0.25, 0.38}, s)
	lift := mixRGB(rgb{0.5, 0.18, 0.2}, mixRGB(rgb{1, 0.6, 0.42}, rgb{1, 0.96, 0.92}, s), lit)
	haze := mixRGB(rgb{0.03, 0.02, 0.07}, rgb{0.42, 0.3, 0.36}, lit)
	ground := mixRGB(rgb{0.004, 0.003, 0.01}, rgb{0.07, 0.05, 0.08}, lit)
	halo := lit * (1 - 0.7*smoothstep(0.5, 1, s))
	u := []float32{
		float32(math.Mod(t*0.0012, 1)),
		float32(s),
		float32(v.st.w) / float32(v.st.h),
		float32(sunHeight(s)),
		float32(halo),
		float32(lit * (1 - s) * 0.45),
		float32(0.18 + 0.82*smoothstep(0, 0.3, s)),
	}
	for _, c := range []rgb{low, zenith, warm, dark, lift, sunCore(s), haze, ground} {
		u = append(u, float32(c[0]), float32(c[1]), float32(c[2]))
	}
	return u
}

func sunHeight(s float64) float64 {
	rise := s * s * (3 - 2*s)
	return 1.06 + (0.5-1.06)*rise
}

func sunCore(s float64) rgb {
	return mixRGB(rgb{1, 0.36, 0.08}, rgb{1, 0.95, 0.82}, smoothstep(0.05, 0.8, s))
}

func ramp4(a, b, c, d rgb, s float64) rgb {
	s = clamp01(s) * 3
	switch {
	case s < 1:
		return mixRGB(a, b, s)
	case s < 2:
		return mixRGB(b, c, s-1)
	}
	return mixRGB(c, d, s-2)
}

func mixRGB(a, b rgb, k float64) rgb {
	return rgb{a[0] + (b[0]-a[0])*k, a[1] + (b[1]-a[1])*k, a[2] + (b[2]-a[2])*k}
}

func wakeFace(w, h int, clock string) *image.RGBA {
	return (&wake{st: stage{w: w, h: h}}).face(clock)
}
