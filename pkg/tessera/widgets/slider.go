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

func (v *View) shownLevel(t tessera.Tile, now time.Time) float64 {
	real, _ := level(t)
	v.mu.Lock()
	defer v.mu.Unlock()
	g, ok := v.grips[t.Entity]
	switch {
	case !ok:
		return real
	case g.sent.IsZero():
		return g.value
	case now.Sub(g.sent) > holdMost || math.Abs(real-g.value) <= holdNear || !available(t):
		delete(v.grips, t.Entity)
		return real
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
	height := style.Reach()
	value := v.shownLevel(t, time.Now())
	if t.Domain() == "light" && t.State != "on" {
		value = 0
	}
	thumb := gogui.CurrentTheme().Cfg.SizeSliderThumb
	cfg := gogui.SliderCfg{
		ID:     id + "-slider",
		Width:  max(1, width-thumb),
		Height: height,
		Sizing: gogui.FixedFit,
		Value:  float32(value * 1000),
		Max:    1000,
		OnChange: func(x float32, e gogui.EventCtx) {
			v.nudge(e.Window, t, float64(x)/1000)
			e.Window.InvalidateLayout()
		},
		Disabled: !f.Available,
	}
	if f.On && f.Accent != nil {
		cfg.ColorLeft = *f.Accent
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
				v.letGo(w, t)
			}
		},
	})
}

func (v *View) nudge(w *gogui.Window, t tessera.Tile, value float64) {
	v.mu.Lock()
	g, ok := v.grips[t.Entity]
	if !ok || !g.sent.IsZero() {
		g = &grip{}
		v.grips[t.Entity] = g
	}
	g.value, g.moved = min(max(value, 0), 1), time.Now()
	if g.commit != nil {
		g.commit.Stop()
	}
	g.commit = time.AfterFunc(tapRelease, func() {
		w.QueueCommand(func(w *gogui.Window) { v.letGo(w, t) })
	})
	v.mu.Unlock()
}

func (v *View) letGo(w *gogui.Window, t tessera.Tile) {
	v.mu.Lock()
	g, ok := v.grips[t.Entity]
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
	v.commitLevel(w, t, value)
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
	if t.State == "off" {
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
