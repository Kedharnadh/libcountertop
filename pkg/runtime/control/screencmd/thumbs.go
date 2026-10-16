package screencmd

import (
	"image"
	"math"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/analysis"
	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
)

const (
	ThumbFrames = 50
	ThumbStep   = 40 * time.Millisecond
)

func Render(r *gpu.Renderer, k visual.Kind, w, h, frames int, label string) (*image.RGBA, error) {
	return shade(r, k, w, h, frames, visual.Input{Mic: Loud(), Speaker: Loud(), Label: label})
}

func RenderWake(
	r *gpu.Renderer,
	k visual.Kind,
	w, h int,
	dawn float64,
	label string,
) (*image.RGBA, error) {
	return shade(r, k, w, h, 4, visual.Input{Label: label, Dawn: dawn})
}

func shade(r *gpu.Renderer, k visual.Kind, w, h, frames int, x visual.Input) (*image.RGBA, error) {
	l, err := r.OpenOffscreen(w, h)
	if err != nil {
		return nil, err
	}
	defer l.Close()
	v := visual.New(k)
	x.Dt = ThumbStep
	for i := range frames {
		x.Now = time.Duration(i+1) * ThumbStep
		x.Replying = i >= frames/2
		if err := v.Shade(l, i == 0, ui.Rect{W: w, H: h}, x); err != nil {
			return nil, err
		}
		time.Sleep(ThumbStep)
	}
	return l.Read()
}

func Loud() analysis.Analysis {
	var a analysis.Analysis
	a.Level, a.Peak, a.Onsets = 0.4, 0.7, 3
	for b := range a.Bands {
		a.Bands[b] = float32(0.35 + 0.3*math.Sin(float64(b)*0.55) + 0.25*math.Cos(float64(b)*0.21))
	}
	for i := range a.Wave {
		p := 2 * math.Pi * float64(i) / float64(len(a.Wave))
		a.Wave[i] = float32(0.55*math.Sin(3*p) + 0.3*math.Sin(7*p+1) + 0.1*math.Sin(13*p))
	}
	return a
}

func Shrink(src *image.RGBA, tw, th int) *image.RGBA {
	w, h := src.Bounds().Dx(), src.Bounds().Dy()
	out := image.NewRGBA(image.Rect(0, 0, tw, th))
	for y := range th {
		y0, y1 := y*h/th, (y+1)*h/th
		for x := range tw {
			x0, x1 := x*w/tw, (x+1)*w/tw
			var r, g, b, n int
			for sy := y0; sy < y1; sy++ {
				row := src.Pix[sy*src.Stride:]
				for sx := x0; sx < x1; sx++ {
					r, g, b, n = r+int(row[sx*4]), g+int(row[sx*4+1]), b+int(row[sx*4+2]), n+1
				}
			}
			o := out.PixOffset(x, y)
			out.Pix[o], out.Pix[o+1], out.Pix[o+2], out.Pix[o+3] = uint8(
				r/n,
			), uint8(
				g/n,
			), uint8(
				b/n,
			), 255
		}
	}
	return out
}
