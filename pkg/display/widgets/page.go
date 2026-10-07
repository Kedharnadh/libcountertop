package widgets

import (
	"fmt"
	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"slices"
)

// PageRenderer supplies the styled controls and platform preview for a page.
type PageRenderer struct {
	Row     func(string, string, model.Row, func(int), bool) gogui.View
	Tile    func(*gogui.Window, string, string, model.Cell, func(int)) gogui.View
	Preview func(*gogui.Window, func(ui.Rect) bool) gogui.View
	Beside  func(*gogui.Window, gogui.View, gogui.View) gogui.View
}

// PageBody composes rows or tiles with an optional live preview.
func PageBody(w *gogui.Window, p *shell.Page, render PageRenderer) gogui.View {
	var views []gogui.View
	switch {
	case p.Build != nil:
		rows, taps := p.Build()
		glyphs := slices.ContainsFunc(rows, func(r model.Row) bool { return r.Glyph != "" })
		for i, r := range rows {
			var tap func(int)
			if i < len(taps) {
				tap = taps[i]
			}
			views = append(views, render.Row(fmt.Sprintf("row-%d", i), p.Title, r, tap, glyphs))
		}
	case p.Tiles != nil:
		cells, taps := p.Tiles()
		var tiles []gogui.View
		for i, c := range cells {
			var tap func(int)
			if i < len(taps) {
				tap = taps[i]
			}
			tiles = append(tiles, render.Tile(w, fmt.Sprintf("tile-%d", i), p.Title, c, tap))
		}
		views = append(views, gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Wrap: true, Spacing: gogui.SpacingMedium, Content: tiles}))
	}
	list := gogui.Column(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding, Spacing: gogui.SpacingTight, Content: views})
	if p.Preview == nil {
		return list
	}
	return render.Beside(w, render.Preview(w, p.Preview), list)
}
