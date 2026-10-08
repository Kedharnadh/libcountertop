package widgets

import (
	"strconv"
	"strings"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func pageNumber(entity string) int {
	rest, ok := strings.CutPrefix(entity, "screen.page_")
	if !ok || rest == "" || rest[0] == '0' || len(rest) > 2 {
		return 0
	}
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return n
}

func (v *View) link(e *kit.EntityTile, t tessera.Tile, l tessera.Layout) {
	to := pageNumber(t.Entity) - 1
	if to < 0 || to >= len(l.Pages) {
		e.Available = false
		return
	}
	e.Glyph = icons.Glyph(0xF0054)
	e.Available = true
	if e.Name == "" {
		e.Name = l.Pages[to].Title
	}
	e.Status = ""
	e.Tap = func(e gogui.EventCtx) { v.move(e.Window, func() { v.open(l, to) }) }
}
