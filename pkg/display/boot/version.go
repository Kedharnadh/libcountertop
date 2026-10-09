package boot

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func Version(s ui.Surface, p theme.Theme, version string) {
	w, h := s.Size()
	f := ui.MustLoad(ui.Regular, max(int(float64(min(w, h))*0.024), 1))
	tw, th := f.Measure(version)
	ui.DrawText(s, f, w-tw-th, h-2*th, p.Muted, p.Background, version)
}
