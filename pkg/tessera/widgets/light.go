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
		hue := 0.0
		if hs := t.Numbers("hs_color"); len(hs) == 2 {
			hue = hs[0]
		}
		d := dial{
			key:  t.Entity + "/hue",
			real: hue / 360,
			live: e.Available,
			commit: func(w *gogui.Window, x float64) {
				h := int(math.Round(x * 360))
				v.send(w, t, "light.turn_on", nil,
					map[string]string{"hs_color": fmt.Sprintf("[%d,100]", h)}, on)
			},
		}
		x := v.held(d, now)
		c := lampColor(x*360, 100, v.o.UI.Palette())
		fill := gogui.RGB(c.R, c.G, c.B)
		out = append(out, v.labelled(say.T("tessera.light.hue"),
			strconv.Itoa(int(math.Round(x*360)))+"°", width,
			v.dialSlider(w, "tessera-hue", d, x, width, &fill, !e.Available)))
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
