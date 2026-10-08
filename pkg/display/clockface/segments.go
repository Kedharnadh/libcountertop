package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() { register(Segments, segments{}) }

type segments struct{}

const (
	segmentsWidthShare  = 0.88
	segmentsHeightShare = 0.34
	segmentsRowShare    = 0.26
	segmentsRowGap      = 0.22
)

const (
	digitAspect = 0.56
	digitGap    = 0.16
	colonGap    = 0.30

	barThick = 0.15
	barNotch = 0.030

	segmentGhost = 0.09

	segmentsDateOf  = 0.13
	segmentsDateGap = 0.34
)

type segmentsCell struct {
	At ui.Rect
	Ch rune
}

type segmentsLayout struct {
	Cells []segmentsCell

	Date ui.Rect
}

func segmentsArrange(in ui.Rect, r Reading) (segmentsLayout, *ui.Font) {
	hour, minute, ok := lines(r)
	if !ok {
		return segmentsLayout{}, nil
	}

	rows := []string{hour + ":" + minute}
	if in.H > in.W {
		rows = []string{hour, minute}
	}

	height := segmentsFit(in, rows)

	width := int(float64(height) * digitAspect)
	gap := int(float64(height) * digitGap)
	between := int(float64(height) * segmentsRowGap)
	colon := int(float64(height) * colonGap)

	date := ui.MustLoad(ui.Regular, max(int(float64(height)*segmentsDateOf), 1))

	dateW, dateH, under := 0, 0, 0
	if r.Dated() {
		dateW, dateH = date.Measure(r.Date)
		under = int(float64(height) * segmentsDateGap)
	}

	tall := height*len(rows) + between*(len(rows)-1)
	top := in.Y + (in.H-(tall+under+dateH))/2

	var l segmentsLayout
	for _, row := range rows {
		x := in.X + (in.W-runWidth(row, width, gap, height))/2

		for _, c := range row {
			cell := width
			if c == ':' {
				cell = colon
			}

			l.Cells = append(l.Cells, segmentsCell{
				At: ui.Rect{X: x, Y: top, W: cell, H: height},
				Ch: c,
			})
			x += cell + gap
		}
		top += height + between
	}

	top -= between

	if r.Dated() {
		l.Date = ui.Rect{X: in.X + (in.W-dateW)/2, Y: top + under, W: dateW, H: dateH}
	}

	return l, date
}

func (segments) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	if _, _, ok := lines(r); !ok {
		plain{}.Draw(s, in, r, palette)
		return
	}

	l, date := segmentsArrange(in, r)

	on := palette.Text
	off := palette.Surface.Blend(palette.Text, segmentGhost)

	for _, c := range l.Cells {
		if c.Ch == ':' {
			drawColon(s, c.At.X, c.At.Y, c.At.H, on)
			continue
		}
		drawDigit(s, c.At, c.Ch, on, off)
	}

	if r.Dated() {
		ui.DrawText(s, date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}

func runWidth(text string, width, gap, height int) int {
	var w int
	for _, c := range text {
		if c == ':' {
			w += int(float64(height)*colonGap) + gap
			continue
		}
		w += width + gap
	}
	return w - gap
}

func segmentsFit(in ui.Rect, rows []string) int {
	share := segmentsHeightShare
	if len(rows) > 1 {
		share = segmentsRowShare
	}
	height := max(int(float64(in.H)*share), 1)

	var wide int
	for _, row := range rows {
		wide = max(wide, runWidth(row,
			int(float64(height)*digitAspect), int(float64(height)*digitGap), height))
	}

	if room := int(float64(in.W) * segmentsWidthShare); wide > room {
		height = height * room / wide
	}
	return max(height, 1)
}
