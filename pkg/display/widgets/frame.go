package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

// FrameOptions binds navigation and input handling to a page frame.
type FrameOptions struct {
	Title   string
	Fixed   bool
	Build   func(*gogui.Window) gogui.View
	Back    func()
	Gesture func(gogui.EventCtx)
	Room    []gogui.View
}

// Placed positions a view in a fixed rectangle.
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

// Floating adds a view above the base layer.
func Floating(z int, v gogui.View) gogui.View {
	return gogui.Column(gogui.ContainerCfg{Float: true, FloatZIndex: z + 1, Sizing: gogui.FillFill, Padding: gogui.NoPadding, Content: []gogui.View{v}})
}

// PageFrame renders a page title, back button, and scrollable body.
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
					kit.KeyButton("back", gogui.IconArrowLeft, back, style.Rest, func(e gogui.EventCtx) {
						p.Back()
						e.Window.InvalidateLayout()
					}),
					gogui.Label(p.Title, t.TextStyleDisplay),
				},
			}),
			gogui.Column(gogui.ContainerCfg{ID: "page", Sizing: gogui.FillFill, Scrollable: !p.Fixed, OnGesture: p.Gesture, Content: append([]gogui.View{p.Build(w)}, p.Room...)}),
		},
	})
}

// Room leaves scrolling space for the active keyboard.
func (e *Editor) Room(w *gogui.Window) []gogui.View {
	_, vh := w.WindowSize()
	if !e.Typing(w) || e.Top <= 0 {
		return nil
	}
	return []gogui.View{gogui.Column(gogui.ContainerCfg{Height: float32(vh) - e.Top, Sizing: gogui.FillFixed, Padding: gogui.NoPadding})}
}
