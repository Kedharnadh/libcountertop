package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Weather struct {
	Entity    string `json:"entity"`
	Dashboard bool   `json:"dashboard"`
	Idle      bool   `json:"idle"`
	WeatherLayout
	Saver WeatherLayout `json:"saver"`
}

type WeatherLayout struct {
	Look    WeatherLook `json:"look"`
	Animate bool        `json:"animate"`
	Themed  bool        `json:"themed"`
	Placement
}

func (w Weather) SaverLayout() WeatherLayout {
	if w.Saver.Look == "" {
		return w.WeatherLayout
	}
	return w.Saver
}

func (w *Weather) SeedSaver() *WeatherLayout {
	if w.Saver.Look == "" {
		w.Saver = w.WeatherLayout
	}
	return &w.Saver
}

type WeatherLook string

const (
	WeatherCompact WeatherLook = "compact"
	WeatherStack   WeatherLook = "stack"
	WeatherIcon    WeatherLook = "icon"
	WeatherWords   WeatherLook = "words"
	WeatherCard    WeatherLook = "card"
	WeatherDetail  WeatherLook = "detail"
)

func WeatherLooks() []WeatherLook {
	return []WeatherLook{
		WeatherCompact,
		WeatherStack,
		WeatherIcon,
		WeatherWords,
		WeatherCard,
		WeatherDetail,
	}
}

func (l WeatherLook) Label() string { return say.T("weather.look." + string(l)) }

func DefaultWeather() Weather {
	return Weather{
		Dashboard: true,
		WeatherLayout: WeatherLayout{
			Look:      WeatherCompact,
			Animate:   true,
			Placement: Placement{Position: PositionTop, Align: AlignRight, Size: SizeSmall},
		},
	}
}
