package schema

import "github.com/ygelfand/libcountertop/pkg/say"

type Weather struct {
	Entity    string      `json:"entity"`
	Look      WeatherLook `json:"look"`
	Dashboard bool        `json:"dashboard"`
	Idle      bool        `json:"idle"`
	Animate   bool        `json:"animate"`
	Themed    bool        `json:"themed"`
	Position  Position    `json:"position"`
	Align     Align       `json:"align"`
	Size      Size        `json:"size"`
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
		Look:      WeatherCompact,
		Dashboard: true,
		Animate:   true,
		Position:  PositionTop,
		Align:     AlignRight,
		Size:      SizeSmall,
	}
}
