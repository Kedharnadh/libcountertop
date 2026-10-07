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
		Height:    float32(vh) * 0.45,
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
			Padding: gogui.PaddingSmall,
			Radius:  gogui.RadiusMedium,
			Content: []gogui.View{gogui.Label(stream.Label, secondary())},
		}, func(gogui.EventCtx) { options.Expand() }),
	}
	if open {
		for _, s := range options.Choices {
			if s.Key == stream.Key {
				continue
			}
			pick := s
			capsule = append(capsule, kit.Pressable(gogui.Row, gogui.ContainerCfg{
				ID:      "volume-pick-" + s.Key,
				Padding: gogui.PaddingSmall,
				Radius:  gogui.RadiusMedium,
				Content: []gogui.View{gogui.Label(s.Label, gogui.TextStyle{})},
			}, func(gogui.EventCtx) { options.Pick(pick.Key) }))
		}
	}

	card := gogui.Column(kit.Panel(gogui.ContainerCfg{
		ID:      "volume",
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingMedium,
		HAlign:  gogui.HAlignCenter,
		OnClick: func(e gogui.EventCtx) {
			options.Linger()
			e.Consume()
		},
		Content: capsule,
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
