package widgets

import (
	"fmt"
	"hash/fnv"
	"math"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

var ranges = []int{1, 24, 168}

func historyCard(t tessera.Tile) bool {
	switch t.Domain() {
	case "sensor", "binary_sensor", "switch", "input_boolean", "person", "number", "input_number":
		return true
	}
	return false
}

func clock(epoch int64, offset int, h24, weekday bool) string {
	at := time.Unix(epoch+int64(offset), 0).UTC()
	layout := "3:04 PM"
	if h24 {
		layout = "15:04"
	}
	if weekday {
		return at.Format("Mon")
	}
	return at.Format(layout)
}

func span(seconds int64) string {
	d := time.Duration(seconds) * time.Second
	switch {
	case d >= 24*time.Hour:
		return fmt.Sprintf("%d d %d h", int(d.Hours())/24, int(d.Hours())%24)
	case d >= time.Hour:
		return fmt.Sprintf("%d h %d min", int(d.Hours()), int(d.Minutes())%60)
	}
	return fmt.Sprintf("%d min", max(1, int(d.Minutes())))
}

func (v *View) graph(
	h tessera.History,
	width, height float32,
	ink gogui.Color,
	h24 bool,
) gogui.View {
	pal := v.o.UI.Palette()
	muted := gogui.RGB(pal.Muted.R, pal.Muted.G, pal.Muted.B)
	small := gogui.CurrentTheme().Cfg.TextStyleDef
	small.Size *= 0.8
	small.Color = muted
	return gogui.DrawCanvas(gogui.DrawCanvasCfg{
		ID:      "tessera-history",
		Width:   width,
		Height:  height,
		Sizing:  gogui.FixedFixed,
		Version: version(h, width, height, ink, h24),
		OnDraw: func(dc *gogui.DrawContext) {
			if h.Line {
				lineGraph(dc, h, width, height, ink, muted, small, h24)
				return
			}
			timeline(dc, h, width, height, muted, small, h24)
		},
	})
}

func version(h tessera.History, width, height float32, ink gogui.Color, h24 bool) uint64 {
	f := fnv.New64a()
	_, _ = fmt.Fprint(
		f,
		h.Entity,
		h.Hours,
		h.Start,
		h.End,
		h.Values,
		h.Runs,
		width,
		height,
		ink,
		h24,
	)
	return f.Sum64()
}

func ticks(
	dc *gogui.DrawContext,
	h tessera.History,
	left, width, y float32,
	small gogui.TextStyle,
	h24 bool,
) {
	for _, at := range h.Ticks {
		x := left + float32(at-h.Start)/float32(h.End-h.Start)*width
		text := clock(int64(at), h.Offset, h24, h.Hours == 168)
		tw := dc.TextWidth(text, small)
		dc.Text(min(max(x-tw/2, 0), left+width-tw), y, text, small)
	}
}

func lineGraph(
	dc *gogui.DrawContext,
	h tessera.History,
	width, height float32,
	ink, muted gogui.Color,
	small gogui.TextStyle,
	h24 bool,
) {
	line := dc.FontHeight(small)
	left := float32(0)
	for _, l := range h.Labels {
		left = max(left, dc.TextWidth(l.Text, small))
	}
	if left > 0 {
		left += line * 0.5
	}
	top := line * 0.5
	bottom := height - line*1.5
	plot := max(1, bottom-top)
	wide := max(1, width-left)
	y := func(v float64) float32 {
		return top + plot - float32((v-h.Bottom)/(h.Top-h.Bottom))*plot
	}
	for _, l := range h.Labels {
		at := y(l.Value)
		dc.Line(left, at, width, at, muted.WithOpacity(0.25), 1)
		dc.Text(0, at-line/2, l.Text, small)
	}
	n := len(h.Values)
	x := func(i int) float32 { return left + (float32(i)+0.5)/float32(max(1, n))*wide }
	stroke := max(2, style.Reach()*0.04)
	var run []float32
	flush := func() {
		switch len(run) {
		case 0:
		case 2:
			dc.FilledCircle(run[0], run[1], stroke, ink)
		default:
			for i := 0; i+3 < len(run); i += 2 {
				dc.FilledPolygon(
					[]float32{
						run[i],
						run[i+1],
						run[i+2],
						run[i+3],
						run[i+2],
						bottom,
						run[i],
						bottom,
					},
					ink.WithOpacity(0.18),
				)
			}
			dc.Polyline(run, ink, stroke)
		}
		run = run[:0]
	}
	for i, value := range h.Values {
		if math.IsNaN(value) {
			flush()
			continue
		}
		run = append(run, x(i), y(value))
	}
	flush()
	ticks(dc, h, left, wide, bottom+line*0.4, small, h24)
}

func timeline(
	dc *gogui.DrawContext,
	h tessera.History,
	width, height float32,
	muted gogui.Color,
	small gogui.TextStyle,
	h24 bool,
) {
	line := dc.FontHeight(small)
	bar := min(height*0.3, style.Reach())
	x := func(at int64) float32 {
		return min(max(float32(at-h.Start)/float32(h.End-h.Start)*width, 0), width)
	}
	dc.FilledRoundedRect(0, 0, width, bar, bar*0.2, muted.WithOpacity(0.15))
	for _, r := range h.Runs {
		c := muted.WithOpacity(0.35)
		if r.State >= 0 {
			rgb := h.States[r.State].Color
			c = gogui.RGB(uint8(rgb>>16), uint8(rgb>>8), uint8(rgb))
		}
		x0, x1 := x(r.Begin), x(r.End)
		dc.FilledRect(x0, 0, max(1, x1-x0), bar, c)
	}
	ticks(dc, h, 0, width, bar+line*0.3, small, h24)
	row := bar + line*1.8
	swatch := line * 0.7
	name := small
	name.Color = gogui.CurrentTheme().Cfg.TextStyleDef.Color
	for _, s := range h.States {
		if row+line > height {
			break
		}
		rgb := s.Color
		dc.FilledRoundedRect(0, row+(line-swatch)/2, swatch, swatch, swatch*0.25,
			gogui.RGB(uint8(rgb>>16), uint8(rgb>>8), uint8(rgb)))
		dc.Text(swatch*1.6, row, s.Label, name)
		took := span(s.Seconds)
		dc.Text(width-dc.TextWidth(took, small), row, took, small)
		row += line * 1.4
	}
}

func rangeName(hours int) string {
	switch hours {
	case 1:
		return say.T("tessera.range.hour")
	case 168:
		return say.T("tessera.range.week")
	}
	return say.T("tessera.range.day")
}

func sparkline(
	id string,
	values []float64,
	ink gogui.Color,
) func(width, height float32) gogui.View {
	low, high := math.Inf(1), math.Inf(-1)
	for _, v := range values {
		if !math.IsNaN(v) {
			low, high = min(low, v), max(high, v)
		}
	}
	return func(width, height float32) gogui.View {
		f := fnv.New64a()
		_, _ = fmt.Fprint(f, values, width, height, ink)
		return gogui.DrawCanvas(gogui.DrawCanvasCfg{
			ID:      id + "-spark",
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			Version: f.Sum64(),
			OnDraw: func(dc *gogui.DrawContext) {
				stroke := max(2, style.Reach()*0.04)
				plot := max(1, height-stroke*2)
				step := (width - stroke) / float32(max(1, len(values)-1))
				var run []float32
				flush := func() {
					if len(run) == 2 {
						run = append(run, width-stroke/2, run[1])
					}
					for i := 0; i+3 < len(run); i += 2 {
						dc.FilledPolygon(
							[]float32{
								run[i],
								run[i+1],
								run[i+2],
								run[i+3],
								run[i+2],
								height,
								run[i],
								height,
							},
							ink.WithOpacity(0.18),
						)
					}
					if len(run) >= 4 {
						dc.Polyline(run, ink, stroke)
					}
					run = run[:0]
				}
				for i, v := range values {
					if math.IsNaN(v) {
						flush()
						continue
					}
					level := float32(0.5)
					if high > low {
						level = float32((v - low) / (high - low))
					}
					run = append(run, stroke/2+float32(i)*step, stroke+plot-level*plot)
				}
				flush()
			},
		})
	}
}
