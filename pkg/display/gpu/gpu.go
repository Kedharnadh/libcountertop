package gpu

import (
	"errors"
	"image"
	"sync/atomic"

	"github.com/ygelfand/libcountertop/pkg/display/surface"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
)

const z = -1

type Renderer struct {
	options Options
	ids     atomic.Uint32
}
type Options struct {
	Helper func() *surface.Client
	Place  func(ui.Rect, int, int) (float32, float32, surface.Matrix)
}

func New(options Options) *Renderer { return &Renderer{options: options} }

var ErrNoHelper = errors.New("gpu: the display helper is not connected")

type Layer struct {
	owner  *Renderer
	closed atomic.Bool
	c      *surface.Client
	id     uint32
	z      int
	W, H   int
}

func (r *Renderer) Open(w, h int) (*Layer, error) { return r.OpenAt(w, h, z) }

func (r *Renderer) OpenAt(w, h, depth int) (*Layer, error) {
	c := r.options.Helper()
	if c == nil || c.Err() != nil {
		return nil, ErrNoHelper
	}
	id := 2000 + r.ids.Add(1)
	if err := c.GLOpen(id, w, h, depth); err != nil {
		if c.Err() != nil {
			return nil, ErrNoHelper
		}
		return nil, err
	}
	return &Layer{owner: r, c: c, id: id, z: depth, W: w, H: h}, nil
}

func (r *Renderer) OpenOffscreen(w, h int) (*Layer, error) {
	c := r.options.Helper()
	if c == nil || c.Err() != nil {
		return nil, ErrNoHelper
	}
	id := 2000 + r.ids.Add(1)
	if err := c.GLOffscreen(id, w, h); err != nil {
		if c.Err() != nil {
			return nil, ErrNoHelper
		}
		return nil, err
	}
	return &Layer{owner: r, c: c, id: id, W: w, H: h}, nil
}

func (l *Layer) Read() (*image.RGBA, error) {
	pix, w, h, err := l.c.GLRead(l.id)
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		copy(img.Pix[y*img.Stride:(y+1)*img.Stride], pix[(h-1-y)*w*4:(h-y)*w*4])
	}
	return img, nil
}

func (l *Layer) Alive() bool {
	return !l.closed.Load() && l.owner.options.Helper() == l.c && l.c.Err() == nil
}

func (l *Layer) Texture(unit int, img *image.RGBA) error {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	pix := img.Pix
	if img.Stride != w*4 || b.Min != (image.Point{}) {
		pix = make([]byte, w*h*4)
		for y := range h {
			o := img.PixOffset(b.Min.X, b.Min.Y+y)
			copy(pix[y*w*4:(y+1)*w*4], img.Pix[o:])
		}
	}
	return l.c.GLTexture(l.id, unit, w, h, pix[:w*h*4])
}

func (l *Layer) Program(src string, passes visual.Passes) error {
	return l.c.GLProgram(l.id, uint32(passes), src)
}

func (l *Layer) Values(u []float32, amount, radius float32, passes int) error {
	return l.c.GLValues(l.id, u, surface.Glow{Amount: amount, Radius: radius, Passes: passes})
}

func (l *Layer) Points(p []visual.Point) error {
	pts := make([]float32, 0, len(p)*surface.PointFloats)
	for _, q := range p {
		flags := float32(0)
		if q.Feed {
			flags++
		}
		if q.Round {
			flags += 2
		}
		if q.Light {
			flags += 4
		}
		if q.Splat {
			flags += 8
		}
		if q.Disc {
			flags += 16
		}
		if q.Over {
			flags += 32
		}
		pts = append(pts, q.X, q.Y, q.Size, q.R, q.G, q.B, q.A, flags)
	}
	return l.c.GLPoints(l.id, pts)
}

func (l *Layer) Lines(s []visual.Segment) error {
	segs := make([]float32, 0, len(s)*surface.LineFloats)
	for _, q := range s {
		segs = append(segs, q.X0, q.Y0, q.X1, q.Y1, q.Width, q.R, q.G, q.B, q.A)
	}
	return l.c.GLLines(l.id, segs)
}

func (l *Layer) Quads(q []visual.Quad) error {
	quads := make([]float32, 0, len(q)*surface.QuadFloats)
	for _, v := range q {
		quads = append(quads, v.X[0], v.Y[0], v.X[1], v.Y[1], v.X[2], v.Y[2], v.X[3], v.Y[3], v.R, v.G, v.B, v.A)
	}
	return l.c.GLQuads(l.id, quads)
}

func (l *Layer) Place(at ui.Rect) error {
	x, y, m := l.owner.options.Place(at, l.W, l.H)
	return l.c.VideoPlace(l.id, x, y, m, l.z, true)
}

func (l *Layer) Close() error {
	if !l.closed.CompareAndSwap(false, true) {
		return nil
	}
	return l.c.Destroy(l.id)
}
