package weather

import (
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const entityMost = 128

type entities struct {
	entity    *esphome.Text
	look      *esphome.Select
	dashboard *esphome.Switch
	idle      *esphome.Switch
	animate   *esphome.Switch
	themed    *esphome.Switch
	position  *esphome.Select
	align     *esphome.Select
	size      *esphome.Select
}

type labeled interface{ Label() string }

func labels[T labeled](values []T) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		out = append(out, v.Label())
	}
	return out
}

func byLabel[T labeled](values []T, label string, use func(T)) {
	for _, v := range values {
		if v.Label() == label {
			use(v)
			return
		}
	}
}

func (w *Weather) Name() string { return "weather" }

func (w *Weather) base(id, name, icon string) esphome.Base {
	return esphome.Base{
		ObjectID: id,
		Name:     name,
		Icon:     icon,
		Category: esphome.CategoryConfig,
		DeviceID: w.opts.DeviceID,
	}
}

func (w *Weather) build() {
	w.ha = entities{
		entity: &esphome.Text{
			Base:      w.base("weather_entity", "Weather entity", "mdi:weather-partly-cloudy"),
			MaxLength: entityMost,
		},
		look: &esphome.Select{
			Base:    w.base("weather_look", "Weather look", "mdi:palette-outline"),
			Options: labels(schema.WeatherLooks()),
		},
		dashboard: &esphome.Switch{
			Base: w.base("weather_dashboard", "Weather on dashboard", "mdi:view-dashboard-outline"),
		},
		idle: &esphome.Switch{
			Base: w.base("weather_screensaver", "Weather on screen saver", "mdi:monitor-star"),
		},
		animate: &esphome.Switch{
			Base: w.base("weather_animate", "Animate weather", "mdi:animation-play-outline"),
		},
		themed: &esphome.Switch{
			Base: w.base("weather_themed", "Themed weather colors", "mdi:palette"),
		},
		position: &esphome.Select{
			Base:    w.base("weather_position", "Weather vertical position", "mdi:arrow-up-down"),
			Options: labels(schema.Positions()),
		},
		align: &esphome.Select{
			Base:    w.base("weather_align", "Weather horizontal position", "mdi:arrow-left-right"),
			Options: labels(schema.Aligns()),
		},
		size: &esphome.Select{
			Base:    w.base("weather_size", "Weather size", "mdi:resize"),
			Options: labels(schema.Sizes()),
		},
	}
	h := w.ha
	h.entity.OnCommand = w.SetEntity
	h.look.OnCommand = func(l string) { byLabel(schema.WeatherLooks(), l, w.SetLook) }
	h.dashboard.OnCommand = w.SetDashboard
	h.idle.OnCommand = w.SetIdle
	h.animate.OnCommand = w.SetAnimate
	h.themed.OnCommand = w.SetThemed
	h.position.OnCommand = func(l string) { byLabel(schema.Positions(), l, w.SetPosition) }
	h.align.OnCommand = func(l string) { byLabel(schema.Aligns(), l, w.SetAlign) }
	h.size.OnCommand = func(l string) { byLabel(schema.Sizes(), l, w.SetSize) }
}

func (w *Weather) Entities() []esphome.Entity {
	h := w.ha
	return []esphome.Entity{
		h.entity,
		h.look,
		h.dashboard,
		h.idle,
		h.animate,
		h.themed,
		h.position,
		h.align,
		h.size,
	}
}

func (w *Weather) Publish(c schema.Weather) {
	h := w.ha
	h.entity.Set(c.Entity)
	h.look.Set(c.Look.Label())
	h.dashboard.Set(c.Dashboard)
	h.idle.Set(c.Idle)
	h.animate.Set(c.Animate)
	h.themed.Set(c.Themed)
	h.position.Set(c.Position.Label())
	h.align.Set(c.Align.Label())
	h.size.Set(c.Size.Label())
}
