package widgets

import (
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

func PageDots(id string, at ui.Rect, count, current int) gogui.View {
	t := gogui.CurrentTheme().Cfg
	side := style.Reach() * 0.3
	dots := make([]gogui.View, 0, count)
	for i := range count {
		fill := t.ColorTextSecondary.WithOpacity(0.4)
		if i == current {
			fill = t.ColorAccent
		}
		dots = append(dots, gogui.Row(gogui.ContainerCfg{
			ID:      fmt.Sprintf("%s-%d", id, i),
			Width:   side,
			Height:  side,
			Sizing:  gogui.FixedFixed,
			Radius:  gogui.RadiusLarge,
			Color:   fill,
			Padding: gogui.NoPadding,
		}))
	}
	return Placed(at, gogui.Row(gogui.ContainerCfg{
		Width:   float32(at.W),
		Height:  float32(at.H),
		Sizing:  gogui.FixedFixed,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Spacing: gogui.SpacingMedium,
		Padding: gogui.NoPadding,
		Content: dots,
	}))
}
