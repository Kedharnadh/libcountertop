package widgets

import (
	"math"
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	holdMost   = 8 * time.Second
	holdNear   = 0.03
	tapRelease = 500 * time.Millisecond
)

type grip struct {
	value  float64
	moved  time.Time
	sent   time.Time
	commit *time.Timer
}

func sliding(t tessera.Tile) bool {
	if t.Inline != "slider" || t.Display == "watch" {
		return false
	}
	switch t.Domain() {
	case "light", "fan", "cover", "number", "input_number":
		return true
	}
	return false
}

func level(t tessera.Tile) (float64, bool) {
	switch t.Domain() {
	case "light":
		b, ok := t.Number("brightness")
		return b / 255, ok || t.State == "off"
	case "fan":
		p, ok := t.Number("percentage")
		return p / 100, ok || t.State == "off"
	case "cover":
		p, ok := t.Number("current_position")
		if !ok {
			return 1, false
		}
		return (100 - p) / 100, true
	case "media_player":
		v, ok := t.Number("volume_level")
		return v, ok
	case "number", "input_number":
		lo, hi := bounds(t)
		v, err := strconv.ParseFloat(t.State, 64)
		if err != nil || hi <= lo {
			return 0, false
		}
		return (v - lo) / (hi - lo), true
	}
	return 0, false
}

func bounds(t tessera.Tile) (lo, hi float64) {
	lo, hi = 0, 100
	if v, ok := t.Number("min"); ok {
		lo = v
	}
	if v, ok := t.Number("max"); ok {
		hi = v
	}
	return lo, hi
}

type dial struct {
	key    string
	real   float64
	live   bool
	commit func(*gogui.Window, float64)
}

func (v *View) levelDial(t tessera.Tile) dial {
	real, _ := level(t)
	return dial{
		key:    t.Entity,
		real:   real,
		live:   available(t),
		commit: func(w *gogui.Window, x float64) { v.commitLevel(w, t, x) },
	}
}

func (v *View) held(d dial, now time.Time) float64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	g, ok := v.grips[d.key]
	switch {
	case !ok:
		return d.real
	case g.sent.IsZero():
		return g.value
	case now.Sub(g.sent) > holdMost || math.Abs(d.real-g.value) <= holdNear || !d.live:
		delete(v.grips, d.key)
		return d.real
	}
	return g.value
}

func (v *View) slider(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f kit.EntityTile,
	width float32,
) gogui.View {
	d := v.levelDial(t)
	value := v.held(d, time.Now())
	if t.Domain() == "light" && t.State != "on" {
		value = 0
	}
	var fill *gogui.Color
	if f.On {
		fill = f.Accent
	}
	return v.dialSlider(w, id, d, value, width, fill, !f.Available)
}

func (v *View) dialSlider(
	w *gogui.Window,
	id string,
	d dial,
	value float64,
	width float32,
	fill *gogui.Color,
	disabled bool,
) gogui.View {
	thumb := gogui.CurrentTheme().Cfg.SizeSliderThumb
	cfg := gogui.SliderCfg{
		ID:     id + "-slider",
		Width:  max(1, width-thumb),
		Height: style.Reach(),
		Sizing: gogui.FixedFit,
		Value:  float32(value * 1000),
		Max:    1000,
		OnChange: func(x float32, e gogui.EventCtx) {
			v.nudge(e.Window, d, float64(x)/1000)
			e.Window.InvalidateLayout()
		},
		Disabled: disabled,
	}
	if fill != nil {
		cfg.ColorLeft = *fill
	}
	return gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FitFit,
		Padding: gogui.PadVH(0, thumb/2),
		Spacing: gogui.NoSpacing,
		Content: []gogui.View{v.o.UI.Toolkit.Slider(cfg)},
		OnGesture: func(e gogui.EventCtx) {
			if e.Event == nil || e.Event.GestureType != gogui.GesturePan {
				return
			}
			e.Consume()
			switch e.Event.GesturePhase {
			case gogui.GesturePhaseEnded, gogui.GesturePhaseCancelled:
				v.letGo(w, d)
			}
		},
	})
}

func (v *View) nudge(w *gogui.Window, d dial, value float64) {
	v.mu.Lock()
	g, ok := v.grips[d.key]
	if !ok || !g.sent.IsZero() {
		g = &grip{}
		v.grips[d.key] = g
	}
	g.value, g.moved = min(max(value, 0), 1), time.Now()
	if g.commit != nil {
		g.commit.Stop()
	}
	g.commit = time.AfterFunc(tapRelease, func() {
		w.QueueCommand(func(w *gogui.Window) { v.letGo(w, d) })
	})
	v.mu.Unlock()
}

func (v *View) letGo(w *gogui.Window, d dial) {
	v.mu.Lock()
	g, ok := v.grips[d.key]
	if !ok || !g.sent.IsZero() {
		v.mu.Unlock()
		return
	}
	if g.commit != nil {
		g.commit.Stop()
	}
	g.sent = time.Now()
	value := g.value
	v.mu.Unlock()
	d.commit(w, value)
	w.InvalidateLayout()
}

func (v *View) commitLevel(w *gogui.Window, t tessera.Tile, value float64) {
	if action, data, wish := levelAction(t, value); action != "" {
		v.send(w, t, action, data, nil, wish)
	}
}

func levelAction(
	t tessera.Tile,
	value float64,
) (action string, data map[string]string, wish string) {
	if t.State == "off" && t.Domain() != "media_player" {
		wish = "on"
	}
	switch t.Domain() {
	case "light":
		b := max(3, int(math.Round(value*255)))
		return "light.turn_on", map[string]string{"brightness": strconv.Itoa(b)}, wish
	case "fan":
		p := int(math.Round(value * 100))
		return "fan.set_percentage", map[string]string{"percentage": strconv.Itoa(p)}, wish
	case "cover":
		p := 100 - int(math.Round(value*100))
		return "cover.set_cover_position", map[string]string{"position": strconv.Itoa(p)}, ""
	case "media_player":
		return "media_player.volume_set", map[string]string{
			"volume_level": strconv.FormatFloat(math.Round(value*100)/100, 'f', -1, 64),
		}, ""
	case "number", "input_number":
		lo, hi := bounds(t)
		step := 1.0
		if s, ok := t.Number("step"); ok && s > 0 {
			step = s
		}
		if hi <= lo {
			return "", nil, ""
		}
		x := min(max(lo+math.Round(value*(hi-lo)/step)*step, lo), hi)
		return t.Domain() + ".set_value", map[string]string{
			"value": strconv.FormatFloat(x, 'f', -1, 64),
		}, ""
	}
	return "", nil, ""
}
