package boot

import (
	"image"
	"image/draw"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
)

type Moment = startup.Moment

type Reveal struct {
	logo          *image.RGBA
	bg            theme.Color
	dark          bool
	shader        string
	width, height float32
}

func NewReveal(palette theme.Theme, shader string, src image.Image, width, height float32) *Reveal {
	if src == nil {
		src = image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	logo := image.NewRGBA(src.Bounds())
	draw.Draw(logo, logo.Bounds(), src, src.Bounds().Min, draw.Src)
	return &Reveal{
		logo:   logo,
		bg:     palette.Background,
		dark:   theme.Dark(palette.Background),
		shader: shader,
		width:  max(width, 1),
		height: max(height, 1),
	}
}

func (r *Reveal) centred(w, h int) (float32, float32, float32) {
	s := min(float32(w)*0.92/r.width, float32(h)*0.62/r.height)
	return float32(w) / 2, float32(h) * 0.46, s
}

const LogoShare = 0.42

func (r *Reveal) header(w, h int) (float32, float32, float32) {
	logo, _ := Split(w, h)
	s := min(float32(logo.W)*0.86/r.width, float32(logo.H)*0.86/r.height)
	return float32(logo.X) + float32(logo.W)/2, float32(logo.Y) + float32(logo.H)/2, s
}

func (r *Reveal) Shade(g visual.GL, fresh bool, w, h int, m Moment) error {
	if fresh {
		if err := g.Program(r.shader, visual.Light); err != nil {
			return err
		}
		if err := g.Texture(0, r.logo); err != nil {
			return err
		}
	}
	dark := float32(0)
	if r.dark {
		dark = 1
	}
	cx, cy, cs := r.centred(w, h)
	hx, hy, hs := r.header(w, h)
	vals := []float32{
		float32(m.At.Seconds()), float32(m.Trace), float32(m.Header), float32(m.Ready), dark,
		float32(r.bg.R) / 255, float32(r.bg.G) / 255, float32(r.bg.B) / 255,
		cx, cy, cs, hx, hy, hs,
	}
	return g.Values(vals, 0, 0, 0)
}
