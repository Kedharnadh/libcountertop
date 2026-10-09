package screens

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/go-gui-org/go-glyph"
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/assistant/timer"
	face "github.com/ygelfand/libcountertop/pkg/display/clockface"
	"github.com/ygelfand/libcountertop/pkg/display/clockkind"
	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/idleview"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	dialShare    = 0.62
	squarish     = 1.15
	sweepFastest = time.Second / 24
	sweepSlowest = time.Second
	sweepStep    = 2
)

func (a *App) timer(w *gogui.Window) gogui.View {
	c := a.o.Timers.Now()
	if !c.Showing {
		return nil
	}
	switch a.o.Settings().Screen.Timer {
	case schema.TimerBanner:
		return a.timerBanner(w, c)
	case schema.TimerDial:
		return a.timerDial(w, c)
	case schema.TimerFull:
		return a.timerFull(w, c)
	case schema.TimerFace:
		return a.timerFace(w, c)
	case schema.TimerCorner:
		return a.timerCorner(w, c)
	}
	return a.timerBar(w, c)
}

func (a *App) timerCovers() bool {
	return a.o.Timers.Now().Showing && a.o.Settings().Screen.Timer == schema.TimerFull
}

func (a *App) timerBar(w *gogui.Window, c timer.Card) gogui.View {
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
				gogui.ProgressBarCfg{Percent: float32(c.Share()), Sizing: gogui.FillFit},
			),
		)
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
		a.timerTap(c),
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

func (a *App) timerBanner(w *gogui.Window, c timer.Card) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	tint := color(a.timerTint(w, c, pal.Background))
	margin := reach() * 0.5
	label, name := timerReads(c)

	head := t.TextStyleTitle
	head.Color = tint
	words := []gogui.View{gogui.Label(label, head)}
	if name != "" {
		words = append(words, gogui.Label(name, secondary()))
	}
	icon := iconStyle(tint)
	icon.Size = t.TextStyleTitle.Size * 1.6
	line := []gogui.View{
		edge(tint, t.Cfg.RadiusMedium),
		gogui.Label(timerGlyph(c), icon),
		gogui.Column(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingSmall,
			Clip:    true,
			Content: words,
		}),
	}
	if c.Ringing {
		line = append(line, a.stopKey(w, t.Cfg.TextStyleDef, reach(), 0, c))
	}
	content := []gogui.View{gogui.Row(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		Content: line,
	})}
	if !c.Ringing && c.Of > 0 {
		content = append(content, a.ui.Toolkit.ProgressBar(gogui.ProgressBarCfg{
			Percent:  float32(c.Share()),
			Height:   max(t.Cfg.RadiusMedium*0.6, 2),
			Sizing:   gogui.FillFixed,
			ColorBar: color(pal.Accent),
		}))
	}
	banner := a.ui.Presses.Pressable(gogui.Column, blocking(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:      "timer",
		Width:   float32(vw) - 2*margin,
		Sizing:  gogui.FixedFit,
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingSmall,
		Clip:    true,
		Content: content,
	})), a.timerTap(c))
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NewPadding(margin, margin, margin, margin),
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignTop,
		Content: []gogui.View{banner},
	})
}

func (a *App) timerDial(w *gogui.Window, c timer.Card) gogui.View {
	vw, vh := w.WindowSize()
	scrim := gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Color:   gogui.Black.WithOpacity(0.5),
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{a.dial(w, c, float32(min(vw, vh))*dialShare)},
	}
	if c.Ringing {
		scrim = blocking(scrim)
	}
	return gogui.Column(scrim)
}

func (a *App) dial(w *gogui.Window, c timer.Card, side float32) gogui.View {
	pal := a.ui.Palette()
	tint := a.timerTint(w, c, pal.Surface)
	wide := side * 0.07
	radius := side/2 - wide*1.1
	share := c.Share()
	if c.Ringing {
		share = 1
	} else {
		a.sweep(w, c, 2*math.Pi*radius)
	}
	label, under := timerReads(c)

	big := clockview.Lettering(glyph.TypefaceBold, 0, pal.Text)
	big.Size = clockview.Fitted(w, label, big, side*0.56, side*0.24)
	small := clockview.Lettering(glyph.TypefaceRegular, 0, pal.Muted)
	if c.Ringing {
		big = iconStyle(color(tint))
		big.Size = side * 0.24
		label = timerGlyph(c)
		under = say.T("timer.stop")
		small = clockview.Lettering(glyph.TypefaceBold, 0, tint)
	}
	if under != "" {
		small.Size = clockview.Fitted(w, under, small, side*0.5, side*0.08)
	}

	fill := color(tint)
	track := color(tint.Blend(pal.Surface, 0.8))
	ring := gogui.DrawCanvas(gogui.DrawCanvasCfg{
		ID:      "timer-dial",
		Width:   side,
		Height:  side,
		Version: uint64(time.Now().UnixNano()),
		OnDraw: func(dc *gogui.DrawContext) {
			cx, cy := side/2, side/2
			drawRing(dc, cx, cy, radius, wide, share, fill, track)

			tall := dc.FontHeight(big)
			block, gap := tall, float32(0)
			if under != "" {
				gap = side * 0.02
				block += gap + dc.FontHeight(small)
			}
			top := cy - block/2
			dc.Text(cx-dc.TextWidth(label, big)/2, top, label, big)
			if under != "" {
				dc.Text(cx-dc.TextWidth(under, small)/2, top+tall+gap, under, small)
			}
		},
	})
	disc := a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:      "timer",
		Width:   side,
		Height:  side,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Clip:    true,
		Content: []gogui.View{ring},
	})
	disc.Radius = gogui.RadiusPx(side / 2)
	return a.ui.Presses.Pressable(gogui.Column, blocking(disc), a.timerTap(c))
}

func (a *App) timerFull(w *gogui.Window, c timer.Card) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	fw, fh := float32(vw), float32(vh)
	pal := a.ui.Palette()
	label, name := timerReads(c)

	back, ink, muted := pal.Background, pal.Text, pal.Muted
	if c.Ringing {
		deeper := white
		if theme.Dark(pal.Warning) {
			deeper = black
		}
		back = a.flash(w, pal.Warning, deeper)
		ink = pal.Contrast(pal.Warning)
		muted = ink
	}

	big := clockview.Lettering(glyph.TypefaceBold, 0, ink)
	big.Size = clockview.Fitted(w, label, big, fw*0.86, fh*0.42)
	var middle []gogui.View
	if c.Ringing {
		bell := iconStyle(color(ink))
		bell.Size = big.Size * 0.5
		middle = append(middle, gogui.Label(timerGlyph(c), bell))
	}
	middle = append(middle, gogui.Label(label, big))
	if name != "" {
		st := t.TextStyleTitle
		st.Color, st.Align = color(muted), gogui.TextAlignCenter
		middle = append(middle, gogui.Label(name, st))
	}
	content := []gogui.View{gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingLarge,
		Clip:    true,
		Content: middle,
	})}
	if c.Ringing {
		key := a.loudStopKey(w, t.TextStyleTitle, style.Reach()*1.6, fw*0.4, c)
		content = append(content, gogui.Row(gogui.ContainerCfg{
			Sizing:  gogui.FillFit,
			HAlign:  gogui.HAlignCenter,
			Padding: gogui.NoPadding,
			Content: []gogui.View{key},
		}))
	}

	var tide *gogui.GradientDef
	if !c.Ringing {
		a.sweep(w, c, fh)
		drained := 1 - float32(c.Share())
		full := color(back)
		tinted := color(pal.Accent).WithOpacity(0.28).Over(full)
		tide = &gogui.GradientDef{
			Direction: gogui.GradientToBottom,
			Stops: []gogui.GradientStop{
				{Color: full, Pos: 0},
				{Color: full, Pos: drained},
				{Color: tinted, Pos: drained},
				{Color: tinted, Pos: 1},
			},
		}
	}
	return a.ui.Presses.Pressable(gogui.Column, blocking(gogui.ContainerCfg{
		ID:       "timer",
		Width:    fw,
		Height:   fh,
		Sizing:   gogui.FixedFixed,
		Color:    color(back),
		Radius:   gogui.NoRadius,
		Gradient: tide,
		Padding:  gogui.PaddingLarge,
		Spacing:  gogui.SpacingLarge,
		Content:  content,
	}), a.timerTap(c))
}

func (a *App) timerFace(w *gogui.Window, c timer.Card) gogui.View {
	vw, vh := w.WindowSize()
	cfg := a.o.Settings()
	pal := a.ui.Palette()
	ink := cfg.Clock.Ink.Over(pal)
	ink.Text = a.timerTint(w, c, ink.Surface)

	kind := timerFaceKind(cfg.Clock.Face)
	build, ok := clockview.Of(kind)
	if !ok {
		build = clockview.Plain
	}
	left, name := c.Left, c.Name
	if c.Ringing {
		left, name = 0, ""
	}
	bw := float32(vw) * 0.8
	bh := min(float32(vh)*0.55, bw*0.62)
	box := ui.Rect{W: int(bw), H: int(bh)}
	panel := a.ui.Presses.Pressable(gogui.Column, blocking(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:      "timer",
		Width:   bw,
		Height:  bh,
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Clip:    true,
		Content: []gogui.View{build(w, box, timerReading(left, kind, name), ink)},
	})), a.timerTap(c))
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Content: []gogui.View{panel},
	})
}

func (a *App) timerCorner(w *gogui.Window, c timer.Card) gogui.View {
	if c.Ringing {
		return a.timerDial(w, c)
	}
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	label := timer.Remaining(c.Left)

	st := t.TextStyleTitle
	st.Color = color(pal.Text)
	mini := st.Size * 1.6
	wide := mini * 0.16
	radius := mini/2 - wide/2
	share := c.Share()
	a.sweep(w, c, 2*math.Pi*radius)
	fill, track := color(pal.Accent), color(pal.Accent.Blend(pal.Surface, 0.75))
	ring := gogui.DrawCanvas(gogui.DrawCanvasCfg{
		ID:      "timer-corner",
		Width:   mini,
		Height:  mini,
		Version: uint64(time.Now().UnixNano()),
		OnDraw: func(dc *gogui.DrawContext) {
			drawRing(dc, mini/2, mini/2, radius, wide, share, fill, track)
		},
	})

	pad := t.Cfg.PaddingMedium.Left
	gap := t.Cfg.SpacingMedium
	words := w.TextWidth(zeroed(label), st) * 1.05
	cw, ch := 2*pad+mini+gap+words, 2*pad+mini
	chip := a.ui.Presses.Pressable(gogui.Row, blocking(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:      "timer",
		Width:   cw,
		Height:  ch,
		Sizing:  gogui.FixedFixed,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingMedium,
		Clip:    true,
		Content: []gogui.View{ring, gogui.Label(label, st)},
	})), a.timerTap(c))

	margin := reach() * 0.5
	marks := a.o.Marks()
	left := crowds(a.weatherShown())
	mark := marks.Camera
	if left {
		mark = marks.Mic
	}
	top := margin
	if mark {
		top = max(top, markSize(vw, vh)*1.3)
	}
	x, y := cornerSpot(vw, vh, cw, margin, top)
	if left {
		x = float32(vw) - x - cw
	}
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NoPadding,
		Content: []gogui.View{
			widgets.Placed(ui.Rect{X: int(x), Y: int(y), W: int(cw), H: int(ch)}, chip),
		},
	})
}

func (a *App) timerTint(w *gogui.Window, c timer.Card, toward theme.Color) theme.Color {
	pal := a.ui.Palette()
	if !c.Ringing {
		return pal.Accent
	}
	return a.flash(w, pal.Warning, toward)
}

func (a *App) timerTap(c timer.Card) func(gogui.EventCtx) {
	if !c.Ringing {
		return nil
	}
	return func(gogui.EventCtx) { a.o.Timers.Stop() }
}

func (a *App) stopKey(
	w *gogui.Window,
	st gogui.TextStyle,
	tall, least float32,
	c timer.Card,
) gogui.View {
	text := say.T("timer.stop")
	cfg := stopShape(w, text, st, tall, least)
	cfg.Color = widgets.KeyFill(a.ui.Palette())
	return a.ui.Toolkit.Keyed(cfg, text, st, style.Rest, a.timerTap(c))
}

func (a *App) loudStopKey(
	w *gogui.Window,
	st gogui.TextStyle,
	tall, least float32,
	c timer.Card,
) gogui.View {
	pal := a.ui.Palette()
	text := say.T("timer.stop")
	cfg := stopShape(w, text, st, tall, least)
	cfg.Color = color(pal.Contrast(pal.Warning))
	cfg.HAlign, cfg.VAlign, cfg.Padding = gogui.HAlignCenter, gogui.VAlignMiddle, gogui.NoPadding
	st.Color = color(pal.Warning)
	cfg.Content = []gogui.View{gogui.Label(text, st)}
	return a.ui.Presses.Pressable(gogui.Row, cfg, a.timerTap(c))
}

func stopShape(
	w *gogui.Window,
	text string,
	st gogui.TextStyle,
	tall, least float32,
) gogui.ContainerCfg {
	return gogui.ContainerCfg{
		ID:     "timer-stop",
		Width:  max(keyWidth(w, text, st, tall), least),
		Height: tall,
		Sizing: gogui.FixedFixed,
		Radius: gogui.RadiusLarge,
	}
}

func (a *App) sweep(w *gogui.Window, c timer.Card, length float32) {
	if frame := sweepFrame(c.Of, length); frame < sweepSlowest {
		a.again(w, frame)
	}
}

func sweepFrame(of time.Duration, length float32) time.Duration {
	if of <= 0 || length <= 0 {
		return sweepSlowest
	}
	step := time.Duration(float64(of) * sweepStep / float64(length))
	return min(max(step, sweepFastest), sweepSlowest)
}

func ringArc(share float64) (start, sweep float32) {
	turn := 2 * math.Pi
	return float32(-math.Pi/2 + turn*(1-share)), float32(turn * share)
}

func drawRing(
	dc *gogui.DrawContext,
	cx, cy, radius, wide float32,
	share float64,
	fill, track gogui.Color,
) {
	dc.Circle(cx, cy, radius, track, wide)
	if share <= 0 {
		return
	}
	start, sweep := ringArc(share)
	dc.Arc(cx, cy, radius, radius, start, sweep, fill, wide)
	for _, at := range []float32{start, start + sweep} {
		dx, dy := math.Cos(float64(at)), math.Sin(float64(at))
		dc.FilledCircle(cx+radius*float32(dx), cy+radius*float32(dy), wide/2, fill)
	}
}

func (a *App) weatherShown() (schema.Placement, bool) {
	cfg := a.o.Settings().Weather
	covering, _ := a.nav.Showing(a.o.Shell.Views())
	if covering == nil {
		return cfg.Placement, cfg.Dashboard && a.o.Offering() == ""
	}
	if _, idle := covering.View.(*idleview.View); idle {
		return cfg.SaverLayout().Placement, cfg.Idle
	}
	return schema.Placement{}, false
}

func crowds(p schema.Placement, shown bool) bool {
	return shown && p.Position == schema.PositionTop && p.Align == schema.AlignRight
}

func cornerSpot(vw, vh int, wide, margin, top float32) (x, y float32) {
	x, y = float32(vw)-margin-wide, top
	short, long := float32(min(vw, vh)), float32(max(vw, vh))
	if long > short*squarish {
		return x, y
	}
	reach := (short/2 - margin) / math.Sqrt2
	return min(x, float32(vw)/2+reach-wide), max(y, float32(vh)/2-reach)
}

func timerReads(c timer.Card) (label, name string) {
	if c.Ringing {
		return timer.Remaining(0), ""
	}
	return timer.Remaining(c.Left), c.Name
}

func timerGlyph(c timer.Card) string {
	if c.Ringing {
		return gogui.IconBell
	}
	return gogui.IconClock
}

func timerFaceKind(k clockkind.Kind) clockkind.Kind {
	switch k {
	case clockkind.Analog:
		return clockkind.AnalogSeconds
	case clockkind.None, clockkind.Words:
		return clockkind.Plain
	}
	return k
}

func timerReading(left time.Duration, kind clockkind.Kind, name string) face.Reading {
	whole := int((max(left, 0) + time.Second - 1) / time.Second)
	h, m, s := whole/3600, whole/60%60, whole%60
	r := face.Reading{Date: name, Second: s}
	if h > 0 || kind == clockkind.AnalogSeconds {
		r.Time = fmt.Sprintf("%d:%02d", h, m)
		return r
	}
	r.Time = fmt.Sprintf("%d:%02d", m, s)
	return r
}

func zeroed(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return '0'
		}
		return r
	}, s)
}
