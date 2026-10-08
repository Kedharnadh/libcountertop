package theme

import (
	"fmt"
	"math"
)

type Color struct{ R, G, B byte }

func rgb(v uint32) Color {
	return Color{R: byte(v >> 16), G: byte(v >> 8), B: byte(v)}
}

func (c Color) String() string { return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B) }

func (c Color) Blend(other Color, amount float64) Color {
	switch {
	case amount <= 0:
		return c
	case amount >= 1:
		return other
	}

	mix := func(a, b byte) byte { return byte(float64(a) + (float64(b)-float64(a))*amount) }
	return Color{mix(c.R, other.R), mix(c.G, other.G), mix(c.B, other.B)}
}

func (c Color) Mix(other Color, cover byte) Color {
	switch cover {
	case 0:
		return c
	case 0xff:
		return other
	}

	mix := func(a, b byte) byte {
		return byte((int(a)*(255-int(cover)) + int(b)*int(cover) + 127) / 255)
	}
	return Color{mix(c.R, other.R), mix(c.G, other.G), mix(c.B, other.B)}
}

type Theme struct {
	Name string
	Dark bool

	Background Color
	Surface    Color

	Text  Color
	Muted Color

	Accent  Color
	Accent2 Color

	Success Color
	Warning Color
	Danger  Color
}

var (
	ink   = rgb(0x0b0d10)
	paper = rgb(0xf5f7fa)
)

func (t Theme) Contrast(on Color) Color {
	if Contrast(ink, on) >= Contrast(paper, on) {
		return ink
	}
	return paper
}

// Contrast ratios run from 1 for identical colors to 21 for black on white.
func Contrast(a, b Color) float64 {
	la, lb := relative(a), relative(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relative(c Color) float64 {
	channel := func(v byte) float64 {
		s := float64(v) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c.R) + 0.7152*channel(c.G) + 0.0722*channel(c.B)
}

func luminance(c Color) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

func Dark(c Color) bool { return Contrast(paper, c) > Contrast(ink, c) }

func (t Theme) On(dark bool) Theme {
	if t.Dark == dark {
		return t
	}
	t.Dark = dark
	if dark {
		t.Background, t.Text = ink, paper
	} else {
		t.Background, t.Text = paper, ink
	}
	t.Surface = t.Background.Blend(t.Text, 0.08)
	t.Muted = t.Text.Blend(t.Background, 0.4)
	return t
}

func RGB(v uint32) Color { return rgb(v) }
