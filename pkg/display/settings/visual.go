package settings

import (
	"image"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
)

func NoneTile(s ui.Surface, at ui.Rect, p theme.Theme) { ui.FillRect(s, at, p.Surface) }

func VisualPicker(
	title, empty string,
	none func(),
	chosen func() string,
	pick func(visual.Kind),
	thumbnail func(visual.Kind, int, int) *image.RGBA,
) *shell.Page {
	return &shell.Page{
		Title: title,
		Tiles: func() ([]menu.Cell, []func(int)) {
			now := ""
			if chosen != nil {
				now = chosen()
			}
			var cells []menu.Cell
			var acts []func(int)
			if none != nil {
				cells = append(
					cells,
					menu.Cell{Label: empty, Chosen: chosen != nil && now == "", Paint: NoneTile},
				)
				acts = append(acts, func(int) { none() })
			}
			for _, k := range visual.Built() {
				cells = append(
					cells,
					menu.Cell{
						Label:  k.Label(),
						Chosen: chosen != nil && string(k) == now,
						Paint:  ThumbTile(k, thumbnail),
					},
				)
				acts = append(acts, func(int) { pick(k) })
			}
			return cells, acts
		},
	}
}

func ThumbTile(
	k visual.Kind,
	thumbnail func(visual.Kind, int, int) *image.RGBA,
) func(ui.Surface, ui.Rect, theme.Theme) {
	return func(s ui.Surface, at ui.Rect, palette theme.Theme) {
		img := thumbnail(k, at.W, at.H)
		if img == nil {
			ui.FillRect(s, at, palette.Surface)
			return
		}
		ui.DrawRGBA(s, at.X, at.Y, img, 1, ui.ClipOf(s))
	}
}
