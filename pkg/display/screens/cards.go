package screens

import (
	gogui "github.com/go-gui-org/go-gui/gui"
	"golang.org/x/exp/shiny/materialdesign/icons"

	"github.com/ygelfand/libcountertop/pkg/assistant/timer"
	"github.com/ygelfand/libcountertop/pkg/assistant/turn"
	"github.com/ygelfand/libcountertop/pkg/display/assistant"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
)

const priorityAlert = 30

func (a *App) messageCard(w *gogui.Window) gogui.View {
	m, ok := a.o.Messages.Current()
	if !ok {
		return nil
	}
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	tint := color(m.Tone.Color(a.ui.Palette()))

	words := []gogui.View{}
	if m.Title != "" {
		st := t.TextStyleDisplay
		st.Color = tint
		words = append(
			words,
			gogui.Text(gogui.TextCfg{Text: m.Title, TextStyle: st, Mode: gogui.TextModeWrap}),
		)
	}
	words = append(
		words,
		gogui.Text(
			gogui.TextCfg{Text: m.Body, TextStyle: t.TextStyleTitle, Mode: gogui.TextModeWrap},
		),
	)

	card := gogui.Row(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:        "message",
		Width:     float32(vw) * 0.82,
		MinHeight: float32(vh) * 0.4,
		Sizing:    gogui.FixedFit,
		VAlign:    gogui.VAlignMiddle,
		Padding:   gogui.PaddingLarge,
		Spacing:   gogui.SpacingLarge,
		Clip:      true,
		Content: []gogui.View{
			gogui.Column(
				gogui.ContainerCfg{
					Width:   t.Cfg.RadiusMedium,
					Sizing:  gogui.FixedFill,
					Color:   tint,
					Radius:  gogui.RadiusSmall,
					Padding: gogui.NoPadding,
				},
			),
			gogui.Column(
				gogui.ContainerCfg{
					Sizing:  gogui.FillFit,
					Padding: gogui.NoPadding,
					Spacing: gogui.SpacingMedium,
					Content: words,
				},
			),
		},
	}))
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Color:   gogui.Black.WithOpacity(0.5),
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{card},
	})
}

const priorityNotice = 20

func (a *App) timerCard(w *gogui.Window) gogui.View {
	c := a.o.Timers.Now()
	if !c.Showing {
		return nil
	}
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	ink := t.Cfg.TextStyleDef.Color
	glyph := gogui.IconClock
	if c.Ringing {
		ink, glyph = t.Cfg.ColorError, gogui.IconBell
	}
	big := t.TextStyleTitle
	big.Color = ink

	line := []gogui.View{
		gogui.Label(glyph, iconStyle(ink)),
		gogui.Label(timer.Remaining(c.Left), big),
	}
	if c.Name != "" {
		line = append(line, gogui.Label(c.Name, secondary()))
	}
	content := []gogui.View{
		gogui.Row(
			gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				VAlign:  gogui.VAlignMiddle,
				Padding: gogui.NoPadding,
				Spacing: gogui.SpacingMedium,
				Content: line,
			},
		),
	}
	if !c.Ringing && c.Of > 0 {
		content = append(
			content,
			a.ui.Toolkit.ProgressBar(
				gogui.ProgressBarCfg{
					Percent: float32(min(max(c.Left.Seconds()/c.Of.Seconds(), 0), 1)),
					Sizing:  gogui.FillFit,
				},
			),
		)
	}

	var stop func(gogui.EventCtx)
	if c.Ringing {
		stop = func(gogui.EventCtx) { a.o.Timers.Stop() }
	}
	card := a.ui.Presses.Pressable(
		gogui.Column,
		a.ui.Toolkit.Panel(gogui.ContainerCfg{
			ID:      "timer",
			Width:   float32(vw) * 0.84,
			Sizing:  gogui.FixedFit,
			Padding: gogui.PaddingMedium,
			Spacing: gogui.SpacingSmall,
			Content: content,
		}),
		stop,
	)
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.PaddingMedium,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignBottom,
		Content: []gogui.View{card},
	})
}

func (a *App) assistantPanel(w *gogui.Window) gogui.View {
	show, ok := a.o.Assistant.Now()
	if !ok {
		return nil
	}
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	custom, full := a.o.Stage.Look(show)
	if !custom {
		a.o.Stage.Unstage()
	}
	if custom && full {
		a.o.Stage.Below(ui.Rect{W: vw, H: vh}, show)
		return a.captions(w, show)
	}
	deep := float32(vh) * 0.26
	pad := t.Cfg.PaddingLarge.Left
	bandW, bandH := float32(vw)-2*pad, deep*0.34

	ink := color(a.ui.Palette().Text)
	switch show.Phase {
	case turn.Listening:
		ink = t.Cfg.ColorAccent
	case turn.Thinking:
		ink = t.Cfg.ColorTextSecondary
	}
	band := gogui.DrawCanvas(gogui.DrawCanvasCfg{
		ID: "assistant-band", Width: bandW, Height: bandH,
		Version: uint64(show.At.UnixNano()),
		OnDraw: func(dc *gogui.DrawContext) {
			if show.Phase == turn.Thinking {
				dots := assistant.Dots(show)
				size := min(bandH/3, bandW/float32(len(dots)*4))
				gap := size * 2
				left := (bandW - (size*float32(len(dots)) + gap*float32(len(dots)-1))) / 2
				for i, lift := range dots {
					dc.FilledCircle(
						left+float32(i)*(size+gap)+size/2,
						bandH/2-size*float32(lift),
						size/2,
						ink,
					)
				}
				return
			}
			bars := assistant.Bars(show)
			n := float32(len(bars))
			gap := max(bandW/(n*4), 1)
			wide := max((bandW-gap*(n-1))/n, 1)
			left := (bandW - (wide*n + gap*(n-1))) / 2
			for i, part := range bars {
				high := max(bandH*float32(part), 1)
				dc.FilledRoundedRect(
					left+float32(i)*(wide+gap),
					(bandH-high)/2,
					wide,
					high,
					wide/2,
					ink,
				)
			}
		},
	})

	if custom {
		share := float32(0.3)
		if vw > vh {
			share = 0.4
		}
		stageH := min(bandW/1.6, float32(vh)*share)
		stage := gogui.Row(gogui.ContainerCfg{
			ID:      "assistant-stage",
			Width:   stageH * 1.6,
			Height:  stageH,
			Sizing:  gogui.FixedFixed,
			Padding: gogui.NoPadding,
			AmendLayout: func(e gogui.EventCtx) {
				s := e.Layout.Shape
				a.o.Stage.Above(
					ui.Rect{X: int(s.X), Y: int(s.Y), W: int(s.Width), H: int(s.Height)},
					show,
				)
			},
		})
		band = gogui.Row(
			gogui.ContainerCfg{
				Sizing:  gogui.FillFit,
				Padding: gogui.NoPadding,
				HAlign:  gogui.HAlignCenter,
				Content: []gogui.View{stage},
			},
		)
	}
	content := []gogui.View{band}
	if show.Said != "" {
		content = append(content, gogui.Label(show.Said, secondary()))
	}
	if show.Reply != "" {
		content = append(
			content,
			gogui.Text(
				gogui.TextCfg{
					Text:      assistant.Revealed(show.Reply, show.Reveal),
					TextStyle: t.TextStyleTitle,
					Mode:      gogui.TextModeWrap,
				},
			),
		)
	}
	return gogui.Column(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:           "assistant",
		Float:        true,
		FloatOffsetY: deep * (float32(show.Down) - 1),
		Width:        float32(vw),
		MaxHeight:    float32(vh) * 0.7,
		Sizing:       gogui.FixedFit,
		Padding:      gogui.PaddingLarge,
		Spacing:      gogui.SpacingSmall,
		Clip:         true,
		Content:      content,
	}))
}

func (a *App) captions(w *gogui.Window, show assistant.Showing) gogui.View {
	t := gogui.CurrentTheme()
	var lines []gogui.View
	if show.Said != "" {
		lines = append(
			lines,
			gogui.Text(
				gogui.TextCfg{Text: show.Said, TextStyle: secondary(), Mode: gogui.TextModeWrap},
			),
		)
	}
	if show.Reply != "" {
		lines = append(
			lines,
			gogui.Text(
				gogui.TextCfg{
					Text:      assistant.Revealed(show.Reply, show.Reveal),
					TextStyle: t.TextStyleTitle,
					Mode:      gogui.TextModeWrap,
				},
			),
		)
	}
	var content []gogui.View
	if len(lines) > 0 {
		content = append(content, gogui.Column(gogui.ContainerCfg{
			ID:      "assistant-captions",
			Sizing:  gogui.FillFit,
			Color:   color(a.ui.Palette().Background).WithOpacity(0.6),
			Radius:  gogui.RadiusLarge,
			Padding: gogui.PaddingLarge,
			Spacing: gogui.SpacingSmall,
			Content: lines,
		}))
	}
	return gogui.Column(gogui.ContainerCfg{
		ID:      "assistant",
		Sizing:  gogui.FillFill,
		Padding: gogui.PaddingLarge,
		VAlign:  gogui.VAlignBottom,
		Content: content,
	})
}

func (a *App) fullLook() bool {
	show, ok := a.o.Assistant.Now()
	if !ok {
		return false
	}
	custom, full := a.o.Stage.Look(show)
	return custom && full && a.o.Stage.Staged()
}

const priorityMarks = 100

var markRed = theme.Color{R: 0xe0, G: 0x3b, B: 0x2f}

func (a *App) privacyMarks(w *gogui.Window) gogui.View {
	m := a.o.Marks()
	if !m.Mic && !m.Camera {
		return nil
	}
	vw, vh := w.WindowSize()
	size := float32(min(vw, vh)) / 11
	glyph := int(size * 0.30)
	at := int(size/3) - glyph/2
	red := color(markRed)

	corner := func(right bool, icon ui.Icon, key string) []gogui.View {
		x := float32(0)
		gx := at
		if right {
			x = float32(vw) - size*1.3
			gx = vw - at - glyph
		}
		room := size * 1.3
		tri := gogui.DrawCanvas(gogui.DrawCanvasCfg{
			Width: room, Height: room,
			OnDraw: func(dc *gogui.DrawContext) {
				shape := func(reach, drop float32) []float32 {
					if right {
						return []float32{room, drop, room - reach, drop, room, reach + drop}
					}
					return []float32{0, drop, reach, drop, 0, reach + drop}
				}
				for k := 4; k >= 1; k-- {
					dc.FilledPolygon(
						shape(size+float32(k)*size*0.05, 2),
						gogui.Black.WithOpacity(0.12),
					)
				}
				dc.FilledPolygon(shape(size, 0), red)
			},
		})
		mark := widgets.Picture(
			a.painted(
				"privacy/"+key,
				glyph,
				glyph,
				markRed,
				func(s ui.Surface, r ui.Rect, _ theme.Theme) {
					ui.DrawIcon(s, icon, r, theme.Color{R: 0xff, G: 0xff, B: 0xff}, markRed)
				},
			),
			glyph,
			glyph,
		)
		return []gogui.View{
			widgets.Placed(ui.Rect{X: int(x), Y: 0, W: int(size * 1.3), H: int(size * 1.3)}, tri),
			widgets.Placed(ui.Rect{X: gx, Y: at, W: glyph, H: glyph}, mark),
		}
	}

	var layers []gogui.View
	if m.Mic {
		layers = append(layers, corner(false, icons.AVMicOff, "mic")...)
	}
	if m.Camera {
		layers = append(layers, corner(true, icons.AVVideocamOff, "camera")...)
	}
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Content: layers,
	})
}
