package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() {
	register(Analog, analog{})
	register(AnalogSeconds, analog{seconds: true})
}

type analog struct {
	seconds bool
}

func (a analog) Ticks() bool { return a.seconds }

const dialShare = 0.78

const (
	numeralOfDial = 0.25
	numeralRadius = 0.36

	hourHandOfDial   = 0.28
	minuteHandOfDial = 0.44
	secondHandOfDial = 0.47
	handWidthOfDial  = 0.058
	handTail         = 0.16
	capOfDial        = 0.075

	minuteOfHand = 0.70
	secondOfHand = 0.28

	analogDateOfDial = 0.075
	analogDateGap    = 0.10
)

type analogLayout struct {
	Hour, Minute, Second []ui.Point

	Cap ui.Rect

	Date ui.Rect

	CX, CY, Dial int
}

func (a analog) arrange(in ui.Rect, r Reading) (analogLayout, *ui.Font, bool) {
	h, m, ok := clock(r)
	if !ok {
		return analogLayout{}, nil, false
	}

	dial := int(float64(min(in.W, in.H)) * dialShare)
	date := ui.MustLoad(ui.Regular, max(int(float64(dial)*analogDateOfDial), 1))

	dateW, dateH, under := 0, 0, 0
	if r.Dated() {
		dateW, dateH = date.Measure(r.Date)
		under = int(float64(dial) * analogDateGap)
	}

	cx := in.X + in.W/2
	cy := in.Y + (in.H-(dial+under+dateH))/2 + dial/2

	turn := float64(h%12)*30 + float64(m)*0.5
	width := float64(dial) * handWidthOfDial

	pin := max(int(float64(dial)*capOfDial), 2)

	l := analogLayout{
		Hour: hand(cx, cy, int(float64(dial)*hourHandOfDial), int(width), turn),
		Minute: hand(cx, cy, int(float64(dial)*minuteHandOfDial),
			int(width*minuteOfHand), float64(m)*6),
		Cap:  ui.Rect{X: cx - pin/2, Y: cy - pin/2, W: pin, H: pin},
		CX:   cx,
		CY:   cy,
		Dial: dial,
	}

	if a.seconds {
		l.Second = hand(cx, cy, int(float64(dial)*secondHandOfDial),
			int(width*secondOfHand), float64(r.Second)*6)
	}

	if r.Dated() {
		l.Date = ui.Rect{X: in.X + (in.W-dateW)/2, Y: cy + dial/2 + under, W: dateW, H: dateH}
	}

	return l, date, true
}

func (a analog) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	l, date, ok := a.arrange(in, r)
	if !ok {
		plain{}.Draw(s, in, r, palette)
		return
	}

	numerals(s, l.CX, l.CY, l.Dial, palette)

	ui.FillPolygon(s, l.Hour, palette.Text)
	ui.FillPolygon(s, l.Minute, palette.Text)

	if l.Second != nil {
		ui.FillPolygon(s, l.Second, palette.Accent)
	}

	ui.FillRounded(s, l.Cap, l.Cap.W/2, palette.Text)

	if r.Dated() {
		ui.DrawText(s, date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}
