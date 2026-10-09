package widgets

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func lightModes(t tessera.Tile) []string { return t.Texts("supported_color_modes") }

func lightDims(t tessera.Tile) bool {
	m := lightModes(t)
	return len(m) != 1 || m[0] != "onoff"
}

func lightHue(t tessera.Tile) bool {
	return slices.ContainsFunc(lightModes(t), func(m string) bool {
		switch m {
		case "hs", "rgb", "rgbw", "rgbww", "xy":
			return true
		}
		return false
	})
}

func lightWarmth(t tessera.Tile) bool { return slices.Contains(lightModes(t), "color_temp") }

func (v *View) lightBody(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
) []gogui.View {
	var out []gogui.View
	now := time.Now()
	on := wishFor(t)
	if lightDims(t) {
		b := v.held(v.levelDial(t), now)
		if t.State != "on" {
			b = 0
		}
		out = append(out, v.labelled(say.T("tessera.light.brightness"),
			strconv.Itoa(int(math.Round(b*100)))+" %", width,
			v.slider(w, "tessera-card", t, e, width)))
	}
	if lightHue(t) {
		out = append(out, v.colorWheel(w, t, e, width, on)...)
	}
	if lightWarmth(t) {
		lo, hi := number(t, "min_color_temp_kelvin", 0), number(t, "max_color_temp_kelvin", 0)
		k, known := t.Number("color_temp_kelvin")
		ready := hi > lo
		real := 0.0
		if ready && known {
			real = (k - lo) / (hi - lo)
		}
		d := dial{
			key:  t.Entity + "/kelvin",
			real: real,
			live: e.Available,
			commit: func(w *gogui.Window, x float64) {
				kelvin := int(math.Round(lo + x*(hi-lo)))
				v.send(w, t, "light.turn_on",
					map[string]string{"color_temp_kelvin": strconv.Itoa(kelvin)}, nil, on)
			},
		}
		x := v.held(d, now)
		says := say.T("tessera.light.waiting")
		if ready {
			says = strconv.Itoa(int(math.Round(lo+x*(hi-lo)))) + " K"
		}
		out = append(out, v.labelled(say.T("tessera.light.warmth"), says, width,
			v.dialSlider(w, "tessera-kelvin", d, x, width, nil, !e.Available || !ready)))
	}
	return out
}

func wishFor(t tessera.Tile) string {
	if t.State == "off" {
		return "on"
	}
	return ""
}

func (v *View) labelled(name, value string, width float32, control gogui.View) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Width:   width,
		Sizing:  gogui.FixedFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingPx(style.Reach() * 0.1),
		Content: []gogui.View{
			gogui.Row(gogui.ContainerCfg{
				Width:   width,
				Sizing:  gogui.FixedFit,
				Padding: gogui.NoPadding,
				Content: []gogui.View{
					gogui.Label(name, gogui.CurrentTheme().Cfg.TextStyleDef),
					gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding}),
					gogui.Label(value, secondary()),
				},
			}),
			control,
		},
	})
}

type preset struct {
	hue, sat float64
	kelvin   float64
}

var presets = []preset{
	{kelvin: 2700, hue: 35, sat: 40},
	{hue: 0, sat: 100},
	{hue: 25, sat: 100},
	{hue: 50, sat: 100},
	{hue: 120, sat: 100},
	{hue: 185, sat: 100},
	{hue: 225, sat: 100},
	{hue: 275, sat: 100},
	{hue: 320, sat: 50},
}

func (v *View) colorWheel(
	w *gogui.Window,
	t tessera.Tile,
	e kit.EntityTile,
	width float32,
	on string,
) []gogui.View {
	hue, sat := 0.0, 0.0
	if hs := t.Numbers("hs_color"); len(hs) == 2 {
		hue, sat = hs[0], hs[1]
	}
	satDial := dial{
		key:    t.Entity + "/sat",
		real:   sat / 100,
		live:   e.Available,
		commit: func(*gogui.Window, float64) {},
	}
	hueDial := dial{
		key:  t.Entity + "/hue",
		real: hue / 360,
		live: e.Available,
		commit: func(w *gogui.Window, x float64) {
			s := v.held(satDial, time.Now())
			v.send(w, t, "light.turn_on", nil, map[string]string{
				"hs_color": fmt.Sprintf("[%d,%d]", int(math.Round(x*360)), int(math.Round(s*100))),
			}, on)
		},
	}
	now := time.Now()
	h, s := v.held(hueDial, now), v.held(satDial, now)
	reach := style.Reach()
	size := min(width*0.7, reach*8)
	wheel := gogui.ColorWheel(gogui.ColorWheelCfg{
		ID:            "tessera-wheel",
		Value:         gogui.HSLA{H: float32(h * 360), S: float32(s), L: 0.5, A: 1},
		Size:          size,
		MarkerSize:    reach * 0.55,
		FocusDisabled: true,
		OnChange: func(c gogui.HSLA, ev gogui.EventCtx) {
			if !e.Available {
				return
			}
			v.nudge(ev.Window, satDial, float64(c.S))
			v.nudge(ev.Window, hueDial, float64(c.H)/360)
			ev.Window.InvalidateLayout()
		},
	})
	var swatches []gogui.View
	side := min(reach*1.1, (width-reach*0.2*float32(len(presets)-1))/float32(len(presets)))
	warm := lightWarmth(t)
	for i, p := range presets {
		swatches = append(swatches, gogui.ColorSwatch(gogui.ColorSwatchCfg{
			ID:     fmt.Sprintf("tessera-swatch-%d", i),
			Color:  p.color(),
			Width:  side,
			Height: side,
			Radius: gogui.RadiusLarge,
			OnClick: func(ev gogui.EventCtx) {
				ev.Consume()
				if !e.Available {
					return
				}
				if p.kelvin > 0 && warm {
					lo := number(t, "min_color_temp_kelvin", 0)
					hi := number(t, "max_color_temp_kelvin", 0)
					k := p.kelvin
					if hi > lo {
						k = min(max(k, lo), hi)
					}
					v.send(ev.Window, t, "light.turn_on",
						map[string]string{"color_temp_kelvin": strconv.Itoa(int(k))}, nil, on)
					return
				}
				v.nudge(ev.Window, satDial, p.sat/100)
				v.nudge(ev.Window, hueDial, p.hue/360)
				v.letGo(ev.Window, hueDial)
				v.letGo(ev.Window, satDial)
			},
		}))
	}
	return []gogui.View{
		v.labelled(say.T("tessera.light.hue"),
			fmt.Sprintf("%d° · %d %%", int(math.Round(h*360)), int(math.Round(s*100))), width,
			gogui.Row(gogui.ContainerCfg{
				Width:   width,
				Sizing:  gogui.FixedFit,
				Padding: gogui.PadVH(reach*0.2, 0),
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{wheel},
			})),
		gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.2),
			HAlign:  gogui.HAlignCenter,
			Content: swatches,
		}),
	}
}

func (p preset) color() gogui.Color {
	light := float32(0.5)
	if p.sat < 100 {
		light = 0.75
	}
	return gogui.HSLA{H: float32(p.hue), S: float32(p.sat) / 100, L: light, A: 1}.Color()
}
