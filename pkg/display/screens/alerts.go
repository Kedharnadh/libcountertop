package screens

import (
	"image"
	"math"
	"strconv"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	mdi "github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/message"
	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	priorityAlert = 30
	flashFrame    = time.Second / 30
	flashPeriod   = 1400 * time.Millisecond
	flashDepth    = 0.35
	bannerLines   = 3
)

var (
	black = theme.Color{}
	white = theme.Color{R: 0xff, G: 0xff, B: 0xff}
)

func (a *App) alert(w *gogui.Window) gogui.View {
	m, ok := a.o.Messages.Current()
	if !ok {
		return nil
	}
	switch a.o.Settings().Screen.Alert {
	case schema.AlertCard:
		return a.alertCard(w, m)
	case schema.AlertFull:
		return a.alertFull(w, m)
	case schema.AlertToast:
		return a.alertToast(w, m)
	}
	return a.alertBanner(w, m)
}

func (a *App) alertCovers() bool {
	_, ok := a.o.Messages.Current()
	return ok && a.o.Settings().Screen.Alert == schema.AlertFull
}

func (a *App) alertBanner(w *gogui.Window, m message.Message) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	tint := m.Tint(pal)
	margin := reach() * 0.5
	width := float32(vw) - 2*margin
	keySt := t.Cfg.TextStyleDef
	beside := keysWidth(w, m, keySt, reach()) <= width*0.3

	head := t.TextStyleTitle
	head.Color = color(tint)
	words := alertWords(w, m, head, t.Cfg.TextStyleDef, bannerLines)
	if !beside {
		words = append(words, a.alertKeys(w, m, pal, keySt, reach(), true)...)
	}
	content := []gogui.View{edge(a.pulse(w, m, tint, pal.Background), t.Cfg.RadiusMedium)}
	content = append(content, messageIcon(m, color(tint), t.TextStyleTitle.Size*1.6)...)
	content = append(content, gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingSmall,
		Content: words,
	}))
	if beside {
		content = append(content, a.alertKeys(w, m, pal, keySt, reach(), false)...)
	}
	banner := gogui.Row(blocking(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:          "message",
		Width:       width,
		MaxHeight:   float32(vh) * 0.6,
		Sizing:      gogui.FixedFit,
		VAlign:      gogui.VAlignMiddle,
		Padding:     gogui.PaddingMedium,
		Spacing:     gogui.SpacingMedium,
		Clip:        true,
		AmendLayout: a.caught,
		Content:     content,
	})))
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

func (a *App) alertCard(w *gogui.Window, m message.Message) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	tint := m.Tint(pal)
	icon := messageIcon(m, color(tint), t.TextStyleDisplay.Size*1.6)

	head := t.TextStyleDisplay
	head.Color = color(tint)
	var words []gogui.View
	if vw < vh {
		words = append(words, icon...)
		icon = nil
	}
	words = append(words, alertWords(w, m, head, t.TextStyleTitle, 0)...)
	words = append(words, a.alertKeys(w, m, pal, t.Cfg.TextStyleDef, style.Reach(), true)...)

	content := []gogui.View{edge(a.pulse(w, m, tint, pal.Background), t.Cfg.RadiusMedium)}
	content = append(content, icon...)
	content = append(content, gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFit,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingMedium,
		Content: words,
	}))
	card := gogui.Row(a.ui.Toolkit.Panel(gogui.ContainerCfg{
		ID:        "message",
		Width:     float32(vw) * 0.82,
		MaxHeight: float32(vh) * 0.85,
		Sizing:    gogui.FixedFit,
		VAlign:    gogui.VAlignMiddle,
		Padding:   gogui.PaddingLarge,
		Spacing:   gogui.SpacingLarge,
		Clip:      true,
		Content:   content,
	}))
	return gogui.Column(blocking(gogui.ContainerCfg{
		Width:       float32(vw),
		Height:      float32(vh),
		Sizing:      gogui.FixedFixed,
		Padding:     gogui.NoPadding,
		Color:       gogui.Black.WithOpacity(0.5),
		HAlign:      gogui.HAlignCenter,
		VAlign:      gogui.VAlignMiddle,
		AmendLayout: a.caught,
		Content:     []gogui.View{card},
	}))
}

func (a *App) alertFull(w *gogui.Window, m message.Message) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	tint := m.Tint(pal)
	ink := pal.Contrast(tint)
	deeper := white
	if theme.Dark(tint) {
		deeper = black
	}

	head := t.TextStyleDisplay
	head.Size *= 1.3
	head.Color, head.Align = color(ink), gogui.TextAlignCenter
	body := t.TextStyleTitle
	body.Color, body.Align = color(ink), gogui.TextAlignCenter
	middle := messageIcon(m, color(ink), t.TextStyleDisplay.Size*3)
	middle = append(middle, alertWords(w, m, head, body, 0)...)

	content := []gogui.View{gogui.Column(gogui.ContainerCfg{
		Sizing:  gogui.FillFill,
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignMiddle,
		Padding: gogui.NoPadding,
		Spacing: gogui.SpacingLarge,
		Clip:    true,
		Content: middle,
	})}
	content = append(content, a.alertKeys(w, m, pal, t.TextStyleTitle, style.Reach()*1.4, true)...)
	return gogui.Column(blocking(gogui.ContainerCfg{
		ID:          "message",
		Width:       float32(vw),
		Height:      float32(vh),
		Sizing:      gogui.FixedFixed,
		Color:       a.pulse(w, m, tint, deeper),
		Padding:     gogui.PaddingLarge,
		Spacing:     gogui.SpacingLarge,
		AmendLayout: a.caught,
		Content:     content,
	}))
}

func (a *App) alertToast(w *gogui.Window, m message.Message) gogui.View {
	t := gogui.CurrentTheme()
	vw, vh := w.WindowSize()
	pal := a.ui.Palette()
	tint := m.Tint(pal)
	margin := reach() * 0.5
	room := float32(vw) - 2*margin
	st := t.Cfg.TextStyleDef
	tall := st.Size * 2
	keys := keysWidth(w, m, st, tall)
	inline := keys <= room*0.35

	icon := messageIcon(m, color(tint), st.Size*1.4)
	bar := t.Cfg.RadiusMedium * 0.6
	pad := t.Cfg.PaddingMedium.Left
	textRoom := room - 2*pad - bar - 2*t.Cfg.SpacingMedium
	if len(icon) > 0 {
		textRoom -= st.Size*1.4 + t.Cfg.SpacingMedium
	}
	if inline && keys > 0 {
		textRoom -= keys + t.Cfg.SpacingMedium
	}

	head := st
	head.Color = color(tint)
	line, rest := m.Title, m.Body
	if line == "" {
		line, rest = m.Body, ""
	}
	words := []gogui.View{gogui.Label(line, head)}
	if rest != "" && w.TextWidth(rest, secondary()) <= textRoom {
		words = append(words, gogui.Label(rest, secondary()))
	}

	top := []gogui.View{edge(a.pulse(w, m, tint, pal.Background), bar)}
	top = append(top, icon...)
	top = append(top, gogui.Column(gogui.ContainerCfg{
		MaxWidth: max(textRoom, 0),
		Padding:  gogui.NoPadding,
		Clip:     true,
		Content:  words,
	}))
	panel := gogui.ContainerCfg{
		ID:          "message",
		MaxWidth:    room,
		VAlign:      gogui.VAlignMiddle,
		Padding:     gogui.PaddingMedium,
		Spacing:     gogui.SpacingMedium,
		Clip:        true,
		AmendLayout: a.caught,
	}
	layout := gogui.Row
	if inline {
		panel.Content = append(top, a.alertKeys(w, m, pal, st, tall, false)...)
	} else {
		layout = gogui.Column
		panel.Width, panel.Sizing = room, gogui.FixedFit
		panel.Content = append([]gogui.View{gogui.Row(gogui.ContainerCfg{
			VAlign:  gogui.VAlignMiddle,
			Padding: gogui.NoPadding,
			Spacing: gogui.SpacingMedium,
			Content: top,
		})}, a.alertKeys(w, m, pal, st, tall, true)...)
	}
	toast := layout(blocking(a.ui.Toolkit.Panel(panel)))
	return gogui.Column(gogui.ContainerCfg{
		Width:   float32(vw),
		Height:  float32(vh),
		Sizing:  gogui.FixedFixed,
		Padding: gogui.NewPadding(margin, margin, reach(), margin),
		HAlign:  gogui.HAlignCenter,
		VAlign:  gogui.VAlignBottom,
		Content: []gogui.View{toast},
	})
}

func (a *App) pulse(w *gogui.Window, m message.Message, c, toward theme.Color) gogui.Color {
	if !m.Flash {
		return color(c)
	}
	return color(a.flash(w, c, toward))
}

func (a *App) flash(w *gogui.Window, c, toward theme.Color) theme.Color {
	a.again(w, flashFrame)
	at := float64(time.Now().UnixNano()%int64(flashPeriod)) / float64(flashPeriod)
	return c.Blend(toward, flashDepth*(1-math.Cos(2*math.Pi*at))/2)
}

func (a *App) caught(e gogui.EventCtx) {
	s := e.Layout.Shape
	a.o.Messages.Catch(image.Rect(int(s.X), int(s.Y), int(s.X+s.Width), int(s.Y+s.Height)))
}

func blocking(cfg gogui.ContainerCfg) gogui.ContainerCfg {
	cfg.OnClick, cfg.OnGesture = consume, consume
	return cfg
}

func consume(e gogui.EventCtx) { e.Consume() }

func edge(c gogui.Color, wide float32) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Width:   wide,
		Sizing:  gogui.FixedFill,
		Color:   c,
		Radius:  gogui.RadiusSmall,
		Padding: gogui.NoPadding,
	})
}

func alertWords(
	w *gogui.Window,
	m message.Message,
	head, body gogui.TextStyle,
	lines int,
) []gogui.View {
	var words []gogui.View
	if m.Title != "" {
		words = append(
			words,
			gogui.Text(gogui.TextCfg{Text: m.Title, TextStyle: head, Mode: gogui.TextModeWrap}),
		)
	}
	if m.Body == "" {
		return words
	}
	text := gogui.Text(gogui.TextCfg{Text: m.Body, TextStyle: body, Mode: gogui.TextModeWrap})
	if lines > 0 {
		text = gogui.Column(gogui.ContainerCfg{
			Sizing:    gogui.FillFit,
			MaxHeight: float32(lines) * lineHeight(w, body),
			Padding:   gogui.NoPadding,
			Clip:      true,
			Content:   []gogui.View{text},
		})
	}
	return append(words, text)
}

func lineHeight(w *gogui.Window, st gogui.TextStyle) float32 {
	if tm := w.TextMeasurer(); tm != nil {
		return tm.FontHeight(st)
	}
	return st.Size * 1.4
}

func messageIcon(m message.Message, ink gogui.Color, size float32) []gogui.View {
	if m.Icon == "" {
		return nil
	}
	glyph := mdi.Named(m.Icon)
	if glyph == 0 {
		glyph = mdi.Named("alert-outline")
	}
	st := gogui.CurrentTheme().TextStyleDisplay
	st.Family, st.Color, st.Size = mdi.Family, ink, size
	return []gogui.View{gogui.Label(mdi.Glyph(glyph), st)}
}

func keyWidth(w *gogui.Window, text string, st gogui.TextStyle, tall float32) float32 {
	return max(tall*1.5, w.TextWidth(text, st)+tall*0.6)
}

func keysWidth(w *gogui.Window, m message.Message, st gogui.TextStyle, tall float32) float32 {
	if len(m.Keys) == 0 {
		return 0
	}
	sum := gogui.CurrentTheme().Cfg.SpacingMedium * float32(len(m.Keys)-1)
	for _, k := range m.Keys {
		sum += keyWidth(w, k.Text, st, tall)
	}
	return sum
}

func (a *App) alertKeys(
	w *gogui.Window,
	m message.Message,
	pal theme.Theme,
	st gogui.TextStyle,
	tall float32,
	fill bool,
) []gogui.View {
	if len(m.Keys) == 0 {
		return nil
	}
	keys := make([]gogui.View, 0, len(m.Keys))
	for i, k := range m.Keys {
		cfg := gogui.ContainerCfg{
			ID:      "message-key-" + strconv.Itoa(i),
			Width:   keyWidth(w, k.Text, st, tall),
			Height:  tall,
			Sizing:  gogui.FixedFixed,
			Radius:  gogui.RadiusLarge,
			Color:   widgets.KeyFill(pal),
			HAlign:  gogui.HAlignCenter,
			VAlign:  gogui.VAlignMiddle,
			Padding: gogui.NoPadding,
		}
		if fill {
			cfg.Width, cfg.Sizing = 0, gogui.FillFixed
		}
		keys = append(keys, a.alertKey(i, k, pal, cfg, st))
	}
	row := gogui.ContainerCfg{Padding: gogui.NoPadding, Spacing: gogui.SpacingMedium, Content: keys}
	if fill {
		row.Sizing = gogui.FillFit
	}
	return []gogui.View{gogui.Row(row)}
}

func (a *App) alertKey(
	i int,
	k message.Key,
	pal theme.Theme,
	cfg gogui.ContainerCfg,
	st gogui.TextStyle,
) gogui.View {
	press := func(gogui.EventCtx) { a.o.Messages.Press(i) }
	c, ok := message.Swatch(k.Swatch, pal)
	if !ok {
		return a.ui.Toolkit.Keyed(cfg, k.Text, st, style.Rest, press)
	}
	st.Color = color(pal.Contrast(c))
	cfg.Color = color(c)
	cfg.Content = []gogui.View{gogui.Label(k.Text, st)}
	return a.ui.Presses.Pressable(gogui.Row, cfg, press)
}
