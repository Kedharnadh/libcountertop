package widgets

import (
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	lockOpen = 1
	cardTime = 220 * time.Millisecond
)

func (v *View) openCard(w *gogui.Window, t tessera.Tile) {
	v.mu.Lock()
	v.card, v.hours, v.lift, v.sub = t.Entity, 24, 0, sub{}
	r := v.rects[t.Entity]
	v.origin = ui.Rect{X: v.area.X + r.X, Y: v.area.Y + r.Y, W: r.W, H: r.H}
	screen := v.screen
	v.mu.Unlock()
	if historyCard(t) && screen != nil {
		screen.AskHistory(t.Entity, 24)
	}
	v.raise(w, 0, 1, nil)
}

func (v *View) closeCard() {
	v.mu.Lock()
	v.card, v.code, v.lift, v.sub = "", code{}, 0, sub{}
	v.mu.Unlock()
}

func (v *View) lower(w *gogui.Window) {
	v.mu.Lock()
	from := v.lift
	v.mu.Unlock()
	v.raise(w, from, 0, func() { v.closeCard() })
}

func (v *View) raise(w *gogui.Window, from, to float32, done func()) {
	tw := gogui.NewTweenAnimation("tessera-card", from, to, func(at float32, w *gogui.Window) {
		v.mu.Lock()
		v.lift = at
		v.mu.Unlock()
	})
	tw.Duration = cardTime
	tw.OnDone = func(w *gogui.Window) {
		if done != nil {
			done()
		}
		w.InvalidateLayout()
	}
	w.AnimationAdd(tw)
}

func (v *View) lifted() float32 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.lift
}

func (v *View) showingCard() (string, int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.card, v.hours
}

func (v *View) pickRange(t tessera.Tile, hours int) {
	v.mu.Lock()
	v.hours = hours
	screen := v.screen
	v.mu.Unlock()
	if screen != nil {
		screen.AskHistory(t.Entity, hours)
	}
}

func cardTile(l tessera.Layout, entity string) (tessera.Tile, bool) {
	for _, t := range l.Tiles {
		if t.Entity == entity {
			return t, true
		}
	}
	return tessera.Tile{}, false
}

func (v *View) cardView(
	w *gogui.Window,
	at ui.Rect,
	l tessera.Layout,
	t tessera.Tile,
	f frame,
) gogui.View {
	reach := style.Reach()
	width := float32(sheetRect(at).W) - reach
	pal := f.pal
	shown := t
	if wt := v.seen(t, f.now); wt != nil && wt.wish != "" {
		shown.State = wt.wish
	}
	e := kit.EntityTile{On: active(shown), Available: f.fresh && available(t)}
	if t.Domain() == "lock" {
		e.On = e.Available
	}
	if e.On {
		a := accent(shown, pal)
		c := gogui.RGB(a.R, a.G, a.B)
		e.Accent = &c
	}
	_, hours := v.showingCard()
	var tail gogui.View
	body := []gogui.View{}
	status := v.line(shown, l, v.seen(t, f.now), f.now, false)
	title, back := t.Name, v.lower
	switch d := t.Domain(); {
	case historyCard(t):
		if d == "switch" || d == "input_boolean" {
			tail = v.toggle(w, "tessera-card", t, e)
		}
		body = v.historyBody(w, t, l, e, width, float32(at.H), hours)
		status = ""
	case d == "light" && v.aside().kind == subLamps:
		back = v.backAside
		status = ""
		if l, ok := lampOf(t, v.aside().lamp); ok {
			title = l.Name
		}
		body = v.lampsBody(w, t, f.fresh, width, float32(sheetRect(at).H))
	case d == "light" && v.aside().kind == subEffects:
		back = v.backAside
		status = ""
		body = v.effectsBody(w, t, e, width, float32(sheetRect(at).H))
	case d == "light":
		tail = v.toggle(w, "tessera-card", t, e)
		body = append(body, v.lightBody(w, t, e, width)...)
		body = append(body, v.asideKeys(t, width)...)
	case d == "fan":
		tail = v.toggle(w, "tessera-card", t, e)
		body = append(body, v.slider(w, "tessera-card", t, e, width))
	case d == "cover":
		if _, ok := t.Number("current_position"); ok {
			body = append(body, v.slider(w, "tessera-card", t, e, width))
		}
		body = append(body, v.keys(w, "tessera-card", t, e, coverKeys(t), width))
	case d == "select" || d == "input_select":
		body = append(body, v.options(w, t, width))
	case d == "lock":
		body = append(body, v.lockKeys(w, t, e, width))
	case d == "timer":
		body = append(body, timerBody(t)...)
		body = append(body, v.keys(w, "tessera-card", t, e, timerKeys(t), width))
	case d == "climate":
		tail = v.power(w, t, e, t.State != "off")
		status = climateStatus(shown, l)
		body = v.climateBody(w, t, e, width)
	case d == "humidifier":
		tail = v.power(w, t, e, t.State == "on")
		status = humidifierStatus(shown, l)
		body = v.humidifierBody(w, t, e, width)
	case d == "alarm_control_panel":
		body = v.alarmBody(w, t, e, width)
	case d == "media_player":
		tail = v.power(w, t, e, t.State != "off" && t.State != "standby")
		body = v.mediaBody(w, t, e, width, f.now)
	case d == "weather":
		status = ""
		body = v.weatherBody(w, t, l, width)
	case d == "vacuum":
		body = v.vacuumBody(w, t, e, width)
	case d == "remote":
		tail = v.power(w, t, e, t.State == "on")
		body = v.remoteBody(w, t, e, width)
	case d == "sun":
		body = sunBody(t)
	}
	content := []gogui.View{
		v.cardHeader(w, title, width, tail, back),
	}
	if status != "" {
		content = append(content, gogui.Label(status, secondary()))
	}
	content = append(content, body...)
	sheet := sheetRect(at)
	top := gogui.PadAll(0)
	top.Top = float32(sheet.Y - at.Y)
	return kit.Placed(at, gogui.Column(gogui.ContainerCfg{
		Width:   float32(at.W),
		Height:  float32(at.H),
		Sizing:  gogui.FixedFixed,
		Color:   gogui.RGB(pal.Background.R, pal.Background.G, pal.Background.B),
		Padding: top,
		HAlign:  gogui.HAlignCenter,
		Content: []gogui.View{gogui.Column(gogui.ContainerCfg{
			Width:   float32(sheet.W),
			Height:  float32(sheet.H),
			Sizing:  gogui.FixedFixed,
			Padding: gogui.PadAll(reach * 0.5),
			Spacing: gogui.SpacingPx(reach * 0.35),
			Radius:  gogui.RadiusLarge,
			Color:   surface(pal),
			HAlign:  gogui.HAlignCenter,
			Content: content,
		})},
	}))
}

func sheetRect(at ui.Rect) ui.Rect {
	reach := style.Reach()
	w := int(min(float32(at.W)-reach, reach*17))
	edge := int(reach * 0.5)
	return ui.Rect{X: at.X + (at.W-w)/2, Y: at.Y + edge, W: w, H: at.H - 2*edge}
}

func (v *View) morph(at ui.Rect, f frame) []gogui.View {
	v.mu.Lock()
	p, from := v.lift, v.origin
	v.mu.Unlock()
	to := sheetRect(at)
	if from.W == 0 {
		from = ui.Rect{X: to.X + to.W/2, Y: to.Y + to.H/2}
	}
	mix := func(a, b int) int { return a + int(float32(b-a)*p) }
	box := ui.Rect{
		X: mix(from.X, to.X),
		Y: mix(from.Y, to.Y),
		W: mix(from.W, to.W),
		H: mix(from.H, to.H),
	}
	pal := f.pal
	return []gogui.View{
		kit.Placed(at, gogui.Column(gogui.ContainerCfg{
			Width:   float32(at.W),
			Height:  float32(at.H),
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			Color:   gogui.RGB(pal.Background.R, pal.Background.G, pal.Background.B).WithOpacity(p),
		})),
		kit.Placed(box, gogui.Column(gogui.ContainerCfg{
			Width:   float32(box.W),
			Height:  float32(box.H),
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			Radius:  gogui.RadiusLarge,
			Color:   surface(pal),
		})),
	}
}

func surface(pal theme.Theme) gogui.Color {
	return gogui.RGB(pal.Surface.R, pal.Surface.G, pal.Surface.B)
}

func secondary() gogui.TextStyle {
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	st.Color = gogui.CurrentTheme().Cfg.ColorTextSecondary
	return st
}

func (v *View) cardHeader(
	w *gogui.Window,
	name string,
	width float32,
	tail gogui.View,
	back func(*gogui.Window),
) gogui.View {
	reach := style.Reach()
	title := gogui.CurrentTheme().Cfg.TextStyleDef
	title.Size *= 1.3
	side := func(align gogui.HorizontalAlign, content ...gogui.View) gogui.View {
		return gogui.Row(gogui.ContainerCfg{
			Width:   reach * 2,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			HAlign:  align,
			Content: content,
		})
	}
	var right []gogui.View
	if tail != nil {
		right = []gogui.View{tail}
	}
	return gogui.Row(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{
			side(gogui.HAlignLeft, v.key("tessera-card-back", gogui.IconArrowLeft, back)),
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{
					gogui.Label(kit.Ellipsize(w, name, title, width-reach*4), title),
				},
			}),
			side(gogui.HAlignRight, right...),
		},
	})
}

func (v *View) wordKey(
	id, text string,
	width float32,
	chosen bool,
	do func(gogui.EventCtx),
) gogui.View {
	state := style.Rest
	if chosen {
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
	}, text, gogui.CurrentTheme().Cfg.TextStyleDef, state, do)
}

func (v *View) historyBody(
	w *gogui.Window,
	t tessera.Tile,
	l tessera.Layout,
	e kit.EntityTile,
	width, room float32,
	hours int,
) []gogui.View {
	reach := style.Reach()
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2.2
	now := word(t, l.Numbers)
	if unit := t.Text(
		"unit_of_measurement",
	); unit != "" && available(t) &&
		now == l.Numbers.Localize(t.State) {
		now += unitSuffix(unit)
	}
	var screen *tessera.Screen
	v.mu.Lock()
	screen = v.screen
	v.mu.Unlock()
	var h tessera.History
	ok := false
	if screen != nil {
		h, ok = screen.History(t.Entity, hours)
	}
	if ok && !h.Line {
		for _, pair := range h.Words {
			if pair[0] == t.State {
				now = pair[1]
			}
		}
	}
	out := []gogui.View{gogui.Label(now, big)}
	if ok {
		if s := summary(h, l.Clock24); s != "" {
			out = append(out, gogui.Label(s, secondary()))
		}
	}
	height := max(reach*4.5, room*0.4)
	accent := gogui.CurrentTheme().Cfg.ColorAccent
	if e.Accent != nil {
		accent = *e.Accent
	}
	switch {
	case !ok:
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{gogui.Label(say.T("tessera.card.loading"), secondary())},
		}))
	case (h.Line && len(h.Values) == 0) || (!h.Line && len(h.Runs) == 0):
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{gogui.Label(say.T("tessera.card.empty"), secondary())},
		}))
	default:
		out = append(out, v.graph(h, width, height, accent, l.Clock24))
	}
	if d := t.Domain(); d == "number" || d == "input_number" {
		out = append(out, v.slider(w, "tessera-card", t, e, width))
	}
	keys := make([]gogui.View, 0, len(ranges))
	for _, r := range ranges {
		keys = append(
			keys,
			v.wordKey("tessera-range-"+strconv.Itoa(r), rangeName(r), reach*2, r == hours,
				func(e gogui.EventCtx) {
					v.pickRange(t, r)
					e.Window.InvalidateLayout()
				}),
		)
	}
	return append(out, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.2),
		Content: keys,
	}))
}

func summary(h tessera.History, h24 bool) string {
	if h.Line {
		var parts []string
		if h.High != nil {
			parts = append(parts, say.F("tessera.card.high", map[string]any{
				"V": strconv.FormatFloat(h.High.Value, 'f', h.Decimals, 64) + h.Unit,
				"T": clock(h.High.At, h.Offset, h24, h.Hours == 168),
			}))
		}
		if h.Low != nil {
			parts = append(parts, say.F("tessera.card.low", map[string]any{
				"V": strconv.FormatFloat(h.Low.Value, 'f', h.Decimals, 64) + h.Unit,
				"T": clock(h.Low.At, h.Offset, h24, h.Hours == 168),
			}))
		}
		out := ""
		for i, p := range parts {
			if i > 0 {
				out += " · "
			}
			out += p
		}
		return out
	}
	if h.Active >= 0 && h.Began > 0 {
		return say.F("tessera.card.since", map[string]any{
			"T": clock(h.Began, h.Offset, h24, time.Now().Unix()-h.Began > 86400),
		})
	}
	return ""
}

func (v *View) options(w *gogui.Window, t tessera.Tile, width float32) gogui.View {
	rows := []gogui.View{}
	for i, o := range t.Texts("options") {
		rows = append(rows, v.wordKey("tessera-option-"+strconv.Itoa(i), o, width, o == t.State,
			func(e gogui.EventCtx) {
				v.send(w, t, t.Domain()+".select_option", map[string]string{"option": o}, nil, o)
				e.Window.InvalidateLayout()
			}))
	}
	return gogui.Column(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(style.Reach() * 0.15),
		Content: rows,
	})
}

func (v *View) lockKeys(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) gogui.View {
	reach := style.Reach()
	key := (width - reach*0.4) / 3
	lock := func(gogui.EventCtx) { v.act(w, t, "lock.lock", nil, nil) }
	unlock := func(ev gogui.EventCtx) {
		v.guard(w, t)
		ev.Window.InvalidateLayout()
	}
	if !e.Available {
		lock, unlock = nil, nil
	}
	keys := []gogui.View{
		v.wordKey("tessera-lock", say.T("tessera.lock.lock"), key, t.State == "locked", lock),
		v.wordKey(
			"tessera-unlock",
			say.T("tessera.lock.unlock"),
			key,
			t.State == "unlocked",
			unlock,
		),
	}
	if f, ok := t.Number("supported_features"); ok && int(f)&lockOpen != 0 {
		var open func(gogui.EventCtx)
		if e.Available {
			open = func(gogui.EventCtx) { v.act(w, t, "lock.open", nil, nil) }
		}
		keys = append(
			keys,
			v.wordKey("tessera-open", say.T("tessera.lock.open.key"), key, false, open),
		)
	}
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.2),
		Content: keys,
	})
}
