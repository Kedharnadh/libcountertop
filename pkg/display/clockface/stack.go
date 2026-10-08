package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() { register(Stack, stack{}) }

type stack struct{}

const (
	lineHeightShare = 0.34
	lineWidthShare  = 0.80
)

const (
	suffixOfLine = 0.22
	dateOfLine   = 0.13

	lineGap = 0.06
	dateGap = 0.34

	suffixGapOfLine = 0.18
)

type stacked struct {
	line, small, date *ui.Font

	hourW, minuteW int
	suffixW        int
	dateW, dateH   int

	lineH   int
	suffixH int

	digit, ascent int

	between, under int
}

func (s stacked) tall() int { return s.digit + s.between + s.under + s.dateH }

func (s stacked) wide() int {
	return max(s.hourW+2*(s.gap()+s.suffixW), s.minuteW)
}

func (s stacked) gap() int {
	if s.suffixW == 0 {
		return 0
	}
	return int(float64(s.digit) * suffixGapOfLine)
}

func (stack) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	hour, minute, ok := lines(r)
	if !ok {
		plain{}.Draw(s, in, r, palette)
		return
	}

	l, m := stackArrange(in, hour, minute, r)

	ui.DrawText(s, m.line, l.Hour.X, l.Hour.Y, palette.Text, palette.Background, hour)
	ui.DrawText(s, m.line, l.Minute.X, l.Minute.Y, palette.Text, palette.Background, minute)

	if r.Suffix != "" {
		ui.DrawText(s, m.small, l.Suffix.X, l.Suffix.Y,
			palette.Muted, palette.Background, r.Suffix)
	}

	if r.Dated() {
		ui.DrawText(s, m.date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}

type stackLayout struct {
	Hour   ui.Rect
	Minute ui.Rect

	Suffix ui.Rect
	Date   ui.Rect
}

func stackArrange(in ui.Rect, hour, minute string, r Reading) (stackLayout, stacked) {
	m := layoutStack(in, hour, minute, r)

	left := in.X + (in.W-max(m.hourW, m.minuteW))/2

	top := in.Y + (in.H-m.tall())/2 - (m.ascent - m.digit)

	l := stackLayout{
		Hour:   ui.Rect{X: left, Y: top, W: m.hourW, H: m.lineH},
		Minute: ui.Rect{X: left, Y: top + m.between, W: m.minuteW, H: m.lineH},
	}

	if r.Suffix != "" {
		l.Suffix = ui.Rect{
			X: left + m.hourW + m.gap(),
			Y: top + (m.ascent - m.digit) - m.small.Descent(),
			W: m.suffixW,
			H: m.suffixH,
		}
	}

	if r.Dated() {
		l.Date = ui.Rect{
			X: max(in.X, min(left, in.X+in.W-m.dateW)),
			Y: top + m.between + m.ascent + m.under,
			W: m.dateW,
			H: m.dateH,
		}
	}

	return l, m
}

func layoutStack(in ui.Rect, hour, minute string, r Reading) stacked {
	maxW := int(float64(in.W) * lineWidthShare)
	maxH := int(float64(in.H) * lineHeightShare)

	size := min(
		fit(ui.Bold, hour, maxW, maxH),
		fit(ui.Bold, minute, maxW, maxH),
	)

	for range 8 {
		m := measureStack(size, hour, minute, r)

		next := size
		if m.wide() > maxW {
			next = min(next, size*maxW/m.wide())
		}
		if m.tall() > in.H {
			next = min(next, size*in.H/m.tall())
		}

		if next == size || size <= 1 {
			return m
		}
		if next >= size {
			next = size - 1
		}
		size = max(next, 1)
	}
	return measureStack(size, hour, minute, r)
}

func measureStack(size int, hour, minute string, r Reading) stacked {
	m := stacked{
		line:  ui.MustLoad(ui.Bold, size),
		small: ui.MustLoad(ui.Medium, max(int(float64(size)*suffixOfLine), 1)),
		date:  ui.MustLoad(ui.Regular, max(int(float64(size)*dateOfLine), 1)),
	}

	m.hourW, m.lineH = m.line.Measure(hour)
	m.minuteW, _ = m.line.Measure(minute)

	if r.Dated() {
		m.dateW, m.dateH = m.date.Measure(r.Date)
	}

	if r.Suffix != "" {
		m.suffixW, m.suffixH = m.small.Measure(r.Suffix)
	}

	m.ascent = m.line.Ascent()
	m.digit = max(m.ascent-m.line.Descent(), 1)

	m.between = m.digit + int(float64(m.digit)*lineGap)
	if r.Dated() {
		m.under = int(float64(m.digit) * dateGap)
	}

	return m
}
