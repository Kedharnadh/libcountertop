package boot

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
)

const (
	rowShare   = 0.11
	nameShare  = 0.36
	doingShare = 0.25
	dotShare   = 0.22
)

func Raster(s ui.Surface, p theme.Theme, progress []startup.Progress, version, skip string) {
	progress = startup.Display(progress)
	w, h := s.Size()
	logo, list := Split(w, h)
	ui.Clear(s, logo)
	ui.FillRect(s, list, p.Background)
	if len(progress) > 0 {
		rows(s, progressArea(w, h), p, progress)
	}
	Version(s, p, version)
	if !startup.Settled(progress) {
		at := Skip(w, h)
		ui.FillRounded(s, at, at.H/2, p.Surface)
		font := ui.MustLoad(ui.Medium, max(at.H*2/5, 1))
		tw, th := font.Measure(skip)
		ui.DrawText(s, font, at.X+(at.W-tw)/2, at.Y+(at.H-th)/2, p.Text, p.Surface, skip)
	}
}

func Version(s ui.Surface, p theme.Theme, version string) {
	w, h := s.Size()
	f := ui.MustLoad(ui.Regular, max(int(float64(min(w, h))*0.024), 1))
	tw, th := f.Measure(version)
	ui.DrawText(s, f, w-tw-th, h-2*th, p.Muted, p.Background, version)
}

func rows(s ui.Surface, in ui.Rect, t theme.Theme, progress []startup.Progress) {
	row, per, colW := rowLayout(in, len(progress))

	name := ui.MustLoad(ui.Medium, max(min(int(float64(row)*nameShare), 22), 1))
	doing := ui.MustLoad(ui.Regular, max(min(int(float64(row)*doingShare), 16), 1))

	dot := int(float64(row) * dotShare)
	top := in.Y + (in.H-per*row)/2
	gap := row / 12

	for i, at := range progress {
		left := in.X + (i/per)*colW + row/2
		y := top + (i%per)*row

		mark := t.Muted
		switch {
		case at.Failed:
			mark = t.Warning
		case at.Done:
			mark = t.Accent
		}
		ui.FillRounded(s, ui.Rect{X: left, Y: y + (row-dot)/2, W: dot, H: dot}, dot/2, mark)

		x := left + dot + row/3
		room := in.X + (i/per+1)*colW - row/2 - x
		label, says := ui.Fit(name, at.Name, room), ui.Fit(doing, at.Doing, room)
		saying := says != ""

		_, nameH := name.Measure(label)
		_, doingH := doing.Measure(says)

		tall := nameH
		if saying {
			tall += gap + doingH
		}
		up := y + (row-tall)/2

		ui.DrawText(s, name, x, up, t.Text, t.Background, label)
		if saying {
			ui.DrawText(s, doing, x, up+nameH+gap, t.Muted, t.Background, says)
		}
	}
}
