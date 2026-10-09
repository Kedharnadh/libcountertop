package widgets

import (
	"fmt"
	"hash/fnv"
	"math"

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
	glyph rune
	name  string
	watts float64
	ink   gogui.Color
	x, y  float32
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
	battery := &energyNode{
		glyph: 0xF0079,
		name:  say.T("tessera.energy.battery"),
		watts: at(p, 3) - at(p, 4),
		ink:   ink(pal.Success),
	}
	if soc, ok := t.ExtraNumber("c"); ok {
		battery.name += fmt.Sprintf(" %.0f%%", soc)
	}
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
	return func(width, height float32) gogui.View {
		reach := style.Reach()
		text := gogui.CurrentTheme().Cfg.TextStyleDef
		small := secondary()
		list := len(devices) > 0 && width >= height*1.6
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
		_, _ = fmt.Fprint(hv, width, height, p, flows, mask, pal.Name, pal.Dark)
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
					}
					dc.Line(l.a.x+ux*r, l.a.y+uy*r, l.b.x-ux*r, l.b.y-uy*r, c, width)
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
					dc.Text(n.x-dc.TextWidth(g, glyph)/2, n.y-r*0.72, g, glyph)
					s := power(math.Abs(n.watts))
					dc.Text(n.x-dc.TextWidth(s, value)/2, n.y+r*0.05, s, value)
				}
			},
		})
		content := []gogui.View{canvas}
		if list {
			rows := []gogui.View{}
			room := width - plot - reach*0.3
			for i, d := range devices {
				if float32(i+1)*text.Size*1.6 > height {
					break
				}
				name, _ := d["n"].(string)
				watts, _ := d["w"].(float64)
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
