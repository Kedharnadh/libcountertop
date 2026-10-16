package visual

import (
	_ "embed"
	"image"
	"math"
	"math/rand/v2"

	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

//go:embed ember.glsl
var emberShader string

const (
	emberSparks = 70
	emberTop    = 0.6
	emberBed    = 0.56
)

func init() {
	register(WakeEmber, func() Visual { return &emberWake{} })
}

type cinder struct {
	x, speed, phase, size, sway float64
}

type emberWake struct {
	st     stage
	sparks []cinder
	label  string
	w, h   int
}

func (v *emberWake) Shade(g GL, fresh bool, in ui.Rect, x Input) error {
	if !v.st.measure(in) {
		return nil
	}
	if fresh {
		if err := g.Program(emberShader, Light); err != nil {
			return err
		}
	}
	if fresh || v.w != v.st.w || v.h != v.st.h {
		if err := g.Texture(1, coalBed(v.st.w, v.st.h)); err != nil {
			return err
		}
		v.scatter()
	}
	if fresh || x.Label != v.label || v.w != v.st.w || v.h != v.st.h {
		if err := g.Texture(0, wakeFace(v.st.w, v.st.h, x.Label)); err != nil {
			return err
		}
		v.label, v.w, v.h = x.Label, v.st.w, v.st.h
	}
	s := clamp01(x.Dawn)
	t := x.Now.Seconds()
	if err := g.Points(v.points(s, t)); err != nil {
		return err
	}
	return g.Values(v.values(s, t), 0.6, 0.03, 2)
}

func (v *emberWake) scatter() {
	r := rand.New(rand.NewPCG(3, 5))
	v.sparks = make([]cinder, emberSparks)
	for i := range v.sparks {
		v.sparks[i] = cinder{
			x:     r.Float64(),
			speed: 0.05 + r.Float64()*0.09,
			phase: r.Float64(),
			size:  2 + r.Float64()*3,
			sway:  r.Float64() * 40,
		}
	}
}

func (v *emberWake) points(s, t float64) []Point {
	W, H := float64(v.st.w), float64(v.st.h)
	lift := s * s * (3 - 2*s)
	dim := 0.3 + 0.7*smoothstep(0, 0.25, s)
	pts := make([]Point, 0, len(v.sparks))
	for _, sp := range v.sparks {
		climb := math.Mod(sp.phase+t*sp.speed, 1)
		y := emberTop + 0.05 - climb*(emberTop-0.05)
		fade := smoothstep(
			0.1,
			emberTop,
			y,
		) * (1 - smoothstep(0.85, 1, climb)) * (0.3 + 0.7*lift) * dim
		if fade < 0.02 {
			continue
		}
		a := float32(fade)
		pts = append(pts, Point{
			X:    float32((sp.x + math.Sin(t*1.1+sp.sway)*0.01) * W),
			Y:    float32(y * H),
			Size: float32(sp.size * v.st.u),
			R:    a, G: 0.62 * a, B: 0.25 * a, A: a,
			Round: true, Light: true,
		})
	}
	return pts
}

func (v *emberWake) values(s, t float64) []float32 {
	lift := s * s * (3 - 2*s)
	return []float32{
		float32(math.Mod(t, 3600)),
		float32(s),
		float32(9 + (2.2-9)*lift),
		float32(0.05 + 0.4*lift),
		float32(lift * lift * 0.22),
		float32(0.3 + lift),
		float32(0.3 + 0.7*smoothstep(0, 0.25, s)),
	}
}

func coalBed(w, h int) *image.RGBA {
	bw, bh := max(w/2, 2), max(int(float64(h)*(1-emberBed)/2), 2)
	aspect := float64(w) / float64(h)
	img := image.NewRGBA(image.Rect(0, 0, bw, bh))
	for y := range bh {
		uy := emberBed + (float64(y)+0.5)/float64(bh)*(1-emberBed)
		depth := smoothstep(emberTop-0.04, emberTop+0.12, uy)
		near := math.Max(uy-emberTop, 0)/(1-emberTop) + 0.12
		for x := range bw {
			px := (float64(x) + 0.5) / float64(bw) * aspect
			qx, qy := (px-0.5*aspect)/near*3.2, 2.6/near
			cell, edge, seed := coals(qx, qy)
			crust := emberFBM(qx*2.3+seed*9, qy*2.3+seed*9)
			inner := smoothstep(0.62, 0.05, cell)
			gap := smoothstep(0.02, 0.12, edge)
			body := inner * gap * (0.55 + 0.6*smoothstep(0.35, 0.75, crust)) / 1.15
			i := img.PixOffset(x, y)
			img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = unit8(
				body,
			), unit8(
				seed,
			), unit8(
				depth,
			), 255
		}
	}
	return img
}

func coals(qx, qy float64) (near, edge, seed float64) {
	ix, iy := math.Floor(qx), math.Floor(qy)
	fx, fy := qx-ix, qy-iy
	near, next := 8.0, 8.0
	ox, oy := ix, iy
	for dy := -1.0; dy <= 1; dy++ {
		for dx := -1.0; dx <= 1; dx++ {
			gx, gy := ix+dx, iy+dy
			jx, jy := shaderHash(gx+3.1, gy+3.1), shaderHash(gx+19.7, gy+19.7)
			d := math.Hypot(dx+jx-fx, dy+jy-fy)
			switch {
			case d < near:
				next, near, ox, oy = near, d, gx, gy
			case d < next:
				next = d
			}
		}
	}
	return near, next - near, shaderHash(ox+7.7, oy+7.7)
}

func emberFBM(x, y float64) float64 {
	v, amp := 0.0, 0.5
	for range 4 {
		v += amp * shaderNoise(x, y)
		x, y = x*2.02+5.3, y*2.02+11.7
		amp *= 0.5
	}
	return v
}
