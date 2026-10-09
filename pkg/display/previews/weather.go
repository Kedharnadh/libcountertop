package previews

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/homeassistant/weather"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type weatherTile struct {
	kit    Kit
	layout schema.WeatherLayout
}

func (k Kit) Weather(l schema.WeatherLayout) menu.Preview { return weatherTile{k, l} }

func (t weatherTile) View(w *gogui.Window, width, height float32) gogui.View {
	return t.kit.weather.Sample(w, sample(), t.layout, width, height)
}

func sample() weather.Reading {
	t := 72.0
	return weather.Reading{Condition: "sunny", Temperature: &t}
}
