package widgets

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	alarmHome     = 1
	alarmAway     = 2
	alarmNight    = 4
	alarmBypass   = 16
	alarmVacation = 32
	vacuumLocate  = 512
	codeMost      = 10
)

type code struct {
	entity, action string
	digits         string
}

func pretty(s string) string {
	s = strings.ReplaceAll(s, "_", " ")
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (v *View) power(w *gogui.Window, t tessera.Tile, e kit.EntityTile, on bool) gogui.View {
	action := t.Domain() + ".turn_on"
	if on {
		action = t.Domain() + ".turn_off"
	}
	return v.o.UI.Presses.Pressable(gogui.Row, gogui.ContainerCfg{
		ID:       "tessera-card-power",
		Sizing:   gogui.FitFit,
		Padding:  gogui.NoPadding,
		Disabled: !e.Available,
		Content:  []gogui.View{v.o.UI.Kit().Toggle("tessera-card-power-switch", on)},
	}, func(gogui.EventCtx) { v.send(w, t, action, nil, nil, "") })
}

func (v *View) choices(
	w *gogui.Window,
	id string,
	options []string,
	current string,
	width float32,
	enabled bool,
	do func(string),
) gogui.View {
	reach := style.Reach()
	gap := reach * 0.15
	per := max(1, min(len(options), int((width+gap)/(reach*3+gap))))
	key := (width - float32(per-1)*gap) / float32(per)
	var rows []gogui.View
	var row []gogui.View
	for i, o := range options {
		var tap func(gogui.EventCtx)
		if enabled {
			tap = func(e gogui.EventCtx) {
				do(o)
				e.Window.InvalidateLayout()
			}
		}
		label := kit.Ellipsize(w, pretty(o), gogui.CurrentTheme().Cfg.TextStyleDef, key-reach*0.4)
		row = append(row, v.wordKey(id+"-"+strconv.Itoa(i), label, key, o == current, tap))
		if len(row) == per || i == len(options)-1 {
			rows = append(rows, gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FitFit,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingPx(gap),
				Content: row,
			}))
			row = nil
		}
	}
	return gogui.Column(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		HAlign:  gogui.HAlignCenter,
		Content: rows,
	})
}

func (v *View) setpoint(
	id, slot string,
	value, step, lo, hi float64,
	suffix string,
	enabled bool,
	commit func(float64),
) gogui.View {
	shown, _ := v.editingAt(slot, value)
	press := func(direction int) func(gogui.EventCtx) {
		if !enabled {
			return nil
		}
		return func(e gogui.EventCtx) {
			v.stepAt(slot, stepValue(shown, step, lo, hi, direction), commit)
			e.Window.InvalidateLayout()
		}
	}
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2.2
	reach := style.Reach()
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.5),
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{
			v.panelKey(id+"-down", key{glyph: 0xF0374}, reach*1.2, press(-1)),
			gogui.Label(formatStep(shown, step)+suffix, big),
			v.panelKey(id+"-up", key{glyph: 0xF0415}, reach*1.2, press(1)),
		},
	})
}

func number(t tessera.Tile, key string, fallback float64) float64 {
	if v, ok := t.Number(key); ok {
		return v
	}
	return fallback
}

func (v *View) climateBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	enabled := e.Available
	step := number(t, "target_temp_step", 0.5)
	lo, hi := number(t, "min_temp", 7), number(t, "max_temp", 35)
	send := func(data map[string]string) { v.send(w, t, "climate.set_temperature", data, nil, "") }
	var out []gogui.View
	low, lowOK := t.Number("target_temp_low")
	high, highOK := t.Number("target_temp_high")
	target, targetOK := t.Number("temperature")
	switch {
	case lowOK && highOK:
		out = append(
			out,
			v.setpoint(
				"tessera-low",
				t.Entity+"/low",
				low,
				step,
				lo,
				hi,
				"°",
				enabled,
				func(x float64) {
					send(
						map[string]string{"target_temp_low": num(x), "target_temp_high": num(high)},
					)
				},
			),
			v.setpoint(
				"tessera-high",
				t.Entity+"/high",
				high,
				step,
				lo,
				hi,
				"°",
				enabled,
				func(x float64) {
					send(map[string]string{"target_temp_low": num(low), "target_temp_high": num(x)})
				},
			),
		)
	case targetOK:
		out = append(
			out,
			v.setpoint("tessera-target", t.Entity+"/target", target, step, lo, hi, "°", enabled,
				func(x float64) { send(map[string]string{"temperature": num(x)}) }),
		)
	}
	var modes []string
	for _, m := range t.Texts("hvac_modes") {
		if m != "off" {
			modes = append(modes, m)
		}
	}
	if len(modes) > 0 {
		out = append(
			out,
			v.choices(w, "tessera-mode", modes, t.State, width, enabled, func(m string) {
				v.send(w, t, "climate.set_hvac_mode", map[string]string{"hvac_mode": m}, nil, m)
			}),
		)
	}
	for _, row := range []struct{ list, current, action, field string }{
		{"fan_modes", "fan_mode", "climate.set_fan_mode", "fan_mode"},
		{"swing_modes", "swing_mode", "climate.set_swing_mode", "swing_mode"},
	} {
		if options := t.Texts(row.list); len(options) > 0 {
			out = append(
				out,
				v.choices(
					w,
					"tessera-"+row.field,
					options,
					t.Text(row.current),
					width,
					enabled,
					func(o string) { v.send(w, t, row.action, map[string]string{row.field: o}, nil, "") },
				),
			)
		}
	}
	return out
}

func climateStatus(t tessera.Tile, l tessera.Layout) string {
	parts := []string{}
	if a := t.Text("hvac_action"); a != "" {
		parts = append(parts, pretty(a))
	} else {
		parts = append(parts, word(t, l.Numbers))
	}
	if now, ok := t.Number("current_temperature"); ok {
		parts = append(
			parts,
			say.F("tessera.card.now", map[string]any{"T": l.Numbers.Localize(num(now)) + "°"}),
		)
	}
	if h, ok := t.Number("current_humidity"); ok {
		parts = append(parts, l.Numbers.Localize(num(h))+"%")
	}
	return strings.Join(parts, " · ")
}

func (v *View) humidifierBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	var out []gogui.View
	if target, ok := t.Number("humidity"); ok {
		lo, hi := number(t, "min_humidity", 0), number(t, "max_humidity", 100)
		out = append(
			out,
			v.setpoint(
				"tessera-humidity",
				t.Entity+"/humidity",
				target,
				1,
				lo,
				hi,
				"%",
				e.Available,
				func(x float64) {
					v.send(
						w,
						t,
						"humidifier.set_humidity",
						map[string]string{"humidity": num(x)},
						nil,
						"",
					)
				},
			),
		)
	}
	if modes := t.Texts("available_modes"); len(modes) > 0 {
		out = append(
			out,
			v.choices(
				w,
				"tessera-hmode",
				modes,
				t.Text("mode"),
				width,
				e.Available,
				func(m string) {
					v.send(w, t, "humidifier.set_mode", map[string]string{"mode": m}, nil, "")
				},
			),
		)
	}
	return out
}

func humidifierStatus(t tessera.Tile, l tessera.Layout) string {
	parts := []string{}
	if a := t.Text("action"); a != "" {
		parts = append(parts, pretty(a))
	} else {
		parts = append(parts, word(t, l.Numbers))
	}
	if now, ok := t.Number("current_humidity"); ok {
		parts = append(
			parts,
			say.F("tessera.card.now", map[string]any{"T": l.Numbers.Localize(num(now)) + "%"}),
		)
	}
	return strings.Join(parts, " · ")
}

func num(x float64) string { return strconv.FormatFloat(x, 'f', -1, 64) }

func (v *View) alarmBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	v.mu.Lock()
	pending := v.code
	v.mu.Unlock()
	if pending.entity == t.Entity {
		return v.keypad(w, t, width, pending)
	}
	needs := func(arm bool) bool {
		format := t.Text("code_format")
		if format == "" {
			return false
		}
		if arm {
			if required, ok := t.Bool("code_arm_required"); ok && !required {
				return false
			}
		}
		return true
	}
	run := func(action string, arm bool) func(gogui.EventCtx) {
		if !e.Available {
			return nil
		}
		return func(ev gogui.EventCtx) {
			if needs(arm) {
				v.mu.Lock()
				v.code = code{entity: t.Entity, action: action}
				v.mu.Unlock()
			} else {
				v.send(w, t, action, nil, nil, "")
			}
			ev.Window.InvalidateLayout()
		}
	}
	reach := style.Reach()
	gap := reach * 0.2
	half := (width - gap) / 2
	switch t.State {
	case "arming", "pending", "triggered":
		return []gogui.View{v.wordKey("tessera-disarm", say.T("tessera.alarm.disarm"), width, true,
			run("alarm_control_panel.alarm_disarm", false))}
	}
	features := int(number(t, "supported_features", alarmHome|alarmAway))
	var keys []gogui.View
	for _, m := range []struct {
		bit           int
		state, action string
	}{
		{alarmHome, "armed_home", "alarm_arm_home"},
		{alarmAway, "armed_away", "alarm_arm_away"},
		{alarmNight, "armed_night", "alarm_arm_night"},
		{alarmVacation, "armed_vacation", "alarm_arm_vacation"},
		{alarmBypass, "armed_custom_bypass", "alarm_arm_custom_bypass"},
	} {
		if features&m.bit != 0 {
			keys = append(keys, v.wordKey("tessera-"+m.state, say.T("tessera.alarm."+m.state), half,
				t.State == m.state, run("alarm_control_panel."+m.action, true)))
		}
	}
	if t.State != "disarmed" {
		keys = append(keys, v.wordKey("tessera-disarm", say.T("tessera.alarm.disarm"), half, false,
			run("alarm_control_panel.alarm_disarm", false)))
	}
	return []gogui.View{pairs(keys, gap)}
}

func pairs(keys []gogui.View, gap float32) gogui.View {
	var rows []gogui.View
	for i := 0; i < len(keys); i += 2 {
		rows = append(rows, gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FitFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap),
			Content: keys[i:min(i+2, len(keys))],
		}))
	}
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: rows,
	})
}

func (v *View) keypad(w *gogui.Window, t tessera.Tile, width float32, c code) []gogui.View {
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2
	if t.Text("code_format") == "text" {
		return []gogui.View{
			gogui.Label(say.T("tessera.alarm.text"), secondary()),
			v.wordKey(
				"tessera-code-cancel",
				say.T("tessera.alarm.cancel"),
				width,
				false,
				func(e gogui.EventCtx) {
					v.forgetCode()
					e.Window.InvalidateLayout()
				},
			),
		}
	}
	reach := style.Reach()
	gap := reach * 0.2
	key := min(reach*2, (width-2*gap)/3)
	press := func(label string, do func()) gogui.View {
		return v.wordKey("tessera-code-"+label, label, key, false, func(e gogui.EventCtx) {
			do()
			e.Window.InvalidateLayout()
		})
	}
	digit := func(d string) gogui.View {
		return press(d, func() {
			v.mu.Lock()
			if len(v.code.digits) < codeMost {
				v.code.digits += d
			}
			v.mu.Unlock()
		})
	}
	var rows []gogui.View
	for r := 0; r < 3; r++ {
		row := []gogui.View{}
		for col := 1; col <= 3; col++ {
			row = append(row, digit(strconv.Itoa(r*3+col)))
		}
		rows = append(rows, gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FitFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap),
			Content: row,
		}))
	}
	rows = append(rows, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: []gogui.View{
			press(say.T("tessera.alarm.clear"), func() {
				v.mu.Lock()
				v.code.digits = ""
				v.mu.Unlock()
			}),
			digit("0"),
			press(say.T("tessera.alarm.ok"), func() {
				v.mu.Lock()
				sent := v.code
				v.code = code{}
				v.mu.Unlock()
				if sent.digits != "" {
					v.send(w, t, sent.action, map[string]string{"code": sent.digits}, nil, "")
				}
			}),
		},
	}))
	dots := strings.Repeat("●", len(c.digits)) + strings.Repeat("○", max(0, 4-len(c.digits)))
	return append([]gogui.View{gogui.Label(dots, big)}, gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: rows,
	}))
}

func (v *View) forgetCode() {
	v.mu.Lock()
	v.code = code{}
	v.mu.Unlock()
}

func playTime(seconds float64) string {
	s := int(max(0, seconds))
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func (v *View) mediaBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
	now time.Time,
) []gogui.View {
	reach := style.Reach()
	var out []gogui.View
	if c := v.cover(w, t, width, now); c != nil {
		out = append(out, c)
	}
	title := gogui.CurrentTheme().Cfg.TextStyleDef
	title.Size *= 1.4
	if s := t.Text("media_title"); s != "" {
		out = append(out, gogui.Label(kit.Ellipsize(w, s, title, width), title))
	}
	if s := t.ExtraText("artist"); s != "" {
		out = append(out, gogui.Label(kit.Ellipsize(w, s, secondary(), width), secondary()))
	}
	if dur, ok := t.ExtraNumber("dur"); ok && dur > 0 {
		pos, _ := t.ExtraNumber("pos")
		if at, ok := t.ExtraNumber("at"); ok && t.State == "playing" {
			pos += float64(now.Unix()) - at
			v.tick(w, now, time.Second)
		}
		pos = min(max(pos, 0), dur)
		out = append(out, v.progress(width, float32(pos/dur)),
			gogui.Label(playTime(pos)+" / "+playTime(dur), secondary()))
	}
	enabled := e.Available
	call := func(action string) func(gogui.EventCtx) {
		if !enabled {
			return nil
		}
		return func(ev gogui.EventCtx) {
			v.send(w, t, "media_player."+action, nil, nil, "")
			ev.Window.InvalidateLayout()
		}
	}
	middle := key{glyph: 0xF040A}
	if t.State == "playing" {
		middle = key{glyph: 0xF03E4}
	}
	out = append(out, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.3),
		Content: []gogui.View{
			v.panelKey(
				"tessera-media-prev",
				key{glyph: 0xF04AE},
				reach*1.4,
				call("media_previous_track"),
			),
			v.panelKey("tessera-media-play", middle, reach*1.8, call("media_play_pause")),
			v.panelKey(
				"tessera-media-next",
				key{glyph: 0xF04AD},
				reach*1.4,
				call("media_next_track"),
			),
		},
	}))
	if _, ok := t.Number("volume_level"); ok {
		muted, _ := t.Bool("is_volume_muted")
		glyph := rune(0xF057E)
		if muted {
			glyph = 0xF075F
		}
		var mute func(gogui.EventCtx)
		if enabled {
			mute = func(ev gogui.EventCtx) {
				v.send(w, t, "media_player.volume_mute",
					map[string]string{"is_volume_muted": strconv.FormatBool(!muted)}, nil, "")
				ev.Window.InvalidateLayout()
			}
		}
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.3),
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{
				v.panelKey("tessera-media-mute", key{glyph: glyph, checked: muted}, reach, mute),
				v.slider(w, "tessera-card", t, e, width-reach*1.3),
			},
		}))
	}
	if sources := t.ExtraTexts("sl"); len(sources) > 0 {
		out = append(
			out,
			v.choices(
				w,
				"tessera-source",
				sources,
				t.ExtraText("so"),
				width,
				enabled,
				func(s string) {
					v.send(
						w,
						t,
						"media_player.select_source",
						map[string]string{"source": s},
						nil,
						"",
					)
				},
			),
		)
	}
	return out
}

func (v *View) progress(width, done float32) gogui.View {
	pal := v.o.UI.Palette()
	track := gogui.RGB(pal.Muted.R, pal.Muted.G, pal.Muted.B).WithOpacity(0.25)
	fill := gogui.CurrentTheme().Cfg.ColorAccent
	height := max(4, style.Reach()*0.08)
	return gogui.DrawCanvas(gogui.DrawCanvasCfg{
		ID:      "tessera-media-progress",
		Width:   width,
		Height:  height,
		Sizing:  gogui.FixedFixed,
		Version: uint64(done * 10000),
		OnDraw: func(dc *gogui.DrawContext) {
			dc.FilledRoundedRect(0, 0, width, height, height/2, track)
			dc.FilledRoundedRect(0, 0, max(height, width*done), height, height/2, fill)
		},
	})
}

func (v *View) weatherBody(
	w *gogui.Window,
	t tessera.Tile,
	l tessera.Layout,
	width float32,
) []gogui.View {
	reach := style.Reach()
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2.4
	symbol := weatherSymbol
	unit := t.Text("temperature_unit")
	temp := func(x float64) string { return l.Numbers.Localize(num(math.Round(x))) + "°" }
	now := []gogui.View{symbol(t.State, big.Size)}
	if x, ok := t.Number("temperature"); ok {
		now = append(now, gogui.Label(temp(x)+strings.TrimPrefix(unit, "°"), big))
	}
	out := []gogui.View{gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(reach * 0.4),
		VAlign:  gogui.VAlignMiddle,
		Content: now,
	})}
	var details []string
	if x, ok := t.Number("apparent_temperature"); ok {
		details = append(details, say.F("tessera.weather.feels", map[string]any{"T": temp(x)}))
	}
	if x, ok := t.Number("humidity"); ok {
		details = append(details, l.Numbers.Localize(num(x))+"%")
	}
	if x, ok := t.Number("wind_speed"); ok {
		details = append(details, say.F("tessera.weather.wind", map[string]any{
			"W": l.Numbers.Localize(num(math.Round(x))) + " " + t.Text("wind_speed_unit"),
		}))
	}
	if len(details) > 0 {
		out = append(out, gogui.Label(strings.Join(details, " · "), secondary()))
	}
	if hours := t.ExtraList("hours"); len(hours) > 0 {
		hours = hours[:min(len(hours), max(1, int(width/(reach*1.5))))]
		cols := []gogui.View{}
		cell := width / float32(len(hours))
		for _, h := range hours {
			c, _ := h["c"].(string)
			at, _ := h["t"].(string)
			col := []gogui.View{gogui.Label(at, secondary()), symbol(c, reach*0.7)}
			if x, ok := h["h"].(float64); ok {
				col = append(col, gogui.Label(temp(x), gogui.CurrentTheme().Cfg.TextStyleDef))
			}
			cols = append(cols, gogui.Column(gogui.ContainerCfg{
				Width:   cell,
				Sizing:  gogui.FixedFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: col,
			}))
		}
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.NoSpacing,
			Content: cols,
		}))
	}
	days := t.ExtraList("days")
	if len(days) == 0 {
		return append(out, gogui.Label(say.T("tessera.weather.none"), secondary()))
	}
	for _, d := range days {
		name, _ := d["d"].(string)
		c, _ := d["c"].(string)
		var hl []string
		if x, ok := d["h"].(float64); ok {
			hl = append(hl, temp(x))
		}
		if x, ok := d["l"].(float64); ok {
			hl = append(hl, temp(x))
		}
		rain := ""
		if x, ok := d["p"].(float64); ok && x > 0 {
			rain = l.Numbers.Localize(num(x)) + "%"
		}
		out = append(out, gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.3),
			VAlign:  gogui.VAlignMiddle,
			Content: []gogui.View{
				gogui.Row(gogui.ContainerCfg{
					Width:   reach * 2,
					Sizing:  gogui.FixedFit,
					Padding: gogui.NoPadding,
					Content: []gogui.View{gogui.Label(name, gogui.CurrentTheme().Cfg.TextStyleDef)},
				}),
				symbol(c, reach*0.7),
				gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding,
					Content: []gogui.View{gogui.Label(rain, secondary())}}),
				gogui.Label(strings.Join(hl, " / "), gogui.CurrentTheme().Cfg.TextStyleDef),
			},
		}))
	}
	return out
}

func (v *View) vacuumBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	reach := style.Reach()
	gap := reach * 0.2
	call := func(action string) func(gogui.EventCtx) {
		if !e.Available {
			return nil
		}
		return func(ev gogui.EventCtx) {
			v.send(w, t, "vacuum."+action, nil, nil, "")
			ev.Window.InvalidateLayout()
		}
	}
	var out []gogui.View
	if b, ok := t.Number("battery_level"); ok {
		out = append(
			out,
			gogui.Label(say.F("tessera.vacuum.battery", map[string]any{"N": int(b)}), secondary()),
		)
	}
	main, label := "start", say.T("tessera.vacuum.start")
	if t.State == "cleaning" {
		main, label = "pause", say.T("tessera.vacuum.pause")
	}
	keys := []gogui.View{
		v.wordKey("tessera-vacuum-main", label, reach*3, t.State == "cleaning", call(main)),
		v.wordKey(
			"tessera-vacuum-dock",
			say.T("tessera.vacuum.dock"),
			reach*3,
			false,
			call("return_to_base"),
		),
	}
	if int(number(t, "supported_features", 0))&vacuumLocate != 0 {
		keys = append(
			keys,
			v.wordKey("tessera-vacuum-locate", say.T("tessera.vacuum.locate"), reach*3, false,
				call("locate")),
		)
	}
	out = append(out, gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(gap),
		Content: keys,
	}))
	if speeds := t.Texts("fan_speed_list"); len(speeds) > 0 {
		out = append(
			out,
			v.choices(w, "tessera-suction", speeds, t.Text("fan_speed"), width, e.Available,
				func(s string) {
					v.send(w, t, "vacuum.set_fan_speed", map[string]string{"fan_speed": s}, nil, "")
				}),
		)
	}
	return out
}

func (v *View) remoteBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	var out []gogui.View
	if activities := t.Texts("activity_list"); len(activities) > 0 {
		out = append(
			out,
			v.choices(
				w,
				"tessera-activity",
				activities,
				t.Text("current_activity"),
				width,
				e.Available,
				func(a string) {
					v.send(w, t, "remote.turn_on", map[string]string{"activity": a}, nil, "")
				},
			),
		)
	}
	var commands []string
	for _, c := range t.ExtraTexts("keys") {
		if c != "" {
			commands = append(commands, c)
		}
	}
	if len(commands) > 0 {
		out = append(
			out,
			v.choices(w, "tessera-command", commands, "", width, e.Available, func(c string) {
				v.send(w, t, "remote.send_command", map[string]string{"command": c}, nil, "")
			}),
		)
	}
	return out
}

func weatherSymbol(condition string, size float32) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Family = icons.Family
	st.Size = size
	r, ok := weatherIcons[condition]
	if !ok {
		r = 0xF0590
	}
	return gogui.Label(icons.Glyph(r), st)
}

func sunBody(t tessera.Tile) []gogui.View {
	var out []gogui.View
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	if s := t.ExtraText("rise"); s != "" {
		out = append(out, gogui.Label(say.F("tessera.sun.rise", map[string]any{"T": s}), st))
	}
	if s := t.ExtraText("set"); s != "" {
		out = append(out, gogui.Label(say.F("tessera.sun.set", map[string]any{"T": s}), st))
	}
	return out
}

func timerBody(t tessera.Tile) []gogui.View {
	big := gogui.CurrentTheme().Cfg.TextStyleDef
	big.Size *= 2.2
	if s := t.ExtraText("rem"); s != "" {
		return []gogui.View{gogui.Label(s, big)}
	}
	if s := t.ExtraText("dur"); s != "" {
		return []gogui.View{gogui.Label(s, big)}
	}
	return nil
}
