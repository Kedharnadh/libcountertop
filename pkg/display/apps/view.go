package apps

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/navigation"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
)

const settle = 400 * time.Millisecond

func (a *Apps) Screens(look *widgets.UI) func(shell.View) *navigation.Screen {
	return func(v shell.View) *navigation.Screen {
		if own, ok := v.(*View); !ok || own.apps != a {
			return nil
		}
		return &navigation.Screen{
			Title: say.T("apps.title"),
			View:  v,
			Build: func(w *gogui.Window) gogui.View { return a.body(w, look) },
		}
	}
}

type grid struct {
	cols      int
	side, gap float32
}

func layout(width float32) grid {
	gap := style.Reach() * 0.25
	cols := max(3, int(width/(style.Reach()*4)))
	return grid{cols: cols, side: (width - gap*float32(cols-1)) / float32(cols), gap: gap}
}

func (g grid) cell(n int) ui.Rect {
	step := g.side + g.gap
	return ui.Rect{
		X: int(float32(n%g.cols) * step),
		Y: int(float32(n/g.cols) * step),
		W: int(g.side),
		H: int(g.side),
	}
}

func (g grid) height(count int) float32 {
	rows := (count + g.cols - 1) / g.cols
	return max(0, float32(rows)*(g.side+g.gap)-g.gap)
}

func (g grid) drop(from int, dx, dy float32, count int) int {
	at := g.cell(from)
	step := g.side + g.gap
	x := float32(at.X) + g.side/2 + dx
	y := float32(at.Y) + g.side/2 + dy
	col := min(max(int(x/step), 0), g.cols-1)
	row := max(int(y/step), 0)
	return min(row*g.cols+col, count-1)
}

func (a *Apps) body(w *gogui.Window, look *widgets.UI) gogui.View {
	vw, vh := w.WindowSize()
	width := float32(vw - 2*gogui.PadLarge)
	g := layout(width)
	list := a.List()

	a.mu.Lock()
	d := a.drag
	a.mu.Unlock()

	var tiles []gogui.View
	var lifted gogui.View
	for n, app := range list {
		at := g.cell(n)
		tile := a.tile(look, n, app, g, len(list))
		if n == d.from {
			at.X += int(d.dx)
			at.Y += int(d.dy)
			lifted = widgets.Placed(at, tile)
			continue
		}
		tiles = append(tiles, widgets.Placed(at, tile))
	}
	if lifted != nil {
		tiles = append(tiles, lifted)
	}

	content := []gogui.View{gogui.Column(gogui.ContainerCfg{
		ID:      "apps",
		Width:   width,
		Height:  g.height(len(list)),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Content: tiles,
	})}
	note := say.T("apps.hint")
	if len(list) == 0 {
		note = say.T("apps.empty")
	}
	content = append(content, gogui.Label(note, widgets.Secondary()))

	return widgets.Placed(ui.Rect{W: int(width), H: vh}, gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(g.gap),
		Content: content,
	}))
}

func (a *Apps) tile(look *widgets.UI, n int, app App, g grid, count int) gogui.View {
	docked := a.Docked(app.ID)
	mark := gogui.IconStarO
	if docked {
		mark = gogui.IconStar
	}
	st := gogui.CurrentTheme().TextStyleIconLarge
	st.Color = gogui.CurrentTheme().Cfg.TextStyleDef.Color
	st.Size = g.side * 0.2
	star := look.Presses.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:      "app-star-" + app.ID,
		Padding: gogui.PadAll(g.side * 0.06),
		Content: []gogui.View{gogui.Label(mark, st)},
	}, func(e gogui.EventCtx) {
		a.SetDocked(app.ID, !docked)
		e.Consume()
	})

	glyph := ""
	if app.Glyph != nil {
		glyph = app.Glyph()
	}
	return look.Entity(widgets.EntityTile{
		ID:        "app-" + app.ID,
		Width:     g.side,
		Height:    g.side,
		Glyph:     glyph,
		Name:      app.Label(),
		Available: true,
		Corner:    star,
		Tap: func(gogui.EventCtx) {
			if !a.settling() {
				app.Open()
			}
		},
		Pan: func(e gogui.EventCtx) { a.pan(n, e, g, count) },
	})
}

func (a *Apps) settling() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.drag.from >= 0 || time.Since(a.drag.dropped) < settle
}

func (a *Apps) pan(n int, e gogui.EventCtx, g grid, count int) {
	a.mu.Lock()
	switch e.Event.GesturePhase {
	case gogui.GesturePhaseBegan:
		a.drag = drag{from: n, dx: e.Event.GestureDX, dy: e.Event.GestureDY}
	case gogui.GesturePhaseChanged:
		if a.drag.from == n {
			a.drag.dx += e.Event.GestureDX
			a.drag.dy += e.Event.GestureDY
		}
	case gogui.GesturePhaseEnded, gogui.GesturePhaseCancelled:
		d := a.drag
		a.drag = drag{from: -1, dropped: time.Now()}
		a.mu.Unlock()
		if e.Event.GesturePhase == gogui.GesturePhaseEnded && d.from == n {
			a.Move(n, g.drop(n, d.dx, d.dy, count))
		}
		a.options.Shell.Redraw()
		return
	}
	a.mu.Unlock()
	a.options.Shell.Redraw()
}
