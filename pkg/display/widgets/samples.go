package widgets

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func RowPreview(page string, r menu.Row, palette theme.Theme) gogui.View {
	h := int(gogui.CurrentTheme().Cfg.SizeTextMedium * 2.4)
	w := h * 2
	return Picture(
		Painted(
			"row/"+page+"/"+r.Label+"/"+r.Value+"/"+time.Now().Format("15:04"),
			w,
			h,
			palette.Background,
			palette,
			r.Preview,
		),
		w,
		h,
	)
}

func Swatch(p theme.Theme, w, h float32) gogui.View {
	side := min(h*0.28, (w*0.8)/5)
	var chips []gogui.View
	for _, c := range []theme.Color{p.Surface, p.Text, p.Accent, p.Accent2} {
		chips = append(
			chips,
			gogui.Column(
				gogui.ContainerCfg{
					Width:   side,
					Height:  side,
					Sizing:  gogui.FixedFixed,
					Color:   guiColor(c),
					Radius:  gogui.RadiusPx(side * 0.2),
					Padding: gogui.NoPadding,
				},
			),
		)
	}
	return gogui.Row(gogui.ContainerCfg{
		Width:       w,
		Height:      h,
		Sizing:      gogui.FixedFixed,
		Color:       guiColor(p.Background),
		ColorBorder: guiColor(p.Text.Blend(p.Background, 0.7)),
		SizeBorder:  gogui.BorderPx(1),
		Radius:      gogui.RadiusMedium,
		HAlign:      gogui.HAlignCenter,
		VAlign:      gogui.VAlignMiddle,
		Spacing:     gogui.SpacingPx(side * 0.35),
		Padding:     gogui.NoPadding,
		Content:     chips,
	})
}
