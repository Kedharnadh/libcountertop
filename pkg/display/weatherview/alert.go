package weatherview

import (
	"bytes"
	"embed"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"strings"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/style"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/nws"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

//go:embed hazards/*.png
var hazards embed.FS

type Alerts interface {
	Current() (nws.Alert, bool)
	Dismiss(nws.Alert)
}

type hazard string

const (
	hazardGeneric   hazard = "generic"
	hazardTornado   hazard = "tornado"
	hazardHurricane hazard = "hurricane"
	hazardStorm     hazard = "storm"
	hazardFlood     hazard = "flood"
	hazardWinter    hazard = "winter"
	hazardHeat      hazard = "heat"
	hazardFire      hazard = "fire"
	hazardWind      hazard = "wind"
	hazardFog       hazard = "fog"
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

func (h hazard) picture() string {
	key := "hazard/" + string(h)
	if gogui.HasImage(key) {
		return "mem:" + key
	}
	raw, err := hazards.ReadFile("hazards/" + string(h) + ".png")
	if err != nil {
		slog.Warn("a hazard picture is missing", "hazard", h, "err", err)
		return ""
	}
	src, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		slog.Warn("a hazard picture would not decode", "hazard", h, "err", err)
		return ""
	}
	img := image.NewNRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, src.Bounds().Min, draw.Src)
	return gogui.UseImage(key, img.Rect.Dx(), img.Rect.Dy(), img.Pix)
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
	_ schema.WeatherLayout,
) gogui.View {
	t := gogui.CurrentTheme()
	tone := ink.Warning
	if a.Severity >= nws.Extreme {
		tone = ink.Danger
	}

	head := t.TextStyleTitle
	head.Size, head.Color = size*0.55, widgets.Color(tone)
	small := widgets.Secondary()
	small.Size, small.Color = size*0.32, widgets.Color(ink.Muted)
	words := []gogui.View{gogui.Label(a.Event, head)}
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

	content := []gogui.View{
		widgets.Edge(
			widgets.Color(widgets.Pulse(w, v.Again, tone, ink.Background)),
			t.Cfg.RadiusMedium,
		),
	}
	side := size * 1.5
	if src := hazardOf(a.Event).picture(); src != "" {
		content = append(content, gogui.Image(gogui.ImageCfg{Src: src, Width: side, Height: side}))
	}
	content = append(content,
		gogui.Column(gogui.ContainerCfg{Padding: gogui.NoPadding, Content: words}),
		v.dismiss(w, a, ink),
	)
	return gogui.Row(v.UI.Toolkit.Panel(gogui.ContainerCfg{
		ID:      "weather-alert",
		Padding: gogui.PaddingMedium,
		Spacing: gogui.SpacingMedium,
		VAlign:  gogui.VAlignMiddle,
		Content: content,
		OnClick: func(e gogui.EventCtx) {
			if v.Radar != nil {
				v.Radar(&a)
				e.Consume()
			}
		},
	}))
}

func (v *View) dismiss(w *gogui.Window, a nws.Alert, ink theme.Theme) gogui.View {
	st := gogui.CurrentTheme().Cfg.TextStyleDef
	label := say.T("weather.alert.dismiss")
	tall := style.Reach()
	return v.UI.Toolkit.Keyed(gogui.ContainerCfg{
		ID:     "weather-alert-dismiss",
		Width:  widgets.KeyWidth(w, label, st, tall),
		Height: tall,
		Sizing: gogui.FixedFixed,
		Radius: gogui.RadiusLarge,
		Color:  widgets.KeyFill(ink),
	}, label, st, style.Rest, func(e gogui.EventCtx) {
		v.Alerts.Dismiss(a)
		e.Consume()
	})
}
