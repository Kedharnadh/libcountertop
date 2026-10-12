package weatherview

import (
	"math"
	"strings"

	"github.com/go-gui-org/go-glyph"
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/clockview"
	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/nws"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Alerts interface {
	Current() (nws.Alert, bool)
	Dismiss(nws.Alert)
}

type hazard int

const (
	hazardGeneric hazard = iota
	hazardTornado
	hazardHurricane
	hazardStorm
	hazardFlood
	hazardWinter
	hazardHeat
	hazardFire
	hazardWind
	hazardFog
)

func hazardOf(event string) hazard {
	e := strings.ToLower(event)
	has := func(words ...string) bool {
		for _, w := range words {
			if strings.Contains(e, w) {
				return true
			}
		}
		return false
	}
	switch {
	case has("tornado"):
		return hazardTornado
	case has("hurricane", "tropical", "typhoon"):
		return hazardHurricane
	case has("thunderstorm", "lightning"):
		return hazardStorm
	case has("flood", "surge", "tsunami"):
		return hazardFlood
	case has("winter", "blizzard", "snow", "ice", "freez", "frost", "cold", "chill"):
		return hazardWinter
	case has("heat"):
		return hazardHeat
	case has("fire", "red flag", "smoke"):
		return hazardFire
	case has("wind", "dust"):
		return hazardWind
	case has("fog"):
		return hazardFog
	}
	return hazardGeneric
}

func (v *View) alerting() (nws.Alert, bool) {
	if v.Alerts == nil {
		return nws.Alert{}, false
	}
	return v.Alerts.Current()
}

func (v *View) Alerting() bool {
	_, ok := v.alerting()
	return ok
}

func (v *View) alert(
	w *gogui.Window,
	size float32,
	a nws.Alert,
	ink theme.Theme,
	l schema.WeatherLayout,
) gogui.View {
	v.Again(w, weatherFrame)
	p := v.since()
	k := iconColors(ink, l)
	tone := gogui.RGB(0xff, 0x8a, 0x1f)
	if a.Severity >= nws.Extreme {
		tone = gogui.RGB(0xff, 0x3b, 0x30)
	}
	title := clockview.Lettering(glyph.TypefaceBold, size*0.55, ink.Text)
	small := clockview.Lettering(glyph.TypefaceRegular, size*0.32, ink.Muted)
	words := []gogui.View{gogui.Label(a.Event, title)}
	if !a.Expires.IsZero() {
		layout := "3:04 PM"
		if v.Clock24 != nil && v.Clock24() {
			layout = "15:04"
		}
		words = append(words, gogui.Label(
			say.F("weather.alert.until", map[string]any{"T": a.Expires.Local().Format(layout)}),
			small,
		))
	}
	cross := clockview.Lettering(glyph.TypefaceRegular, size*0.45, ink.Muted)
	cross.Family = icons.Family
	close := gogui.Column(gogui.ContainerCfg{
		ID:      "weather-alert-dismiss",
		Padding: gogui.NewPadding(0, 0, 0, size*0.2),
		Content: []gogui.View{gogui.Label(icons.Glyph(icons.Named("close")), cross)},
		OnClick: func(e gogui.EventCtx) {
			v.Alerts.Dismiss(a)
			e.Consume()
		},
	})
	panel := v.UI.Toolkit.Panel(gogui.ContainerCfg{
		ID:          "weather-alert",
		Padding:     gogui.PaddingLarge,
		Spacing:     gogui.SpacingPx(size * 0.3),
		VAlign:      gogui.VAlignMiddle,
		ColorBorder: tone,
		SizeBorder:  gogui.BorderPx(max(2, size*0.06)),
		Content: []gogui.View{
			hazardIcon(hazardOf(a.Event), p, size*1.5, k, tone),
			gogui.Column(gogui.ContainerCfg{Padding: gogui.NoPadding, Content: words}),
			close,
		},
		OnClick: func(e gogui.EventCtx) {
			if v.Radar != nil {
				v.Radar(&a)
				e.Consume()
			}
		},
	})
	return gogui.Row(panel)
}

func hazardIcon(h hazard, p float64, side float32, k tints, tone gogui.Color) gogui.View {
	frac := func(rate float64) float32 { return float32(p*rate - math.Floor(p*rate)) }
	sway := func(rate float64) float32 { return float32(math.Sin(p * rate)) }
	return gogui.DrawCanvas(gogui.DrawCanvasCfg{
		Width:   side,
		Height:  side,
		Sizing:  gogui.FixedFixed,
		Version: uint64(h) + uint64(side)<<8 + uint64(p*20)<<24,
		OnDraw: func(dc *gogui.DrawContext) {
			s := side
			switch h {
			case hazardTornado:
				for i := range 7 {
					t := float32(i) / 6
					y := s * (0.18 + 0.7*t)
					rx := s * (0.36 - 0.3*t)
					dx := sway(2.2+float64(i)*0.5) * s * 0.06 * t
					dc.Arc(s/2+dx, y, rx, s*0.05, 0, 2*math.Pi, k.cloud, s*0.045)
				}
				dc.Line(s*0.2, s*0.86, s*0.8, s*0.86, tone, s*0.04)
			case hazardHurricane:
				turn := p * 2.4
				for arm := range 3 {
					a := turn + float64(arm)*2*math.Pi/3
					dc.Arc(s/2, s/2, s*0.36, s*0.36, float32(a), math.Pi*0.7, k.cloud, s*0.07)
				}
				dc.FilledCircle(s/2, s/2, s*0.08, tone)
			case hazardStorm:
				drawCloud(dc, s/2+sway(0.8)*s*0.03, s*0.36, s*0.8, k.cloud)
				bolt := tone
				if frac(0.7) > 0.15 {
					bolt = tone.WithOpacity(0.3)
				}
				dc.FilledPolygon(
					[]float32{s * 0.6, s * 0.5, s * 0.36, s * 0.78, s * 0.56, s * 0.74},
					bolt,
				)
				dc.FilledPolygon(
					[]float32{s * 0.48, s * 0.7, s * 0.68, s * 0.68, s * 0.42, s * 0.98},
					bolt,
				)
				drawDrops(dc, s, 2, k.drop, frac(1.6))
			case hazardFlood:
				drawCloud(dc, s/2, s*0.26, s*0.6, k.cloud)
				drawDrops(dc, s*0.7, 3, k.drop, frac(1.8))
				rise := 0.08 * (0.5 + 0.5*sway(0.6))
				for row := range 3 {
					y := s * (0.66 + 0.11*float32(row) - float32(rise))
					var pts []float32
					for i := 0; i <= 16; i++ {
						x := s * float32(i) / 16
						off := math.Sin(float64(i)*0.8 + p*2.5 + float64(row))
						pts = append(pts, x, y+float32(off)*s*0.025)
					}
					dc.Polyline(pts, k.drop.WithOpacity(1-0.25*float32(row)), s*0.045)
				}
			case hazardWinter:
				drawCloud(dc, s/2, s*0.32, s*0.75, k.cloud)
				drawFlakes(dc, s, 4, k.flake, frac(0.9), sway(2.4)*2)
			case hazardHeat:
				drawSun(dc, s/2, s*0.36, s*0.17, s*0.3, tone, s*0.05, p*0.6)
				for i := range 3 {
					y := s * (0.7 + 0.1*float32(i))
					var pts []float32
					for j := 0; j <= 12; j++ {
						x := s * (0.15 + 0.7*float32(j)/12)
						pts = append(
							pts,
							x,
							y+float32(math.Sin(float64(j)*1.1+p*4+float64(i)))*s*0.02,
						)
					}
					dc.Polyline(pts, tone.WithOpacity(0.6), s*0.035)
				}
			case hazardFire:
				for i := range 3 {
					cx := s * (0.32 + 0.18*float32(i))
					hgt := s * (0.42 + 0.12*sway(5+float64(i)*1.7))
					base := s * 0.86
					dc.FilledPolygon([]float32{
						cx - s*0.12, base,
						cx + sway(3+float64(i))*s*0.04, base - hgt,
						cx + s*0.12, base,
					}, tone.WithOpacity(0.55+0.15*float32(i)))
				}
				for i := range 3 {
					y := s * (0.6 - 0.5*frac(0.8+float64(i)*0.3))
					dc.FilledCircle(
						s*(0.35+0.15*float32(i))+sway(2+float64(i))*s*0.04,
						y,
						s*0.025,
						tone,
					)
				}
			case hazardWind:
				for i, ln := range []float32{0.7, 0.55, 0.8} {
					ln *= 1 + 0.18*sway(2.2+float64(i)*0.4)
					y := s * (0.32 + 0.18*float32(i))
					dc.Line(s*0.12, y, s*(0.12+ln*0.7), y, k.wind, s*0.06)
				}
			case hazardFog:
				for i := range 4 {
					y := s*0.3 + float32(i)*s*0.14
					dx := sway(0.9+float64(i)*0.3) * s * 0.08
					dc.Line(s*0.15+dx, y, s*0.85+dx, y, k.fog, s*0.06)
				}
			default:
				glow := 0.55 + 0.45*(0.5+0.5*sway(3))
				dc.FilledPolygon(
					[]float32{s / 2, s * 0.1, s * 0.92, s * 0.86, s * 0.08, s * 0.86},
					tone.WithOpacity(glow),
				)
				dc.FilledRect(s*0.47, s*0.34, s*0.06, s*0.3, k.cloud)
				dc.FilledCircle(s/2, s*0.74, s*0.04, k.cloud)
			}
		},
	})
}
