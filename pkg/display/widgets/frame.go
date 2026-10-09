package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type FrameOptions struct {
	Title   string
	Fixed   bool
	Build   func(*gogui.Window) gogui.View
	Back    func()
	Gesture func(gogui.EventCtx)
	Room    []gogui.View
}

func Placed(at ui.Rect, v gogui.View) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Float:        true,
		FloatOffsetX: float32(at.X),
		FloatOffsetY: float32(at.Y),
		Width:        float32(at.W),
		Height:       float32(at.H),
		Sizing:       gogui.FixedFixed,
		Padding:      gogui.NoPadding,
		Content:      []gogui.View{v},
	})
}

func HAligned(a schema.Align) gogui.HorizontalAlign {
	switch a {
	case schema.AlignLeft:
		return gogui.HAlignLeft
	case schema.AlignRight:
		return gogui.HAlignRight
	}
	return gogui.HAlignCenter
}

func Aligned(p schema.Placement, cfg gogui.ContainerCfg) gogui.ContainerCfg {
	cfg.HAlign, cfg.VAlign = HAligned(p.Align), gogui.VAlignMiddle
	switch p.Position {
	case schema.PositionTop:
		cfg.VAlign = gogui.VAlignTop
	case schema.PositionBottom:
		cfg.VAlign = gogui.VAlignBottom
	}
	return cfg
}

func Floating(z int, v gogui.View) gogui.View {
	return gogui.Column(
		gogui.ContainerCfg{
			Float:       true,
			FloatZIndex: z + 1,
			Sizing:      gogui.FillFill,
			Padding:     gogui.NoPadding,
			Content:     []gogui.View{v},
		},
	)
}

func (kit *Toolkit) PageFrame(w *gogui.Window, p FrameOptions) gogui.View {
	t := gogui.CurrentTheme()
	back := t.TextStyleIconMedium
	back.Color = t.TextStyleDisplay.Color
	back.Size = t.TextStyleDisplay.Size * 0.6
	return gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		Padding: gogui.PaddingLarge,
		Spacing: gogui.SpacingMedium,
		Content: []gogui.View{
			gogui.Row(gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				VAlign:  gogui.VAlignMiddle,
				Spacing: gogui.SpacingMedium,
				Content: []gogui.View{
					kit.KeyButton(
						"back",
						gogui.IconArrowLeft,
						back,
						style.Rest,
						func(e gogui.EventCtx) {
							p.Back()
							e.Window.InvalidateLayout()
						},
					),
					gogui.Label(p.Title, t.TextStyleDisplay),
				},
			}),
			gogui.Column(
				gogui.ContainerCfg{
					ID:         "page",
					Sizing:     gogui.FillFill,
					Scrollable: !p.Fixed,
					OnGesture:  p.Gesture,
					Content:    append([]gogui.View{p.Build(w)}, p.Room...),
				},
			),
		},
	})
}

func (e *Editor) Room(w *gogui.Window) []gogui.View {
	_, vh := w.WindowSize()
	if !e.Typing(w) || e.Top <= 0 {
		return nil
	}
	return []gogui.View{
		gogui.Column(
			gogui.ContainerCfg{
				Height:  float32(vh) - e.Top,
				Sizing:  gogui.FillFixed,
				Padding: gogui.NoPadding,
			},
		),
	}
}
