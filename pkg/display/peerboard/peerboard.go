package peerboard

import (
	"embed"
	"fmt"
	"math"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
)

//go:embed devices/*.svg
var devices embed.FS

const Aspect = 190.0 / 320.0

func Picture(codename string) (key, svg string) {
	if b, err := devices.ReadFile("devices/" + codename + ".svg"); err == nil {
		return "device-" + codename, string(b)
	}
	b, _ := devices.ReadFile("devices/default.svg")
	return "device-default", string(b)
}

type Action struct {
	Name string
	Icon string
	Do   func(gogui.EventCtx)
}

type Board struct {
	UI      *widgets.UI
	ID      string
	Empty   string
	Actions func(peer.Peer) []Action
}

func (b Board) View(at ui.Rect, peers []peer.Peer, pal theme.Theme) gogui.View {
	gap := style.Reach() * 0.25
	cols := max(3, int(float32(at.W)/(style.Reach()*4)))
	side := cell(float32(at.W)-gap, gap, cols)

	var views []gogui.View
	if len(peers) == 0 {
		views = append(views, gogui.Label(b.Empty, widgets.Secondary()))
	}
	stacks := make([][]gogui.View, cols)
	for i, p := range peers {
		stacks[i%cols] = append(stacks[i%cols], b.tile(fmt.Sprintf("%s-%d", b.ID, i), side, p, pal))
	}
	if len(peers) > 0 {
		columns := make([]gogui.View, 0, cols)
		for _, stack := range stacks {
			columns = append(columns, gogui.Column(gogui.ContainerCfg{
				Width:   side,
				Sizing:  gogui.FixedFit,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingPx(gap),
				Content: stack,
			}))
		}
		views = append(views, gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingPx(gap),
			Content: columns,
		}))
	}
	return widgets.Placed(at, gogui.Column(gogui.ContainerCfg{
		ID:         b.ID,
		Sizing:     gogui.FillFill,
		Padding:    gogui.NoPadding,
		Spacing:    gogui.SpacingPx(gap),
		Scrollable: true,
		OnGesture:  b.UI.HoldStill,
		Content:    views,
	}))
}

func (b Board) tile(id string, side float32, p peer.Peer, pal theme.Theme) gogui.View {
	t := gogui.CurrentTheme().Cfg
	inner := side - 2*t.PaddingMedium.Left
	key, svg := Picture(p.Board)

	glyph := gogui.CurrentTheme().TextStyleIconMedium
	glyph.Color = t.TextStyleDef.Color
	glyph.Size = style.Reach() * 0.4
	var keys []gogui.View
	for _, a := range b.Actions(p) {
		keys = append(keys, b.UI.Toolkit.WideKey(
			id+"-"+a.Name,
			a.Icon,
			glyph,
			style.Reach()*0.9,
			style.Partial,
			a.Do,
		))
	}
	tall := min(style.Reach()*1.4, inner*Aspect)

	named := []gogui.View{gogui.Label(p.Name, t.TextStyleDef)}
	if p.Standing == peer.Foreign {
		named = append(named, gogui.Label(say.T("peer.foreign"), widgets.Secondary()))
	}
	cfg := gogui.ContainerCfg{
		ID:      id,
		Width:   side,
		Sizing:  gogui.FixedFit,
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingSmall,
		Radius:  gogui.RadiusLarge,
		Color:   widgets.Color(pal.Surface),
		Content: []gogui.View{
			gogui.Column(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				Content: named,
			}),
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{
					gogui.Svg(gogui.SvgCfg{
						ID:      id + "-" + key,
						SvgData: svg,
						Width:   tall / Aspect,
						Height:  tall,
					}),
				},
			}),
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingSmall,
				Content: keys,
			}),
		},
	}
	b.UI.Kit().Tile(&cfg, style.Rest)
	return gogui.Column(cfg)
}

func cell(width, gap float32, n int) float32 {
	return float32(math.Floor(float64((width - gap*float32(n-1)) / float32(n))))
}
