package widgets

import (
	"math"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

const (
	FlashFrame  = time.Second / 30
	flashPeriod = 1400 * time.Millisecond
	flashDepth  = 0.35
)

var (
	Black = theme.Color{}
	White = theme.Color{R: 0xff, G: 0xff, B: 0xff}
)

func Color(c theme.Color) gogui.Color { return gogui.RGB(c.R, c.G, c.B) }

func Secondary() gogui.TextStyle {
	t := gogui.CurrentTheme().Cfg
	st := t.TextStyleDef
	st.Color = t.ColorTextSecondary
	return st
}

func Blocking(cfg gogui.ContainerCfg) gogui.ContainerCfg {
	cfg.OnClick, cfg.OnGesture = consume, consume
	return cfg
}

func consume(e gogui.EventCtx) { e.Consume() }

func Pulse(
	w *gogui.Window,
	again func(*gogui.Window, time.Duration),
	c, toward theme.Color,
) theme.Color {
	again(w, FlashFrame)
	return Flash(c, toward, time.Now())
}

func IconStyle(c gogui.Color) gogui.TextStyle {
	st := gogui.CurrentTheme().TextStyleIconMedium
	st.Color = c
	return st
}

func Flash(c, toward theme.Color, at time.Time) theme.Color {
	phase := float64(at.UnixNano()%int64(flashPeriod)) / float64(flashPeriod)
	return c.Blend(toward, flashDepth*(1-math.Cos(2*math.Pi*phase))/2)
}

func Edge(c gogui.Color, wide float32) gogui.View {
	return gogui.Column(gogui.ContainerCfg{
		Width:   wide,
		Sizing:  gogui.FixedFill,
		Color:   c,
		Radius:  gogui.RadiusSmall,
		Padding: gogui.NoPadding,
	})
}

func KeyWidth(w *gogui.Window, text string, st gogui.TextStyle, tall float32) float32 {
	return max(tall*1.5, w.TextWidth(text, st)+tall*0.6)
}
