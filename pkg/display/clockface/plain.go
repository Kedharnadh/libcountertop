package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func init() { register(Plain, plain{}) }

type plain struct{}

const (
	timeHeightShare = 0.34
	timeWidthShare  = 0.86
)

const (
	suffixOfTime = 0.24
	dateOfTime   = 0.20
	gapOfTime    = 0.22
	suffixGap    = 0.35
)

type layout struct {
	Time   ui.Rect
	Suffix ui.Rect
	Date   ui.Rect
}

type fonts struct {
	Time   *ui.Font
	Suffix *ui.Font
	Date   *ui.Font
}

func (plain) Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme) {
	l, f := arrange(r, in)

	ui.DrawText(s, f.Time, l.Time.X, l.Time.Y, palette.Text, palette.Background, r.Time)
	if r.Suffix != "" {
		ui.DrawText(
			s,
			f.Suffix,
			l.Suffix.X,
			l.Suffix.Y,
			palette.Muted,
			palette.Background,
			r.Suffix,
		)
	}
	if r.Dated() {
		ui.DrawText(s, f.Date, l.Date.X, l.Date.Y, palette.Muted, palette.Background, r.Date)
	}
}

func arrange(r Reading, in ui.Rect) (layout, fonts) {
	maxH := int(float64(in.H) * timeHeightShare)
	maxW := int(float64(in.W) * timeWidthShare)

	size := fit(ui.Bold, r.Time+r.Suffix, maxW, maxH)
	f := fonts{
		Time:   ui.MustLoad(ui.Bold, size),
		Suffix: ui.MustLoad(ui.Medium, int(float64(size)*suffixOfTime)),
		Date:   ui.MustLoad(ui.Regular, int(float64(size)*dateOfTime)),
	}

	timeW, timeH := f.Time.Measure(r.Time)

	dateW, dateH := 0, 0
	if r.Dated() {
		dateW, dateH = f.Date.Measure(r.Date)
	}

	suffixW, suffixH := 0, 0
	if r.Suffix != "" {
		suffixW, suffixH = f.Suffix.Measure(r.Suffix)

		suffixW += int(float64(suffixH) * suffixGap)
	}

	between := 0
	if r.Dated() {
		between = int(float64(timeH) * gapOfTime)
	}

	top := in.Y + (in.H-(timeH+between+dateH))/2
	timeX := in.X + (in.W-(timeW+suffixW))/2

	l := layout{
		Time: ui.Rect{X: timeX, Y: top, W: timeW, H: timeH},
		Date: ui.Rect{X: in.X + (in.W-dateW)/2, Y: top + timeH + between, W: dateW, H: dateH},
	}
	if r.Suffix != "" {
		l.Suffix = ui.Rect{
			X: timeX + timeW + int(float64(suffixH)*suffixGap),
			Y: top + f.Time.Ascent() - f.Suffix.Ascent(),
			W: suffixW - int(float64(suffixH)*suffixGap),
			H: suffixH,
		}
	}
	return l, f
}
