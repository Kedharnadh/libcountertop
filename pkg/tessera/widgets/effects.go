package widgets

import (
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const lightEffect = 4

func lightEffects(t tessera.Tile) bool {
	f, _ := t.Number("supported_features")
	return int(f)&lightEffect != 0
}

func effectsPage(t tessera.Tile) bool {
	return lightEffects(t) || len(t.ExtraList("rows")) > 0 || len(t.ExtraList("nums")) > 0
}

func (v *View) pickNames(entity string) {
	v.mu.Lock()
	v.sub.pick, v.sub.page, v.sub.source = entity, 0, 0
	screen := v.screen
	v.mu.Unlock()
	if screen != nil {
		screen.AskNames(entity, 0)
	}
}

func (v *View) effectsBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width, room float32,
) []gogui.View {
	if s := v.aside(); s.pick != "" {
		return v.namesBody(w, t, e, s, width, room)
	}
	var out []gogui.View
	choose := func(id, name, value, entity string) gogui.View {
		var do func(gogui.EventCtx)
		if e.Available {
			do = func(gogui.EventCtx) { v.pickNames(entity) }
		}
		return v.labelled(name, value, width, v.wordKey(id, value, width, false, do))
	}
	if lightEffects(t) {
		out = append(
			out,
			choose("tessera-effect", say.T("tessera.light.effect"), t.Text("effect"), t.Entity),
		)
	}
	for i, r := range t.ExtraList("rows") {
		entity, _ := r["e"].(string)
		name, _ := r["n"].(string)
		value, _ := r["s"].(string)
		out = append(out, choose("tessera-effect-row-"+strconv.Itoa(i), name, value, entity))
	}
	now := time.Now()
	for i, n := range t.ExtraList("nums") {
		entity, _ := n["e"].(string)
		name, _ := n["n"].(string)
		lo, _ := n["lo"].(float64)
		hi, _ := n["hi"].(float64)
		step, _ := n["st"].(float64)
		at, known := n["v"].(float64)
		if hi <= lo {
			continue
		}
		if step <= 0 {
			step = 1
		}
		real := 0.0
		if known {
			real = (at - lo) / (hi - lo)
		}
		target := tessera.Tile{Entity: entity, State: num(at)}
		d := dial{
			key:  entity,
			real: real,
			live: e.Available,
			commit: func(w *gogui.Window, x float64) {
				value := min(max(lo+float64(int(x*(hi-lo)/step+0.5))*step, lo), hi)
				v.send(w, target, target.Domain()+".set_value",
					map[string]string{"value": num(value)}, nil, "")
			},
		}
		x := v.held(d, now)
		shown := min(max(lo+float64(int(x*(hi-lo)/step+0.5))*step, lo), hi)
		out = append(out, v.labelled(
			name,
			num(shown),
			width,
			v.dialSlider(
				w,
				"tessera-effect-num-"+strconv.Itoa(i),
				d,
				x,
				width,
				e.Accent,
				!e.Available,
			),
		))
	}
	return out
}

func (v *View) namesBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	s sub,
	width, room float32,
) []gogui.View {
	reach := style.Reach()
	v.mu.Lock()
	screen := v.screen
	v.mu.Unlock()
	if screen == nil {
		return nil
	}
	names, ok := screen.Names(s.pick)
	if !ok {
		return []gogui.View{gogui.Label(say.T("tessera.light.waiting"), secondary())}
	}
	current := t.Text("effect")
	action, field := "light.turn_on", "effect"
	target := t
	if s.pick != t.Entity {
		target = tessera.Tile{Entity: s.pick}
		action, field = target.Domain()+".select_option", "option"
		for _, r := range t.ExtraList("rows") {
			if r["e"] == s.pick {
				current, _ = r["s"].(string)
			}
		}
	}
	per := max(1, int((room-reach*1.4)/(reach*1.15)))
	pages := max(1, (len(names.Names)+per-1)/per)
	page := min(max(s.page, 0), pages-1)
	var out []gogui.View
	for i, name := range names.Names[page*per : min(len(names.Names), (page+1)*per)] {
		var do func(gogui.EventCtx)
		if e.Available {
			do = func(gogui.EventCtx) {
				v.send(w, target, action, map[string]string{field: name}, nil, "")
				v.mu.Lock()
				v.sub.pick = ""
				v.mu.Unlock()
			}
		}
		out = append(
			out,
			v.wordKey("tessera-name-"+strconv.Itoa(i), name, width, name == current, do),
		)
	}
	turn := func(page, source int) func(*gogui.Window) {
		return func(*gogui.Window) {
			v.mu.Lock()
			asked := source != v.sub.source
			v.sub.page, v.sub.source = page, source
			v.mu.Unlock()
			if asked {
				screen.AskNames(s.pick, source)
			}
		}
	}
	keys := []gogui.View{}
	switch {
	case page > 0:
		keys = append(
			keys,
			v.key("tessera-names-back", gogui.IconArrowLeft, turn(page-1, s.source)),
		)
	case names.Page > 0:
		keys = append(keys, v.key("tessera-names-back", gogui.IconArrowLeft, turn(0, names.Page-1)))
	}
	switch {
	case page < pages-1:
		keys = append(
			keys,
			v.key("tessera-names-next", gogui.IconArrowRight, turn(page+1, s.source)),
		)
	case names.Page < names.Pages-1:
		keys = append(
			keys,
			v.key("tessera-names-next", gogui.IconArrowRight, turn(0, names.Page+1)),
		)
	}
	if len(keys) > 0 {
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.3),
			HAlign:  gogui.HAlignCenter,
			Content: keys,
		}))
	}
	return out
}
