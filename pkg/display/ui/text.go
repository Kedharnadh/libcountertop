package ui

import (
	"fmt"
	"image"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gomedium"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

type Weight int

const (
	Regular Weight = iota
	Medium
	Bold
)

var faces = map[Weight][]byte{
	Regular: goregular.TTF,
	Medium:  gomedium.TTF,
	Bold:    gobold.TTF,
}

type Font struct {
	face   font.Face
	height int

	ascent  int
	descent int

	mu     sync.Mutex
	glyphs map[rune]glyph
}

type glyph struct {
	cover *image.Alpha

	at image.Point

	advance fixed.Int26_6
}

var (
	mu     sync.Mutex
	cached = map[key]*Font{}
	parsed = map[Weight]*opentype.Font{}
)

type key struct {
	weight Weight
	size   int
}

func Load(weight Weight, size int) (*Font, error) {
	mu.Lock()
	defer mu.Unlock()

	at := key{weight: weight, size: size}
	if f, ok := cached[at]; ok {
		return f, nil
	}

	ttf, ok := faces[weight]
	if !ok {
		return nil, fmt.Errorf("ui: no font for weight %d", weight)
	}

	if parsed[weight] == nil {
		p, err := opentype.Parse(ttf)
		if err != nil {
			return nil, fmt.Errorf("ui: parsing the font: %w", err)
		}
		parsed[weight] = p
	}

	face, err := opentype.NewFace(parsed[weight], &opentype.FaceOptions{
		Size: float64(size),
		DPI:  72,
	})
	if err != nil {
		return nil, fmt.Errorf("ui: sizing the font: %w", err)
	}

	metrics := face.Metrics()
	f := &Font{
		face:    face,
		height:  size,
		ascent:  metrics.Ascent.Round(),
		descent: metrics.Descent.Round(),
		glyphs:  map[rune]glyph{},
	}

	cached[at] = f
	return f, nil
}

func MustLoad(weight Weight, size int) *Font {
	f, err := Load(weight, size)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *Font) Measure(s string) (w, h int) {
	return f.width(s).Round(), f.ascent + f.descent
}

func (f *Font) Ascent() int { return f.ascent }

func (f *Font) Descent() int { return f.descent }

func (f *Font) width(s string) fixed.Int26_6 {
	var advance fixed.Int26_6

	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			advance += f.kern(prev, r)
		}
		advance += f.glyph(r).advance
		prev = r
	}
	return advance
}

func (f *Font) glyph(r rune) glyph {
	f.mu.Lock()
	defer f.mu.Unlock()

	if g, ok := f.glyphs[r]; ok {
		return g
	}

	var g glyph
	area, src, from, advance, ok := f.face.Glyph(fixed.P(0, 0), r)
	g.advance = advance

	if ok && !area.Empty() {
		g.at = area.Min
		g.cover = image.NewAlpha(image.Rect(0, 0, area.Dx(), area.Dy()))

		if a, isAlpha := src.(*image.Alpha); isAlpha {
			for y := range area.Dy() {
				at := a.PixOffset(from.X, from.Y+y)
				copy(g.cover.Pix[y*g.cover.Stride:][:area.Dx()], a.Pix[at:at+area.Dx()])
			}
		} else {
			for y := range area.Dy() {
				for x := range area.Dx() {
					_, _, _, alpha := src.At(from.X+x, from.Y+y).RGBA()
					g.cover.Pix[y*g.cover.Stride+x] = uint8(alpha >> 8)
				}
			}
		}
	}

	f.glyphs[r] = g
	return g
}

func (f *Font) kern(prev, r rune) fixed.Int26_6 {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.face.Kern(prev, r)
}

func DrawText(dst Surface, f *Font, x, y int, fg, bg theme.Color, s string) {
	clip := ClipOf(dst)
	if !clip.Overlaps(Rect{X: x, Y: y, W: f.width(s).Round(), H: f.ascent + f.descent}) {
		return
	}

	pen := fixed.I(x)
	base := y + f.ascent

	prev := rune(-1)
	for _, r := range s {
		if prev >= 0 {
			pen += f.kern(prev, r)
		}
		prev = r

		g := f.glyph(r)
		if g.cover != nil {
			blit(dst, g, pen.Round()+g.at.X, base+g.at.Y, fg, bg, clip)
		}
		pen += g.advance
	}
}

func blit(dst Surface, g glyph, x, y int, fg, bg theme.Color, clip Rect) {
	b := g.cover.Bounds()
	if !clip.Overlaps(Rect{X: x, Y: y, W: b.Dx(), H: b.Dy()}) {
		return
	}

	for iy := range b.Dy() {
		row := g.cover.Pix[iy*g.cover.Stride:]

		for ix := range b.Dx() {
			switch a := row[ix]; a {
			case 0:
				continue
			case 0xff:
				setClipped(dst, x+ix, y+iy, fg)
			default:
				setClipped(dst, x+ix, y+iy, bg.Mix(fg, a))
			}
		}
	}
}

func DrawTextIn(dst Surface, f *Font, r Rect, fg, bg theme.Color, s string) {
	w, h := f.Measure(s)
	DrawText(dst, f, r.X+(r.W-w)/2, r.Y+(r.H-h)/2, fg, bg, s)
}

func DrawTextRight(dst Surface, f *Font, r Rect, fg, bg theme.Color, s string) {
	w, h := f.Measure(s)
	DrawText(dst, f, r.X+r.W-w, r.Y+(r.H-h)/2, fg, bg, s)
}

func Wrap(f *Font, says string, width, most int) []string {
	if says == "" {
		return nil
	}
	if w, _ := f.Measure(says); w <= width {
		return []string{says}
	}

	out := make([]string, 0, most)
	rest := says

	for len(out) < most-1 {
		cut := breaks(f, rest, width)
		if cut <= 0 {
			break
		}

		out = append(out, rest[:cut])
		rest = strings.TrimLeft(rest[cut:], " ")

		if w, _ := f.Measure(rest); w <= width {
			break
		}
	}
	return append(out, Fit(f, rest, width))
}

func breaks(f *Font, says string, width int) int {
	last := 0
	for i, r := range says {
		if r != ' ' {
			continue
		}
		if w, _ := f.Measure(says[:i]); w > width {
			break
		}
		last = i
	}
	return last
}

func Fit(f *Font, says string, width int) string {
	if w, _ := f.Measure(says); w <= width {
		return says
	}

	runes := []rune(says)
	for len(runes) > 1 {
		runes = runes[:len(runes)-1]
		if w, _ := f.Measure(string(runes) + "…"); w <= width {
			return string(runes) + "…"
		}
	}
	return "…"
}
