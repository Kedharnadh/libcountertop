package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() { register(Cards, cards{}) }

type cards struct{}

const (
	cardsWidthShare  = 0.88
	cardsHeightShare = 0.74
)

const (
	cardRound    = 0.11
	cardGap      = 0.055
	digitsOfCard = 0.54
	digitsWide   = 0.74

	seamThick = 0.007

	suffixOfCard = 0.15
	suffixInset  = 0.09

	dateOfCard    = 0.085
	dateGapOfCard = 0.16
)

type cardsLayout struct {
	First  ui.Rect
	Second ui.Rect

	Date ui.Rect

	Side int
}

func cardsArrange(in ui.Rect, r Reading) (cardsLayout, *ui.Font) {
	across := in.W >= in.H

	side, gap := cardsFit(in, across)
	date := ui.MustLoad(ui.Regular, max(int(float64(side)*dateOfCard), 1))

	dateW, dateH, under := 0, 0, 0
	if r.Dated() {
		dateW, dateH = date.Measure(r.Date)
		under = int(float64(side) * dateGapOfCard)
	}

	blockW, blockH := side, side*2+gap
	if across {
		blockW, blockH = side*2+gap, side
	}

	left := in.X + (in.W-blockW)/2
	top := in.Y + (in.H-(blockH+under+dateH))/2

	l := cardsLayout{
		First:  ui.Rect{X: left, Y: top, W: side, H: side},
		Second: ui.Rect{X: left, Y: top + side + gap, W: side, H: side},
		Side:   side,
	}
	if across {
		l.Second = ui.Rect{X: left + side + gap, Y: top, W: side, H: side}
	}

	if r.Dated() {
		l.Date = ui.Rect{
			X: in.X + (in.W-dateW)/2,
			Y: top + blockH + under,
			W: dateW,
			H: dateH,
		}
	}

	return l, date
}

func (cards) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	hour, minute, ok := lines(r)
	if !ok {
		plain{}.Draw(s, in, r, palette)
		return
	}

	l, date := cardsArrange(in, r)

	drawCard(s, l.First, hour, palette)
	drawCard(s, l.Second, minute, palette)

	if r.Suffix != "" {
		small := ui.MustLoad(ui.Medium, max(int(float64(l.Side)*suffixOfCard), 1))
		inset := int(float64(l.Side) * suffixInset)

		w, _ := small.Measure(r.Suffix)
		ui.DrawText(s, small, l.First.X+l.First.W-inset-w, l.First.Y+inset,
			palette.Muted, palette.Surface, r.Suffix)
	}

	if r.Dated() {
		ui.DrawText(s, date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}

func drawCard(s ui.Surface, at ui.Rect, text string, palette theme.Theme) {
	ui.FillRounded(s, at, int(float64(at.W)*cardRound), palette.Surface)

	thick := max(int(float64(at.H)*seamThick), 1)
	seam := palette.Surface.Blend(palette.Text, 0.10)

	ui.FillRect(s, ui.Rect{X: at.X, Y: at.Y + (at.H-thick)/2, W: at.W, H: thick}, seam)

	font := ui.MustLoad(ui.Bold, fit(ui.Bold, text,
		int(float64(at.W)*digitsWide), int(float64(at.H)*digitsOfCard)))

	w, _ := font.Measure(text)
	digits := font.Ascent() - font.Descent()

	ui.DrawText(s, font, at.X+(at.W-w)/2, at.Y+(at.H-digits)/2-(font.Ascent()-digits),
		palette.Text, palette.Surface, text)
}

func cardsFit(in ui.Rect, across bool) (side, gap int) {
	tall := int(float64(in.H) * cardsHeightShare)
	wide := int(float64(in.W) * cardsWidthShare)

	if across {
		side = min(tall, int(float64(wide)/(2+cardGap)))
	} else {
		side = min(wide, int(float64(tall)/(2+cardGap)))
	}

	side = max(side, 1)
	return side, int(float64(side) * cardGap)
}
