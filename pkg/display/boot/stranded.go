package boot

import (
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/say"
)

const (
	strandedMarkShare = 0.34
	strandedSignShare = 0.28
	strandedWordShare = 0.045
)

type Logo func(s ui.Surface, at ui.Rect, on theme.Color)

func Stranded(s ui.Surface, t theme.Theme, logo Logo, version string) {
	w, h := s.Size()
	ui.Fill(s, t.Background)

	short := min(w, h)
	mark := int(float64(short) * strandedMarkShare)
	sign := int(float64(short) * strandedSignShare)

	word := say.T("boot.stranded")
	font := ui.MustLoad(ui.Medium, int(float64(short)*strandedWordShare))
	wordW, wordH := font.Measure(word)

	gap := wordH
	top := (h - (mark + gap + sign + gap + wordH)) / 2

	logo(s, ui.Rect{X: (w - mark) / 2, Y: top, W: mark, H: mark}, t.Background)
	ui.DrawIcon(
		s,
		icons.AlertWarning,
		ui.Rect{X: (w - sign) / 2, Y: top + mark + gap, W: sign, H: sign},
		t.Danger,
		t.Background,
	)
	ui.DrawText(s, font, (w-wordW)/2, top+mark+gap+sign+gap, t.Text, t.Background, word)

	Version(s, t, version)
}
