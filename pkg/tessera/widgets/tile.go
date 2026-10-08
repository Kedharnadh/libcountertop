package widgets

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func (v *View) tile(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	l tessera.Layout,
	at ui.Rect,
	c cell,
	f frame,
) gogui.View {
	now := f.now
	wt := v.seen(t, now)
	shown := t
	if wt != nil && wt.wish != "" {
		shown.State = wt.wish
	}
	if d := t.Domain(); d == "number" || d == "input_number" {
		if value, edited := v.editing(t); edited {
			shown.State = stepText(value, t)
		}
	}
	e := kit.EntityTile{
		ID:        id,
		Width:     float32(at.W),
		Height:    float32(at.H),
		Glyph:     icons.Glyph(iconFor(shown, v.arming(t.Entity))),
		Family:    icons.Family,
		Name:      t.Name,
		Available: f.fresh && available(t),
		On:        active(shown),
		Ground:    ground(t.Background, f.pal),
		Window:    w,
	}
	if t.Domain() == "lock" {
		e.On = e.Available
	}
	if e.On {
		a := accent(shown, f.pal)
		c := gogui.RGB(a.R, a.G, a.B)
		e.Accent = &c
	}
	if pageNumber(t.Entity) > 0 {
		v.link(&e, t, l)
		return v.o.UI.Entity(e)
	}
	e.Status = v.line(shown, l, wt, now, at.W < at.H*3/2)
	if t.Display == "watch" {
		e.Value, e.Status = e.Status, ""
		unit := shown.Text("unit_of_measurement")
		measured := unit != "" && available(shown)
		if measured && e.Value == word(shown, l.Numbers) {
			e.Value, e.Unit = l.Numbers.Localize(shown.State), unit
		}
	}
	switch d := t.Domain(); {
	case d == "timer" && t.State == "active":
		v.tick(w, now, time.Second)
	case d == "script" || d == "scene" || d == "button" || d == "input_button" || runs(t):
		v.tick(w, now, time.Minute)
	}
	switch {
	case c.rows > 1 && bigKey(t):
	case sliding(t):
		e.Control = func(width float32) gogui.View { return v.slider(w, id, t, e, width) }
	default:
		e.Control = v.panel(w, id, t, e)
		e.ControlWidth = controlWidth(w, t)
	}
	if e.Available {
		e.Tap = v.press(w, id, t, false)
		if routeOf(t, true).route != nothing {
			e.Hold = v.press(w, id, t, true)
		}
	}
	return v.o.UI.Entity(e)
}

func (v *View) line(
	t tessera.Tile,
	l tessera.Layout,
	wt *watch,
	now time.Time,
	narrow bool,
) string {
	if !available(t) {
		return say.T("home.tile.unavailable")
	}
	if wt != nil {
		if s := wt.line(now); s != "" {
			return s
		}
	}
	if v.arming(t.Entity) {
		return say.T("tessera.again")
	}
	return secondLine(t, domainLine(t, l, now, narrow), l.Clock24, now)
}

func (v *View) press(w *gogui.Window, id string, t tessera.Tile, hold bool) func(gogui.EventCtx) {
	return func(e gogui.EventCtx) {
		r := routeOf(t, hold)
		switch r.route {
		case service:
			v.act(w, t, r.service, nil, nil)
		case custom:
			a := t.Action()
			v.act(w, t, a.Service, a.Data, a.Templates)
		case guarded:
			v.guard(w, t)
		}
		e.Window.InvalidateLayout()
	}
}

func bigKey(t tessera.Tile) bool {
	switch t.Domain() {
	case "light", "switch", "input_boolean", "fan", "script", "scene", "button", "input_button":
	default:
		return false
	}
	switch t.Controls {
	case "", "toggle", "run", "none", "auto":
		return t.Inline != "slider"
	}
	return false
}
