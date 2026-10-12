package widgets

import (
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func (v *View) bodyFor(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	l tessera.Layout,
	c cell,
	f frame,
) func(width, height float32) gogui.View {
	if body := v.clockBody(w, id, t, l, f); body != nil {
		return body
	}
	if t.Entity == energyEntity && f.fresh && len(t.ExtraNumbers("p")) >= 6 {
		return v.energyBody(w, id, t, f)
	}
	if !f.fresh || !available(t) {
		return nil
	}
	if t.Domain() == "weather" && t.Display == "forecast" && len(t.ExtraList("days")) > 0 &&
		(c.columns >= 2 || c.rows >= 2) {
		return forecastBody(w, t, l)
	}
	if c.columns < 2 {
		return nil
	}
	switch {
	case t.Domain() == "sun" && t.Display == "sunpath" && t.ExtraText("rise") != "" &&
		t.ExtraText("set") != "":
		v.tick(w, f.now, time.Minute)
		return v.sunpathBody(w, id, t, f)
	}
	return nil
}

func forecastBody(
	w *gogui.Window,
	t tessera.Tile,
	l tessera.Layout,
) func(width, height float32) gogui.View {
	days := t.ExtraList("days")
	temp := func(x float64) string { return l.Numbers.Localize(num(math.Round(x))) + "°" }
	return func(width, height float32) gogui.View {
		reach := style.Reach()
		shown := days[:min(len(days), max(1, int(width/(reach*1.6))))]
		cell := width / float32(len(shown))
		text := gogui.CurrentTheme().Cfg.TextStyleDef
		cols := make([]gogui.View, 0, len(shown))
		for _, d := range shown {
			name, _ := d["d"].(string)
			cond, _ := d["c"].(string)
			col := []gogui.View{
				gogui.Label(kit.Ellipsize(w, name, secondary(), cell), secondary()),
				weatherSymbol(cond, min(reach, height*0.3)),
			}
			if x, ok := d["h"].(float64); ok {
				col = append(col, gogui.Label(temp(x), text))
			}
			if x, ok := d["l"].(float64); ok {
				col = append(col, gogui.Label(temp(x), secondary()))
			}
			if x, ok := d["p"].(float64); ok && x > 0 && height >= reach*3.2 {
				col = append(col, gogui.Label(l.Numbers.Localize(num(x))+"%", secondary()))
			}
			cols = append(cols, gogui.Column(gogui.ContainerCfg{
				Width:   cell,
				Sizing:  gogui.FixedFit,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingPx(reach * 0.08),
				HAlign:  gogui.HAlignCenter,
				Content: col,
			}))
		}
		return gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			Spacing: gogui.NoSpacing,
			VAlign:  gogui.VAlignMiddle,
			Content: cols,
		})
	}
}

func minutes(clock string) (int, bool) {
	h, m, ok := strings.Cut(clock, ":")
	hours, e1 := strconv.Atoi(h)
	mins, e2 := strconv.Atoi(m)
	if !ok || e1 != nil || e2 != nil {
		return 0, false
	}
	return hours*60 + mins, true
}

func sunFraction(rise, set, now int, day bool) float32 {
	start, end := set, rise
	if day {
		start, end = rise, set
	}
	span := (end - start + 1440) % 1440
	if span == 0 {
		span = 1
	}
	return min(max(float32((now-start+1440)%1440)/float32(span), 0), 1)
}

func (v *View) sunpathBody(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f frame,
) func(width, height float32) gogui.View {
	rise, _ := minutes(t.ExtraText("rise"))
	set, _ := minutes(t.ExtraText("set"))
	day := t.State == "above_horizon"
	fraction := sunFraction(rise, set, f.now.Hour()*60+f.now.Minute(), day)
	pal := f.pal
	path := gogui.RGB(pal.Muted.R, pal.Muted.G, pal.Muted.B)
	a := accent(t, pal)
	disc := gogui.RGB(a.R, a.G, a.B)
	name := t.Name
	if name == "" {
		name = say.T("tessera.sun.name")
	}
	return func(width, height float32) gogui.View {
		reach := style.Reach()
		text := gogui.CurrentTheme().Cfg.TextStyleDef
		line := text.Size * 1.4
		plot := max(1, height-2*line)
		hv := fnv.New64a()
		_, _ = fmt.Fprint(hv, width, plot, fraction, day, path, disc)
		canvas := gogui.DrawCanvas(gogui.DrawCanvasCfg{
			ID:      id + "-sun",
			Width:   width,
			Height:  plot,
			Sizing:  gogui.FixedFixed,
			Version: hv.Sum64(),
			OnDraw: func(dc *gogui.DrawContext) {
				size := reach * 0.25
				x0, x1 := size, width-size
				horizon := plot * 0.75
				lift := horizon - size
				if !day {
					lift = -(plot - horizon - size*0.5)
				}
				at := func(p float32) (float32, float32) {
					s := float32(math.Sin(math.Pi * float64(p)))
					return x0 + (x1-x0)*p, horizon - s*lift
				}
				dc.Line(0, horizon, width, horizon, path.WithOpacity(0.5), 2)
				const segments = 40
				arc := make([]float32, 0, 2*(segments+1))
				done := []float32{}
				for i := 0; i <= segments; i++ {
					p := float32(i) / segments
					x, y := at(p)
					arc = append(arc, x, y)
					if p <= fraction {
						done = append(done, x, y)
					}
				}
				sx, sy := at(fraction)
				done = append(done, sx, sy)
				dc.Polyline(arc, path.WithOpacity(0.6), max(1, reach*0.03))
				if len(done) >= 4 {
					dc.Polyline(done, disc, max(2, reach*0.06))
				}
				dc.FilledCircle(sx, sy, size*1.6, disc.WithOpacity(0.25))
				dc.FilledCircle(sx, sy, size, disc)
			},
		})
		times := gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			Content: []gogui.View{
				gogui.Label(
					say.F("tessera.sun.rise", map[string]any{"T": t.ExtraText("rise")}),
					secondary(),
				),
				gogui.Row(gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding}),
				gogui.Label(
					say.F("tessera.sun.set", map[string]any{"T": t.ExtraText("set")}),
					secondary(),
				),
			},
		})
		return gogui.Column(gogui.ContainerCfg{
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			Spacing: gogui.NoSpacing,
			Content: []gogui.View{
				gogui.Label(kit.Ellipsize(w, name, text, width), text),
				canvas,
				times,
			},
		})
	}
}
