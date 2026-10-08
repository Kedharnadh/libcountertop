package widgets

import (
	"fmt"

	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/style"
)

type VolumeChoice struct {
	Key, Label string
	Level      int
}
type VolumeOptions struct {
	Current                 VolumeChoice
	Open, Left              bool
	Choices                 []VolumeChoice
	Set                     func(string, int)
	Linger, Expand, Dismiss func()
	Grip                    func(gogui.View) gogui.View
}

func (kit *Toolkit) VolumeCard(w *gogui.Window, options VolumeOptions) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	stream, open := options.Current, options.Open
	outerPadding := t.PaddingLarge.Top + t.PaddingLarge.Bottom
	cardPadding := t.PaddingMedium.Top + t.PaddingMedium.Bottom
	inside := max(0, float32(vh)-outerPadding-cardPadding)
	rowHeight := style.Reach()
	titleHeight := t.TextStyleTitle.Size * 1.5
	fixed := titleHeight + rowHeight + 3*t.SpacingMedium + 1
	meterHeight := max(0, inside-fixed)
	bodyHeight := inside

	bar := func(choice VolumeChoice, current bool) gogui.View {
		meter := gogui.SliderCfg{
			ID:        "volume-slider-" + choice.Key,
			Vertical:  true,
			Value:     float32(100 - choice.Level),
			Color:     t.Cfg.ColorAccent,
			ColorLeft: t.Cfg.ColorInterior,
			Max:       100,
			Width:     style.Reach(),
			Size:      style.Reach() * 0.3,
			ThumbSize: style.Reach() * 0.65,
			Height:    meterHeight,
			Sizing:    gogui.FixedFixed,
			OnChange: func(v float32, e gogui.EventCtx) {
				options.Set(choice.Key, 100-int(v+0.5))
				options.Linger()
				e.Window.InvalidateLayout()
			},
		}
		arrow := gogui.IconArrowLeft
		if options.Left {
			arrow = gogui.IconArrowRight
		}
		if open {
			if options.Left {
				arrow = gogui.IconArrowLeft
			} else {
				arrow = gogui.IconArrowRight
			}
		}
		icon := t.TextStyleIconMedium
		icon.Color = secondary().Color
		label := []gogui.View{gogui.Label(choice.Label, secondary())}
		if current {
			label = append([]gogui.View{gogui.Label(arrow, icon)}, label...)
		}
		capsule := []gogui.View{
			gogui.Label(fmt.Sprintf("%d", choice.Level), t.TextStyleTitle),
			options.Grip(kit.Slider(meter)),
			gogui.Separator(gogui.SeparatorCfg{Size: 1}),
			gogui.Row(gogui.ContainerCfg{
				ID:      "volume-label-" + choice.Key,
				Height:  rowHeight,
				Sizing:  gogui.FitFixed,
				Padding: gogui.PaddingSmall,
				Radius:  gogui.RadiusMedium,
				VAlign:  gogui.VAlignMiddle,
				Spacing: gogui.SpacingSmall,
				Content: label,
				OnClick: func(e gogui.EventCtx) {
					if current {
						options.Expand()
					}
					options.Linger()
					e.Window.InvalidateLayout()
					e.Consume()
				},
			}),
		}
		return gogui.Column(gogui.ContainerCfg{ID: "volume-bar-" + choice.Key, Height: bodyHeight, Sizing: gogui.FitFixed, Padding: gogui.NoPadding, Spacing: gogui.SpacingMedium, HAlign: gogui.HAlignCenter, Content: capsule})
	}
	control := bar(stream, true)
	columns := []gogui.View{control}
	if open {
		columns = nil
		for _, s := range options.Choices {
			if s.Key == stream.Key {
				continue
			}
			columns = append(columns, bar(s, false))
		}
		if options.Left {
			columns = append([]gogui.View{control}, columns...)
		} else {
			columns = append(columns, control)
		}
	}
	content := []gogui.View{gogui.Row(gogui.ContainerCfg{
		Padding: gogui.NoPadding, Spacing: gogui.SpacingMedium, VAlign: gogui.VAlignMiddle,
		Content: columns,
	})}

	card := gogui.Column(kit.Panel(gogui.ContainerCfg{
		ID:      "volume",
		Height:  bodyHeight + cardPadding,
		Sizing:  gogui.FitFixed,
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingMedium,
		HAlign:  gogui.HAlignCenter,
		OnClick: func(e gogui.EventCtx) {
			options.Linger()
			e.Consume()
		},
		Content: content,
	}))

	side := gogui.HAlignRight
	if options.Left {
		side = gogui.HAlignLeft
	}
	return kit.Pressable(gogui.Column, gogui.ContainerCfg{
		ID:      "volume-away",
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.PaddingLarge,
		HAlign:  side,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{card},
	}, func(gogui.EventCtx) { options.Dismiss() })
}
