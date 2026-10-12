package widgets

import (
	"cmp"
	"fmt"
	"hash/fnv"
	"math"
	"slices"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	kit "github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const energyEntity = "screen.energy"

type energyNode struct {
	glyph  rune
	name   string
	charge string
	watts  float64
	ink    gogui.Color
	x, y   float32
}

type energyRow struct {
	name  string
	watts float64
	rest  bool
}

func batteryGlyph(soc float64, known bool) rune {
	if !known {
		return 0xF0079
	}
	step := int(soc+5) / 10
	switch {
	case step <= 0:
		return 0xF008E
	case step >= 10:
		return 0xF0079
	}
	return 0xF007A + rune(step-1)
}

func flowWidth(watts float64, base float32) float32 {
	switch {
	case watts >= 2000*2/3.0:
		return base * 3
	case watts >= 2000/3.0:
		return base * 2
	}
	return base
}

func energyRows(devices []map[string]any, other, home float64, places int) []energyRow {
	if places <= 0 {
		return nil
	}
	measured := other
	all := make([]energyRow, 0, len(devices))
	for _, d := range devices {
		name, _ := d["n"].(string)
		watts, _ := d["w"].(float64)
		all = append(all, energyRow{name: name, watts: watts})
		measured += watts
	}
	untracked := home - measured
	tracks := measured > 0 && untracked > 1
	room := places
	if tracks && places >= 2 {
		room--
	}
	var rows []energyRow
	if len(all)+btoi(other > 0) <= room || len(all) == 0 {
		rows = append(rows, all...)
		if other > 0 {
			rows = append(
				rows,
				energyRow{name: say.T("tessera.energy.other"), watts: other, rest: true},
			)
		}
	} else {
		order := make([]int, len(all))
		for i := range order {
			order[i] = i
		}
		slices.SortStableFunc(
			order,
			func(a, b int) int { return cmp.Compare(all[b].watts, all[a].watts) },
		)
		keep := order[:min(max(room-1, 1), len(order))]
		slices.Sort(keep)
		for _, i := range keep {
			rows = append(rows, all[i])
		}
		if room > 1 {
			rest := energyRow{name: say.T("tessera.energy.other"), watts: other, rest: true}
			for _, i := range order[len(keep):] {
				rest.watts += all[i].watts
			}
			rows = append(rows, rest)
		}
	}
	if tracks && len(rows) < places {
		rows = append(
			rows,
			energyRow{name: say.T("tessera.energy.untracked"), watts: untracked, rest: true},
		)
	}
	return rows
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func power(w float64) string {
	if w >= 1000 {
		return fmt.Sprintf("%.1f kW", w/1000)
	}
	return fmt.Sprintf("%.0f W", w)
}

func ink(c theme.Color) gogui.Color { return gogui.RGB(c.R, c.G, c.B) }

func at(list []float64, i int) float64 {
	if i < len(list) && list[i] > 0 {
		return list[i]
	}
	return 0
}

func (v *View) energyBody(
	w *gogui.Window,
	id string,
	t tessera.Tile,
	f frame,
) func(width, height float32) gogui.View {
	has, _ := t.ExtraNumber("h")
	mask := int(has)
	p, flows := t.ExtraNumbers("p"), t.ExtraNumbers("f")
	pal := f.pal
	home := t.ExtraText("n")
	if home == "" {
		home = say.T("tessera.energy.home")
	}
	solar := &energyNode{
		glyph: 0xF0A72,
		name:  say.T("tessera.energy.solar"),
		watts: at(p, 0),
		ink:   ink(pal.Roles.Warm),
	}
	grid := &energyNode{
		glyph: 0xF07DA,
		name:  say.T("tessera.energy.grid"),
		watts: at(p, 1) - at(p, 2),
		ink:   ink(pal.Roles.Cool),
	}
	soc, charged := t.ExtraNumber("c")
	battery := &energyNode{
		glyph: batteryGlyph(soc, charged),
		name:  say.T("tessera.energy.battery"),
		watts: at(p, 3) - at(p, 4),
		ink:   ink(pal.Success),
	}
	if charged {
		battery.charge = fmt.Sprintf("%.0f%%", soc)
	}
	lines := t.Flow == "lines"
	house := &energyNode{glyph: 0xF02DC, name: home, watts: at(p, 5), ink: ink(pal.Text)}
	type link struct {
		a, b  *energyNode
		watts float64
	}
	var links []link
	add := func(a, b *energyNode, watts float64, present bool) {
		if present {
			links = append(links, link{a, b, watts})
		}
	}
	add(solar, house, at(flows, 0), mask&1 != 0)
	add(solar, grid, at(flows, 1), mask&3 == 3)
	add(solar, battery, at(flows, 2), mask&5 == 5)
	add(grid, house, at(flows, 3), mask&2 != 0)
	add(grid, battery, at(flows, 4)+at(flows, 6), mask&6 == 6)
	add(battery, house, at(flows, 5), mask&4 != 0)
	nodes := []*energyNode{house}
	if mask&1 != 0 {
		nodes = append(nodes, solar)
	}
	if mask&2 != 0 {
		nodes = append(nodes, grid)
	}
	if mask&4 != 0 {
		nodes = append(nodes, battery)
	}
	muted := ink(pal.Muted)
	devices := t.ExtraList("d")
	other, _ := t.ExtraNumber("o")
	return func(width, height float32) gogui.View {
		reach := style.Reach()
		text := gogui.CurrentTheme().Cfg.TextStyleDef
		small := secondary()
		list := (len(devices) > 0 || other > 0) && width >= height*1.6
		plot := width
		if list {
			plot = width * 0.6
		}
		r := min(plot, height) * 0.14
		cx, cy := plot/2, height/2
		solar.x, solar.y = cx, r+2
		battery.x, battery.y = cx, height-r-2
		grid.x, grid.y = r+2, cy
		house.x, house.y = plot-r-2, cy
		hv := fnv.New64a()
		_, _ = fmt.Fprint(
			hv,
			width,
			height,
			p,
			flows,
			mask,
			pal.Name,
			pal.Dark,
			lines,
			battery.charge,
		)
		canvas := gogui.DrawCanvas(gogui.DrawCanvasCfg{
			ID:      id + "-energy",
			Width:   plot,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			Version: hv.Sum64(),
			OnDraw: func(dc *gogui.DrawContext) {
				for _, l := range links {
					dx, dy := l.b.x-l.a.x, l.b.y-l.a.y
					d := float32(math.Hypot(float64(dx), float64(dy)))
					if d <= 2*r {
						continue
					}
					ux, uy := dx/d, dy/d
					c := muted.WithOpacity(0.35)
					width := max(1, reach*0.03)
					if l.watts > 0 {
						c = l.a.ink
						width = max(2, reach*0.06)
						if lines {
							width = flowWidth(l.watts, max(2, reach*0.04))
						}
					}
					dc.Line(l.a.x+ux*r, l.a.y+uy*r, l.b.x-ux*r, l.b.y-uy*r, c, width)
					if lines && l.watts > 0 {
						mx, my := (l.a.x+l.b.x)/2, (l.a.y+l.b.y)/2
						s := max(width*1.6, reach*0.1)
						dc.FilledPolygon([]float32{
							mx + ux*s, my + uy*s,
							mx - ux*s*0.6 - uy*s*0.8, my - uy*s*0.6 + ux*s*0.8,
							mx - ux*s*0.6 + uy*s*0.8, my - uy*s*0.6 - ux*s*0.8,
						}, c)
					}
				}
				glyph := gogui.CurrentTheme().TextStyleIconMedium
				glyph.Family = icons.Family
				glyph.Size = r * 0.6
				value := text
				value.Size = min(text.Size, r*0.42)
				for _, n := range nodes {
					glyph.Color = n.ink
					dc.FilledCircle(n.x, n.y, r, n.ink.WithOpacity(0.12))
					dc.Circle(n.x, n.y, r, n.ink, max(2, reach*0.05))
					g := icons.Glyph(n.glyph)
					gw := dc.TextWidth(g, glyph)
					if n.charge == "" {
						dc.Text(n.x-gw/2, n.y-r*0.72, g, glyph)
					} else {
						label := value
						label.Size = value.Size * 0.8
						label.Color = n.ink
						cw := dc.TextWidth(n.charge, label)
						left := n.x - (gw+cw)/2
						dc.Text(left, n.y-r*0.72, g, glyph)
						dc.Text(left+gw, n.y-r*0.6, n.charge, label)
					}
					s := power(math.Abs(n.watts))
					dc.Text(n.x-dc.TextWidth(s, value)/2, n.y+r*0.05, s, value)
				}
			},
		})
		content := []gogui.View{canvas}
		if list {
			rows := []gogui.View{}
			room := width - plot - reach*0.3
			places := int(height / (text.Size * 1.6))
			for _, d := range energyRows(devices, other, at(p, 5), places) {
				name, watts := d.name, d.watts
				text, small := text, small
				if d.rest {
					text.Color, small.Color = muted, muted
				}
				rows = append(rows, gogui.Row(gogui.ContainerCfg{
					Width:   room,
					Sizing:  gogui.FixedFit,
					Padding: gogui.NoPadding,
					Content: []gogui.View{
						gogui.Label(kit.Ellipsize(w, name, text, room*0.62), text),
						gogui.Row(
							gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding},
						),
						gogui.Label(power(watts), small),
					},
				}))
			}
			content = append(content, gogui.Column(gogui.ContainerCfg{
				Width:   room,
				Sizing:  gogui.FixedFit,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingPx(text.Size * 0.4),
				Content: rows,
			}))
		}
		return gogui.Row(gogui.ContainerCfg{
			Width:   width,
			Height:  height,
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(reach * 0.3),
			VAlign:  gogui.VAlignMiddle,
			Content: content,
		})
	}
}
