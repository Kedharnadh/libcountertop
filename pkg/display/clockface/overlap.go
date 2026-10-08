package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() { register(Overlap, overlap{}) }

type overlap struct{}

const (
	overlapHeightShare = 0.46
	overlapWidthShare  = 0.78

	bite = 0.30

	shove = 0.18
)

const (
	overlapSuffixOfLine = 0.20
	overlapDateOfLine   = 0.13
	overlapDateGap      = 0.30
)

func (o overlap) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	hour, minute, ok := lines(r)
	if !ok {
		plain{}.Draw(s, in, r, palette)
		return
	}

	l, m := overlapArrange(in, hour, minute, r)

	ui.DrawText(s, m.line, l.Hour.X, l.Hour.Y, palette.Muted, palette.Background, hour)
	ui.DrawText(s, m.line, l.Minute.X, l.Minute.Y, palette.Text, palette.Background, minute)

	if r.Suffix != "" {
		ui.DrawText(s, m.small, l.Suffix.X, l.Suffix.Y,
			palette.Text, palette.Background, r.Suffix)
	}

	if r.Dated() {
		ui.DrawText(s, m.date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}

type overlapLayout struct {
	Hour   ui.Rect
	Minute ui.Rect

	Suffix ui.Rect
	Date   ui.Rect
}

func overlapArrange(in ui.Rect, hour, minute string, r Reading) (overlapLayout, overlapped) {
	m := layoutOverlap(in, hour, minute, r)

	wide := m.wide()

	left := in.X + (in.W-wide)/2
	top := in.Y + (in.H-m.tall())/2 - (m.ascent - m.digit)

	l := overlapLayout{
		Hour:   ui.Rect{X: left, Y: top, W: m.hourW, H: m.lineH},
		Minute: ui.Rect{X: left + wide - m.minuteW, Y: top + m.between, W: m.minuteW, H: m.lineH},
	}

	if r.Suffix != "" {
		l.Suffix = ui.Rect{
			X: left + m.hourW - m.suffixW,
			Y: top + (m.ascent - m.digit) - m.small.Descent(),
			W: m.suffixW,
			H: m.suffixH,
		}
	}

	if r.Dated() {
		l.Date = ui.Rect{
			X: max(in.X, min(left+wide-m.dateW, in.X+in.W-m.dateW)),
			Y: top + m.between + m.ascent + m.under,
			W: m.dateW,
			H: m.dateH,
		}
	}

	return l, m
}

type overlapped struct {
	line, small, date *ui.Font

	hourW, minuteW int
	suffixW        int
	dateW, dateH   int

	lineH   int
	suffixH int

	digit, ascent int

	between, under int

	across int
}

func (m overlapped) wide() int { return max(m.hourW, m.minuteW) + m.across }

func (m overlapped) tall() int { return m.digit + m.between + m.under + m.dateH }

func layoutOverlap(in ui.Rect, hour, minute string, r Reading) overlapped {
	maxW := int(float64(in.W) * overlapWidthShare)
	maxH := int(float64(in.H) * overlapHeightShare)

	size := min(
		fit(ui.Bold, hour, maxW, maxH),
		fit(ui.Bold, minute, maxW, maxH),
	)

	for range 8 {
		m := measureOverlap(size, hour, minute, r)

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
	return measureOverlap(size, hour, minute, r)
}

func measureOverlap(size int, hour, minute string, r Reading) overlapped {
	m := overlapped{
		line:  ui.MustLoad(ui.Bold, size),
		small: ui.MustLoad(ui.Medium, max(int(float64(size)*overlapSuffixOfLine), 1)),
		date:  ui.MustLoad(ui.Regular, max(int(float64(size)*overlapDateOfLine), 1)),
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

	m.between = max(m.digit-int(float64(m.digit)*bite), 1)
	m.across = int(float64(max(m.hourW, m.minuteW)) * shove)

	if r.Dated() {
		m.under = int(float64(m.digit) * overlapDateGap)
	}
	return m
}
