package widgets

import (
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/interaction"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/style"
)

type RowRenderer struct {
	Field   func(string, menu.Row) gogui.View
	Preview func(string, menu.Row) gogui.View
	Grip    func(gogui.View) gogui.View
}

func secondary() gogui.TextStyle {
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	st.Color = gogui.CurrentTheme().Cfg.ColorTextSecondary
	return st
}

func icon(glyph string, c gogui.Color) gogui.View {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return gogui.Label(glyph, st)
}

func (kit *Toolkit) Row(
	id, page string,
	r menu.Row,
	tap func(int),
	glyphs bool,
	render RowRenderer,
) gogui.View {
	label := []gogui.View{gogui.Label(r.Label, gogui.TextStyle{})}
	if r.Hint != "" {
		label = append(label, gogui.Label(r.Hint, secondary()))
	}
	head := gogui.Column(
		gogui.ContainerCfg{Sizing: gogui.FillFit, Padding: gogui.NoPadding, Content: label},
	)

	content := []gogui.View{head}
	if glyphs {
		var mark []gogui.View
		if r.Glyph != "" {
			mark = []gogui.View{icon(r.Glyph, gogui.CurrentTheme().Cfg.ColorTextSecondary)}
		}
		content = []gogui.View{gogui.Row(gogui.ContainerCfg{
			Width:   style.Reach(),
			Sizing:  gogui.FixedFit,
			Padding: gogui.NoPadding,
			HAlign:  gogui.HAlignCenter,
			Content: mark,
		}), head}
	}
	var onClick func(gogui.EventCtx)
	switch r.Kind {
	case menu.Toggle:
		content = append(content, kit.controls().Toggle(id+"-switch", r.On))
		onClick = interaction.Tapped(tap, 0)
	case menu.Slider:
		snap := r.Snap
		says := r.Value
		if says == "" {
			says = fmt.Sprintf("%d%%", r.Level)
		}
		content = append(content, gogui.Row(gogui.ContainerCfg{
			Padding: gogui.NewPadding(0, style.Reach()*0.4, 0, 0),
			Content: []gogui.View{gogui.Label(says, secondary())},
		}))
		level := gogui.SliderCfg{
			ID:     id + "-slider",
			Value:  float32(r.Level),
			Min:    0,
			Max:    100,
			Sizing: gogui.FillFit,
			Height: style.Reach(),
			OnChange: func(v float32, e gogui.EventCtx) {
				level := int(v + 0.5)
				if snap != nil {
					level = snap(level)
				}
				if tap != nil {
					tap(level)
				}
				e.Window.InvalidateLayout()
			},
		}
		content = append(content, render.Grip(kit.Slider(level)))
	case menu.Field:
		content = append(content, render.Field(page+"/"+id, r))
	case menu.Chevron:
		if r.Value != "" {
			content = append(content, gogui.Label(r.Value, secondary()))
		}
		content = append(
			content,
			icon(gogui.IconArrowRight, gogui.CurrentTheme().Cfg.ColorTextSecondary),
		)
		onClick = interaction.Tapped(tap, 0)
	default:
		if r.Value != "" {
			content = append(content, gogui.Label(r.Value, secondary()))
		}
		if r.Preview != nil {
			content = append(content, render.Preview(page, r))
		}
		if r.Chosen {
			content = append(content, icon(gogui.IconCheck, gogui.CurrentTheme().Cfg.ColorAccent))
		}
		onClick = interaction.Tapped(tap, 0)
	}
	cfg := gogui.ContainerCfg{
		ID:       id,
		Sizing:   gogui.FillFit,
		VAlign:   gogui.VAlignMiddle,
		Padding:  gogui.PaddingMedium,
		Spacing:  gogui.SpacingMedium,
		Radius:   gogui.RadiusMedium,
		Disabled: r.Dim,
		Content:  content,
	}
	kit.ListRow(&cfg, Chosen(r.Kind == menu.Toggle && r.Chosen))
	return kit.Pressable(gogui.Row, cfg, onClick)
}

func (kit *Toolkit) Tile(
	id, label string,
	selected bool,
	content []gogui.View,
	tap func(int),
) gogui.View {
	t := gogui.CurrentTheme().Cfg
	content = append(content, gogui.Label(label, gogui.TextStyle{}))
	cfg := gogui.ContainerCfg{
		ID:         id,
		HAlign:     gogui.HAlignCenter,
		Padding:    gogui.PaddingSmall,
		Spacing:    gogui.SpacingSmall,
		Radius:     gogui.RadiusMedium,
		SizeBorder: gogui.BorderPx(3),
		Clip:       true,
		Content:    content,
	}
	if selected {
		cfg.ColorBorder = t.ColorAccent
	}
	kit.controls().Tile(&cfg, Chosen(selected))
	return kit.Pressable(gogui.Column, cfg, interaction.Tapped(tap, 0))
}
