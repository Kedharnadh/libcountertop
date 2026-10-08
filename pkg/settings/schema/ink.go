package schema

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Ink string

const (
	InkTheme Ink = "theme"

	InkAmber  Ink = "amber"
	InkRed    Ink = "red"
	InkGreen  Ink = "green"
	InkCyan   Ink = "cyan"
	InkBlue   Ink = "blue"
	InkViolet Ink = "violet"
	InkPink   Ink = "pink"
)

const DefaultInk = InkTheme

var inks = map[Ink]theme.Color{
	InkAmber:  {R: 0xf2, G: 0xa6, B: 0x3b},
	InkRed:    {R: 0xe5, G: 0x54, B: 0x4b},
	InkGreen:  {R: 0x4c, G: 0xaf, B: 0x78},
	InkCyan:   {R: 0x35, G: 0xb8, B: 0xc4},
	InkBlue:   {R: 0x4f, G: 0x8e, B: 0xf7},
	InkViolet: {R: 0x9a, G: 0x7b, B: 0xf0},
	InkPink:   {R: 0xe8, G: 0x6a, B: 0xa6},
}

const mutedOfInk = 0.45

// WCAG AA body-text contrast floor: 4.5:1.
const inkFloor = 4.5

func readable(c theme.Color, t theme.Theme) theme.Color {
	const step = 0.05

	for by := 0.0; by < 1; by += step {
		got := c.Blend(t.Text, by)
		if theme.Contrast(got, t.Background) >= inkFloor {
			return got
		}
	}
	return t.Text
}

func (i Ink) Label() string {
	switch i {
	case InkTheme:
		return say.T("ink.theme")
	case InkAmber:
		return say.T("ink.amber")
	case InkRed:
		return say.T("ink.red")
	case InkGreen:
		return say.T("ink.green")
	case InkCyan:
		return say.T("ink.cyan")
	case InkBlue:
		return say.T("ink.blue")
	case InkViolet:
		return say.T("ink.violet")
	case InkPink:
		return say.T("ink.pink")
	}
	return string(i)
}

func (i Ink) Color(t theme.Theme) theme.Color {
	if c, ok := inks[i]; ok {
		return readable(c, t)
	}
	return t.Text
}

func (i Ink) Over(t theme.Theme) theme.Theme {
	if _, ok := inks[i]; !ok {
		return t
	}

	c := i.Color(t)

	t.Text = c
	t.Muted = c.Blend(t.Background, mutedOfInk)
	return t
}

func Inks() []Ink {
	return []Ink{InkTheme, InkAmber, InkRed, InkGreen, InkCyan, InkBlue, InkViolet, InkPink}
}
