package screens

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/dashboard"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/homecontrol"
)

func homeGlyph(t dashboard.Tab) string {
	return glyphFor(homecontrol.Selection{Key: t.Key})
}

func (a *App) homeBoard(at ui.Rect, t dashboard.Tab, pal theme.Theme) gogui.View {
	s, ok := a.o.Home.Dash().Find(t.Key)
	if !ok {
		return nil
	}
	return a.tileBoard(at, s, pal)
}

func (a *App) tabStrip(at ui.Rect, tabs []dashboard.Tab, open string) gogui.View {
	var pills []gogui.View
	for _, t := range tabs {
		st := gogui.CurrentTheme().Cfg.TextStyleDef
		st.Size *= 1.6
		cfg := gogui.ContainerCfg{
			ID:      "tab-" + t.Key,
			Height:  float32(at.H),
			Sizing:  gogui.FitFixed,
			Padding: gogui.NewPadding(0, reach()*0.8, 0, reach()*0.6),
			Spacing: gogui.SpacingMedium,
			VAlign:  gogui.VAlignMiddle,
		}
		a.ui.Toolkit.Tab(&cfg, &st, widgets.Chosen(t.Key == open))
		content := []gogui.View{gogui.Label(t.Name, st)}
		if kind, ok := a.boards[t.Kind]; ok && kind.Glyph != nil {
			mark := iconStyle(st.Color)
			mark.Size = st.Size
			content = append([]gogui.View{gogui.Label(kind.Glyph(t), mark)}, content...)
		}
		cfg.Content = content
		pills = append(
			pills,
			a.ui.Presses.Pressable(
				gogui.Row,
				cfg,
				func(gogui.EventCtx) { a.o.Tabs.Show(t.Key) },
			),
		)
	}
	return widgets.Placed(at, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		VAlign:  gogui.VAlignMiddle,
		Content: pills,
	}))
}
