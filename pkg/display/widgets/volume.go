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
	Set                     func(int)
	Pick                    func(string)
	Linger, Expand, Dismiss func()
	Grip                    func(gogui.View) gogui.View
}

func (kit *Toolkit) VolumeCard(w *gogui.Window, options VolumeOptions) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	stream, open := options.Current, options.Open
	level := options.Current.Level
	outerPadding := t.PaddingLarge.Top + t.PaddingLarge.Bottom
	cardPadding := t.PaddingMedium.Top + t.PaddingMedium.Bottom
	inside := max(0, float32(vh)-outerPadding-cardPadding)
	rowHeight := style.Reach()
	titleHeight := t.TextStyleTitle.Size * 1.5
	fixed := titleHeight + rowHeight + 2*t.SpacingMedium
	meterHeight := max(0, min(float32(vh)*0.45, inside-fixed))
	bodyHeight := min(inside, fixed+meterHeight)

	meter := gogui.SliderCfg{
		ID:        "volume-card",
		Vertical:  true,
		Value:     float32(100 - level),
		Color:     t.Cfg.ColorAccent,
		ColorLeft: t.Cfg.ColorInterior,
		Max:       100,
		Width:     style.Reach(),
		Size:      style.Reach() * 0.3,
		ThumbSize: style.Reach() * 0.65,
		Height:    meterHeight,
		OnChange: func(v float32, e gogui.EventCtx) {
			options.Set(100 - int(v+0.5))
			options.Linger()
			e.Window.InvalidateLayout()
		},
	}
	capsule := []gogui.View{
		gogui.Label(fmt.Sprintf("%d", level), t.TextStyleTitle),
		options.Grip(kit.Slider(meter)),
		kit.Pressable(gogui.Row, gogui.ContainerCfg{
			ID:      "volume-stream",
			Height:  rowHeight,
			Sizing:  gogui.FitFixed,
			Padding: gogui.PaddingSmall,
			Radius:  gogui.RadiusMedium,
			Content: []gogui.View{gogui.Label(stream.Label, secondary())},
		}, func(gogui.EventCtx) { options.Expand() }),
	}
	content := capsule
	if open {
		var choices []gogui.View
		for _, s := range options.Choices {
			if s.Key == stream.Key {
				continue
			}
			pick := s
			choices = append(choices, kit.Pressable(gogui.Row, gogui.ContainerCfg{
				ID:      "volume-pick-" + s.Key,
				Height:  rowHeight,
				Sizing:  gogui.FitFixed,
				Padding: gogui.PaddingSmall,
				Radius:  gogui.RadiusMedium,
				Content: []gogui.View{gogui.Label(s.Label, gogui.TextStyle{})},
			}, func(gogui.EventCtx) { options.Pick(pick.Key) }))
		}
		content = []gogui.View{gogui.Row(gogui.ContainerCfg{
			Padding: gogui.NoPadding, Spacing: gogui.SpacingMedium, VAlign: gogui.VAlignMiddle,
			Content: []gogui.View{
				gogui.Column(gogui.ContainerCfg{Padding: gogui.NoPadding, Spacing: gogui.SpacingMedium, HAlign: gogui.HAlignCenter, Content: capsule}),
				gogui.Column(gogui.ContainerCfg{ID: "volume-choices", Height: bodyHeight, Sizing: gogui.FitFixed, Scrollable: true, Padding: gogui.NoPadding, Spacing: gogui.SpacingSmall, Content: choices}),
			},
		})}
	}

	card := gogui.Column(kit.Panel(gogui.ContainerCfg{
		ID:         "volume",
		Height:     bodyHeight + cardPadding,
		Sizing:     gogui.FitFixed,
		Scrollable: true,
		Padding:    gogui.PaddingMedium,
		Spacing:    gogui.SpacingMedium,
		HAlign:     gogui.HAlignCenter,
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
