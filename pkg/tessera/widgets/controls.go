package widgets

import (
	"math"
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	editDelay = 700 * time.Millisecond
	editMost  = 10 * time.Second

	coverOpen  = 1
	coverClose = 2
	coverStop  = 8
)

type key struct {
	glyph    rune
	action   string
	data     map[string]string
	checked  bool
	disabled bool
}

type edit struct {
	value float64
	sent  time.Time
	timer *time.Timer
}

func coverKeys(t tessera.Tile) []key {
	sideways := false
	switch t.Text("device_class") {
	case "curtain", "awning", "door", "gate":
		sideways = true
	}
	open, closed := t.State == "open", t.State == "closed"
	if p, ok := t.Number("current_position"); ok {
		open, closed = p >= 99.5, p <= 0.5
	}
	up, down := rune(0xF005D), rune(0xF0045)
	if sideways {
		up, down = 0xF084E, 0xF084C
	}
	features := coverOpen | coverClose | coverStop
	if f, ok := t.Number("supported_features"); ok && f > 0 {
		features = int(f)
	}
	var keys []key
	if features&coverOpen != 0 {
		keys = append(keys, key{
			glyph:    up,
			action:   "cover.open_cover",
			checked:  t.State == "opening",
			disabled: open && t.State != "closing",
		})
	}
	if features&coverStop != 0 {
		keys = append(keys, key{glyph: 0xF04DB, action: "cover.stop_cover"})
	}
	if features&coverClose != 0 {
		keys = append(keys, key{
			glyph:    down,
			action:   "cover.close_cover",
			checked:  t.State == "closing",
			disabled: closed && t.State != "opening",
		})
	}
	return keys
}

func timerKeys(t tessera.Tile) []key {
	if t.State == "active" {
		return []key{
			{glyph: 0xF03E4, action: "timer.pause"},
			{glyph: 0xF0156, action: "timer.cancel"},
		}
	}
	return []key{{glyph: 0xF040A, action: "timer.start"}, {glyph: 0xF0156, action: "timer.cancel"}}
}

func nextOption(t tessera.Tile, direction int) string {
	options := t.Texts("options")
	if len(options) == 0 {
		return ""
	}
	current := 0
	for i, o := range options {
		if o == t.State {
			current = i
		}
	}
	return options[(current+direction+len(options))%len(options)]
}

func optionKeys(t tessera.Tile) []key {
	few := len(t.Texts("options")) < 2
	action := t.Domain() + ".select_option"
	return []key{
		{
			glyph:    0xF0141,
			action:   action,
			data:     map[string]string{"option": nextOption(t, -1)},
			disabled: few,
		},
		{
			glyph:    0xF0142,
			action:   action,
			data:     map[string]string{"option": nextOption(t, 1)},
			disabled: few,
		},
	}
}

func runAction(t tessera.Tile) (action, label string) {
	switch d := t.Domain(); d {
	case "scene":
		return "scene.turn_on", say.T("tessera.key.activate")
	case "script":
		return "script.turn_on", say.T("tessera.key.run")
	case "automation":
		return "automation.trigger", say.T("tessera.key.run")
	case "button", "input_button":
		return d + ".press", say.T("tessera.key.press")
	}
	return "", ""
}

func stepValue(current, step, lo, hi float64, direction int) float64 {
	if step <= 0 {
		step = 1
	}
	next := lo + math.Round((current+float64(direction)*step-lo)/step)*step
	return min(max(next, lo), hi)
}

func (v *View) panel(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f kit.EntityTile,
) func(float32) gogui.View {
	if t.Display == "watch" || t.Controls == "" || t.Controls == "none" {
		return nil
	}
	switch c := t.Controls; c {
	case "toggle":
		return func(float32) gogui.View { return v.toggle(w, id, t, f) }
	case "run":
		if action, label := runAction(t); action != "" {
			return func(float32) gogui.View { return v.runKey(w, id, t, f, action, label) }
		}
	case "brightness", "speed", "position", "position_tilt", "slider":
		return func(width float32) gogui.View { return v.slider(w, id, t, f, width) }
	case "stepper":
		switch t.Domain() {
		case "number", "input_number":
			return func(width float32) gogui.View { return v.stepper(w, id, t, f, width) }
		case "select", "input_select":
			return func(width float32) gogui.View { return v.keys(w, id, t, f, optionKeys(t), width) }
		}
	case "buttons", "buttons_tilt":
		switch t.Domain() {
		case "cover":
			return func(width float32) gogui.View { return v.keys(w, id, t, f, coverKeys(t), width) }
		case "timer":
			return func(width float32) gogui.View { return v.keys(w, id, t, f, timerKeys(t), width) }
		}
	}
	return nil
}

func keyStyle() gogui.TextStyle {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Family = icons.Family
	return st
}

func (v *View) panelKey(id string, k key, width float32, do func(gogui.EventCtx)) gogui.View {
	state := style.Rest
	if k.checked {
		state = style.Chosen
	}
	return v.o.UI.Toolkit.Keyed(gogui.ContainerCfg{
		ID:       id,
		Width:    width,
		Height:   style.Reach(),
		Sizing:   gogui.FixedFixed,
		Radius:   gogui.RadiusLarge,
		Color:    kit.KeyFill(v.o.UI.Palette()),
		Disabled: do == nil,
	}, icons.Glyph(k.glyph), keyStyle(), state, do)
}

func (v *View) keys(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f kit.EntityTile,
	keys []key,
	room float32,
) gogui.View {
	gap := style.Reach() * 0.15
	width := min(style.Reach()*1.2, (room-float32(len(keys)-1)*gap)/float32(max(1, len(keys))))
	views := make([]gogui.View, 0, len(keys))
	for i, k := range keys {
		do := func(gogui.EventCtx) { v.send(w, t, k.action, k.data, nil, "") }
		if k.disabled || !f.Available {
			do = nil
		}
		views = append(views, v.panelKey(id+"-key-"+strconv.Itoa(i), k, width, do))
	}
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: views,
	})
}

func (v *View) toggle(w *gogui.Window, id string, t tessera.Tile, f kit.EntityTile) gogui.View {
	on := f.On
	action := t.Domain() + ".turn_on"
	wish := "on"
	if on {
		action, wish = t.Domain()+".turn_off", "off"
	}
	return v.o.UI.Presses.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:       id + "-toggle",
		Sizing:   gogui.FitFit,
		Padding:  gogui.NoPadding,
		Disabled: !f.Available,
		Content:  []gogui.View{v.o.UI.Kit().Toggle(id+"-switch", on)},
	}, func(gogui.EventCtx) { v.send(w, t, action, nil, nil, wish) })
}

func (v *View) runKey(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f kit.EntityTile,
	action, label string,
) gogui.View {
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	do := func(gogui.EventCtx) { v.send(w, t, action, nil, nil, "") }
	if !f.Available {
		do = nil
	}
	return v.o.UI.Toolkit.Keyed(gogui.ContainerCfg{
		ID:     id + "-run",
		Width:  max(style.Reach()*1.5, w.TextWidth(label, st)+style.Reach()*0.6),
		Height: style.Reach(),
		Sizing: gogui.FixedFixed,
		Radius: gogui.RadiusLarge,
		Color:  kit.KeyFill(v.o.UI.Palette()),
	}, label, st, style.Rest, do)
}

func (v *View) stepper(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f kit.EntityTile,
	width float32,
) gogui.View {
	lo, hi := bounds(t)
	step := 1.0
	if s, ok := t.Number("step"); ok && s > 0 {
		step = s
	}
	current, _ := v.editing(t)
	press := func(direction int) func(gogui.EventCtx) {
		if !f.Available {
			return nil
		}
		return func(e gogui.EventCtx) {
			v.stepTo(w, t, stepValue(current, step, lo, hi, direction))
			e.Window.InvalidateLayout()
		}
	}
	key := func(suffix string, glyph rune, do func(gogui.EventCtx)) gogui.View {
		return v.panelKey(id+suffix, key{glyph: glyph}, style.Reach(), do)
	}
	text := formatStep(current, step) + unitSuffix(t.Text("unit_of_measurement"))
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	return gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.NoSpacing,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{
			key("-minus", 0xF0374, press(-1)),
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{gogui.Label(text, st)},
			}),
			key("-plus", 0xF0415, press(1)),
		},
	})
}

func formatStep(value, step float64) string {
	digits := 0
	if step < 1 {
		digits = 1
	}
	return strconv.FormatFloat(value, 'f', digits, 64)
}

func unitSuffix(unit string) string {
	if unit == "" || unit == "°" || unit == "%" {
		return unit
	}
	return " " + unit
}

func (v *View) editing(t tessera.Tile) (float64, bool) {
	real, _ := strconv.ParseFloat(t.State, 64)
	v.mu.Lock()
	defer v.mu.Unlock()
	e, ok := v.edits[t.Entity]
	switch {
	case !ok:
		return real, false
	case e.sent.IsZero():
		return e.value, true
	case real == e.value || time.Since(e.sent) > editMost:
		delete(v.edits, t.Entity)
		return real, false
	}
	return e.value, true
}

func (v *View) stepTo(w *gogui.Window, t tessera.Tile, value float64) {
	v.mu.Lock()
	e, ok := v.edits[t.Entity]
	if !ok || !e.sent.IsZero() {
		e = &edit{}
		v.edits[t.Entity] = e
	}
	e.value = value
	if e.timer != nil {
		e.timer.Stop()
	}
	e.timer = time.AfterFunc(editDelay, func() {
		v.mu.Lock()
		e.sent = time.Now()
		sent := e.value
		v.mu.Unlock()
		data := map[string]string{"value": strconv.FormatFloat(sent, 'f', -1, 64)}
		v.send(w, t, t.Domain()+".set_value", data, nil, "")
	})
	v.mu.Unlock()
}

func stepText(value float64, t tessera.Tile) string {
	step := 1.0
	if s, ok := t.Number("step"); ok && s > 0 {
		step = s
	}
	return formatStep(value, step)
}

func controlWidth(w *gogui.Window, t tessera.Tile) float32 {
	reach := style.Reach()
	row := func(n int) float32 { return float32(n)*reach*1.2 + float32(n-1)*reach*0.15 }
	switch t.Controls {
	case "toggle":
		return reach * 2
	case "run":
		if _, label := runAction(t); label != "" {
			return max(
				reach*1.5,
				w.TextWidth(label, gogui.CurrentTheme().Cfg.TextStyleDef)+reach*0.6,
			)
		}
	case "stepper":
		switch t.Domain() {
		case "number", "input_number":
			text := t.State + unitSuffix(t.Text("unit_of_measurement")) + "8"
			return 2*reach + w.TextWidth(text, gogui.CurrentTheme().Cfg.TextStyleDef) + reach*0.4
		case "select", "input_select":
			return row(2)
		}
	case "buttons", "buttons_tilt":
		switch t.Domain() {
		case "cover":
			return row(len(coverKeys(t)))
		case "timer":
			return row(len(timerKeys(t)))
		}
	}
	return 0
}
