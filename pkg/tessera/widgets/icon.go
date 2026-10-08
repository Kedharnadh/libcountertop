package widgets

import (
	"github.com/ygelfand/libcountertop/pkg/display/icons"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

var weatherIcons = map[string]rune{
	"sunny": 0xF0599, "clear-night": 0xF0594, "cloudy": 0xF0590, "partlycloudy": 0xF0595,
	"rainy": 0xF0597, "pouring": 0xF0596, "snowy": 0xF0598, "snowy-rainy": 0xF067F,
	"fog": 0xF0591, "hail": 0xF0592, "lightning": 0xF0593, "lightning-rainy": 0xF067E,
	"windy": 0xF059D, "windy-variant": 0xF059D, "exceptional": 0xF05D6,
}

var alarmIcons = map[string]rune{
	"disarmed":            0xF099E,
	"armed_home":          0xF068A,
	"armed_away":          0xF099D,
	"armed_night":         0xF1828,
	"armed_vacation":      0xF06BB,
	"armed_custom_bypass": 0xF0483,
	"pending":             0xF0499,
	"triggered":           0xF009E,
}

func lockIcon(state string) rune {
	switch {
	case state == "jammed":
		return 0xF08EE
	case lockMoving(state):
		return 0xF097F
	case state == "unlocked" || state == "open":
		return 0xF0FC6
	}
	return 0xF033E
}

func iconFor(t tessera.Tile, asking bool) rune {
	d := t.Domain()
	if d == "lock" && asking {
		return 0xF0FC6
	}
	if r := icons.Codepoint(t.Icon); r != 0 {
		return r
	}
	off := t.State == "off"
	switch d {
	case "light":
		if off {
			return 0xF0E4F
		}
		return 0xF0335
	case "climate":
		return 0xF001B
	case "humidifier":
		if off {
			return 0xF1466
		}
		return 0xF1099
	case "vacuum":
		return 0xF070D
	case "fan":
		return 0xF0210
	case "cover":
		return 0xF111C
	case "scene", "script":
		return 0xF04B9
	case "automation":
		if off {
			return 0xF16A7
		}
		return 0xF06A9
	case "remote":
		if off {
			return 0xF0EC4
		}
		return 0xF0454
	case "weather":
		if r, ok := weatherIcons[t.State]; ok && available(t) {
			return r
		}
		return 0xF0595
	case "sensor", "binary_sensor":
		return 0xF029A
	case "sun":
		if t.State == "above_horizon" {
			return 0xF059B
		}
		return 0xF059C
	case "timer":
		return 0xF051B
	case "person":
		return 0xF0004
	case "camera", "image":
		return 0xF07AE
	case "alarm_control_panel":
		if r, ok := alarmIcons[t.State]; ok {
			return r
		}
		return 0xF0498
	case "lock":
		return lockIcon(t.State)
	}
	return 0xF0425
}
