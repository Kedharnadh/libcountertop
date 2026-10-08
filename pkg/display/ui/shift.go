package ui

import (
	"image"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func Shift(s Surface, dx, dy int) Surface { return shifted{to: s, dx: dx, dy: dy} }

type shifted struct {
	to     Surface
	dx, dy int
}

func (s shifted) Size() (w, h int) { return s.to.Size() }

func (s shifted) Set(x, y int, c theme.Color) { s.to.Set(x+s.dx, y+s.dy, c) }

func (s shifted) DrawRGBA(x, y int, img *image.RGBA, scale int, clip Rect) {
	if clip.W > 0 && clip.H > 0 {
		clip.X, clip.Y = clip.X+s.dx, clip.Y+s.dy
	}
	DrawRGBA(s.to, x+s.dx, y+s.dy, img, scale, clip)
}

func (s shifted) At(x, y int) theme.Color {
	r, ok := s.to.(Reader)
	if !ok {
		return theme.Color{}
	}
	return r.At(x+s.dx, y+s.dy)
}

func (s shifted) FillRect(r Rect, c theme.Color) {
	FillRect(s.to, Rect{X: r.X + s.dx, Y: r.Y + s.dy, W: r.W, H: r.H}, c)
}

func (s shifted) Clipped() Rect {
	c := ClipOf(s.to)
	if c.W <= 0 || c.H <= 0 {
		w, h := s.to.Size()
		c = Rect{W: w, H: h}
	}
	return Rect{X: c.X - s.dx, Y: c.Y - s.dy, W: c.W, H: c.H}
}
