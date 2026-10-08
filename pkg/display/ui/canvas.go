package ui

import (
	"math"
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

type Surface interface {
	Set(x, y int, c theme.Color)
	Size() (w, h int)
}

type Clearer interface {
	ClearRect(r Rect)
}

func Clear(s Surface, r Rect) {
	if c, ok := s.(Clearer); ok {
		c.ClearRect(r)
		return
	}
	FillRect(s, r, theme.Color{})
}

type Compositor interface {
	Over(x, y int, c theme.Color, a byte)
}

func Over(s Surface, x, y int, c theme.Color, a byte) {
	if o, ok := s.(Compositor); ok {
		o.Over(x, y, c, a)
		return
	}
	if r, ok := s.(Reader); ok && a != 255 {
		d := r.At(x, y)
		keep := 255 - int(a)
		c.R = byte(min(int(c.R)+(int(d.R)*keep+127)/255, 255))
		c.G = byte(min(int(c.G)+(int(d.G)*keep+127)/255, 255))
		c.B = byte(min(int(c.B)+(int(d.B)*keep+127)/255, 255))
	}
	s.Set(x, y, c)
}

type Shader interface {
	Shade(r Rect, top, bottom byte)
}

func Shade(s Surface, r Rect, top, bottom byte) {
	if sh, ok := s.(Shader); ok {
		sh.Shade(r, top, bottom)
		return
	}
	FillRect(s, r, theme.Color{})
}

type Reader interface {
	At(x, y int) theme.Color
}

type Clipper interface {
	Clipped() Rect
}

func ClipOf(s Surface) Rect {
	if c, ok := s.(Clipper); ok {
		return c.Clipped()
	}
	return Rect{}
}

func (r Rect) Overlaps(o Rect) bool {
	if r.W <= 0 || r.H <= 0 || o.W <= 0 || o.H <= 0 {
		return true
	}
	return r.X < o.X+o.W && o.X < r.X+r.W && r.Y < o.Y+o.H && o.Y < r.Y+r.H
}

type Rect struct{ X, Y, W, H int }

func (r Rect) Inset(by int) Rect {
	return Rect{X: r.X + by, Y: r.Y + by, W: r.W - 2*by, H: r.H - 2*by}
}

func (r Rect) Center() (x, y int) { return r.X + r.W/2, r.Y + r.H/2 }

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

func Fill(s Surface, c theme.Color) {
	w, h := s.Size()
	FillRect(s, Rect{W: w, H: h}, c)
}

type Filler interface {
	FillRect(r Rect, c theme.Color)
}

func FillRect(s Surface, r Rect, c theme.Color) {
	if f, ok := s.(Filler); ok {
		f.FillRect(r, c)
		return
	}

	w, h := s.Size()

	for y := max(r.Y, 0); y < min(r.Y+r.H, h); y++ {
		for x := max(r.X, 0); x < min(r.X+r.W, w); x++ {
			s.Set(x, y, c)
		}
	}
}

func FillRounded(s Surface, r Rect, radius int, c theme.Color) {
	if radius <= 0 {
		FillRect(s, r, c)
		return
	}
	radius = min(radius, (min(r.W, r.H)-1)/2)

	FillRect(s, Rect{X: r.X, Y: r.Y + radius, W: r.W, H: r.H - 2*radius}, c)
	FillRect(s, Rect{X: r.X + radius, Y: r.Y, W: r.W - 2*radius, H: radius}, c)
	FillRect(s, Rect{X: r.X + radius, Y: r.Y + r.H - radius, W: r.W - 2*radius, H: radius}, c)

	corners := [][2]int{
		{r.X + radius, r.Y + radius},
		{r.X + r.W - radius - 1, r.Y + radius},
		{r.X + radius, r.Y + r.H - radius - 1},
		{r.X + r.W - radius - 1, r.Y + r.H - radius - 1},
	}
	for _, at := range corners {
		fillDisc(s, at[0], at[1], radius, c)
	}
}

func fillDisc(s Surface, cx, cy, radius int, c theme.Color) {
	under, soft := s.(Reader)

	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			d2 := dx*dx + dy*dy

			if d2 <= (radius-1)*(radius-1) {
				setClipped(s, cx+dx, cy+dy, c)
				continue
			}
			if d2 > (radius+1)*(radius+1) {
				continue
			}

			if !soft {
				if d2 <= radius*radius {
					setClipped(s, cx+dx, cy+dy, c)
				}
				continue
			}

			cover := 0.5 - (math.Sqrt(float64(d2)) - float64(radius))
			switch {
			case cover >= 1:
				setClipped(s, cx+dx, cy+dy, c)
			case cover > 0:
				setClipped(s, cx+dx, cy+dy, under.At(cx+dx, cy+dy).Blend(c, cover))
			}
		}
	}
}

type Point struct{ X, Y int }

func FillPolygon(s Surface, pts []Point, c theme.Color) {
	if len(pts) < 3 {
		return
	}

	w, h := s.Size()

	top, bottom := pts[0].Y, pts[0].Y
	for _, p := range pts[1:] {
		top = min(top, p.Y)
		bottom = max(bottom, p.Y)
	}

	var crossings []int
	for y := max(top, 0); y <= min(bottom, h-1); y++ {
		crossings = crossings[:0]

		for i, a := range pts {
			b := pts[(i+1)%len(pts)]
			if a.Y == b.Y {
				continue
			}

			if (y >= a.Y) == (y >= b.Y) {
				continue
			}
			crossings = append(crossings, a.X+(y-a.Y)*(b.X-a.X)/(b.Y-a.Y))
		}
		if len(crossings) < 2 {
			continue
		}
		slices.Sort(crossings)

		for i := 0; i+1 < len(crossings); i += 2 {
			for x := max(crossings[i], 0); x <= min(crossings[i+1], w-1); x++ {
				s.Set(x, y, c)
			}
		}
	}
}

func Border(s Surface, r Rect, thickness int, c theme.Color) {
	FillRect(s, Rect{X: r.X, Y: r.Y, W: r.W, H: thickness}, c)
	FillRect(s, Rect{X: r.X, Y: r.Y + r.H - thickness, W: r.W, H: thickness}, c)
	FillRect(s, Rect{X: r.X, Y: r.Y, W: thickness, H: r.H}, c)
	FillRect(s, Rect{X: r.X + r.W - thickness, Y: r.Y, W: thickness, H: r.H}, c)
}

func setClipped(s Surface, x, y int, c theme.Color) {
	w, h := s.Size()
	if x < 0 || y < 0 || x >= w || y >= h {
		return
	}
	s.Set(x, y, c)
}
