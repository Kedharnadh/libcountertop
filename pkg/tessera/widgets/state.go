package widgets

import (
	"math"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func timestamped(domain string) bool {
	return domain == "scene" || domain == "button" || domain == "input_button"
}

func available(t tessera.Tile) bool {
	if !t.Received || t.State == "" || t.State == "unavailable" {
		return false
	}
	return t.State != "unknown" || timestamped(t.Domain())
}

func runs(t tessera.Tile) bool { return t.Domain() == "automation" && t.Tap == "run" }

func running(t tessera.Tile) bool { return t.ExtraBool("run") }

func active(t tessera.Tile) bool {
	if !t.Received || t.State == "" || t.State == "unavailable" {
		return false
	}
	d := t.Domain()
	if timestamped(d) || d == "image" {
		return true
	}
	if runs(t) {
		return running(t)
	}
	if t.State == "unknown" || t.State == "off" {
		return false
	}
	switch d {
	case "cover":
		return t.State != "closed"
	case "person":
		return t.State != "not_home"
	case "media_player":
		return t.State != "standby"
	case "vacuum":
		return t.State != "idle" && t.State != "docked" && t.State != "paused"
	case "timer":
		return t.State == "active"
	case "camera":
		return t.State == "streaming" || t.State == "recording"
	case "alarm_control_panel":
		return t.State != "disarmed"
	case "lock":
		return t.State != "locked"
	}
	return true
}

func alarmClass(class string) bool {
	switch class {
	case "battery",
		"carbon_monoxide",
		"gas",
		"heat",
		"lock",
		"moisture",
		"problem",
		"safety",
		"smoke",
		"sound",
		"tamper":
		return true
	}
	return false
}

func modeColor(mode string, pal theme.Theme) (theme.Color, bool) {
	switch mode {
	case "heat":
		return pal.Roles.Heat, true
	case "cool":
		return pal.Roles.Cool, true
	case "heat_cool":
		return pal.Roles.Warm, true
	case "auto":
		return pal.Success, true
	case "fan_only":
		return pal.Roles.Air, true
	case "dry":
		return pal.Warning, true
	}
	return theme.Color{}, false
}

func weatherColor(condition string, pal theme.Theme) theme.Color {
	switch condition {
	case "clear-night", "pouring":
		return pal.Roles.Choice
	case "partlycloudy", "cloudy", "fog":
		return pal.Muted
	case "rainy":
		return pal.Roles.Cool
	case "snowy", "hail":
		return pal.Roles.Air
	case "snowy-rainy":
		return pal.Roles.Media
	}
	return pal.Roles.Warm
}

func lockMoving(state string) bool {
	return state == "locking" || state == "unlocking" || state == "opening"
}

func lampColor(hue, saturation float64, pal theme.Theme) theme.Color {
	if saturation < 10 {
		return pal.Roles.Warm
	}
	h := math.Mod(math.Mod(hue, 360)+360, 360) / 60
	s := min(max(saturation, 40), 100) / 100
	x := 1 - math.Abs(math.Mod(h, 2)-1)
	var r, g, b float64
	switch int(h) {
	case 0:
		r, g, b = 1, x, 0
	case 1:
		r, g, b = x, 1, 0
	case 2:
		r, g, b = 0, 1, x
	case 3:
		r, g, b = 0, x, 1
	case 4:
		r, g, b = x, 0, 1
	default:
		r, g, b = 1, 0, x
	}
	channel := func(c float64) byte { return byte(math.Round(255 * (1 - s + s*c))) }
	return theme.RGB(uint32(channel(r))<<16 | uint32(channel(g))<<8 | uint32(channel(b)))
}

func accent(t tessera.Tile, pal theme.Theme) theme.Color {
	r := pal.Roles
	if hs := t.Numbers("hs_color"); t.Domain() == "light" && len(hs) == 2 {
		return lampColor(hs[0], hs[1], pal)
	}
	switch d := t.Domain(); d {
	case "binary_sensor":
		if alarmClass(t.Text("device_class")) {
			return pal.Danger
		}
		return r.Warm
	case "light", "switch", "input_boolean", "script", "automation", "remote", "timer", "camera":
		return r.Warm
	case "climate":
		if c, ok := modeColor(t.State, pal); ok {
			return c
		}
		return r.Warm
	case "vacuum":
		if t.State == "error" {
			return pal.Danger
		}
		return r.Level
	case "fan":
		return r.Air
	case "humidifier":
		return r.Cool
	case "cover", "scene":
		return r.Motion
	case "media_player":
		return r.Media
	case "select", "input_select":
		return r.Choice
	case "number", "input_number":
		return r.Level
	case "weather":
		return weatherColor(t.State, pal)
	case "sun":
		if t.State == "above_horizon" {
			return r.Warm
		}
		return r.Choice
	case "person":
		if t.State == "home" {
			return pal.Success
		}
		return r.Cool
	case "alarm_control_panel":
		switch t.State {
		case "triggered":
			return pal.Danger
		case "arming", "pending", "disarming":
			return pal.Warning
		}
		return pal.Success
	case "lock":
		switch {
		case t.State == "locked":
			return pal.Success
		case lockMoving(t.State):
			return pal.Warning
		}
		return pal.Danger
	case "sensor":
		if t.Text("device_class") == "battery" {
			if charge, err := strconv.ParseFloat(t.State, 64); err == nil {
				switch {
				case charge >= 70:
					return pal.Success
				case charge >= 30:
					return pal.Warning
				}
				return pal.Danger
			}
		}
		switch t.Text("unit_of_measurement") {
		case "lx":
			return r.Warm
		case "°C", "°F":
			return r.Heat
		case "kWh", "Wh":
			return r.Motion
		case "%":
			return r.Level
		}
	}
	return pal.Accent
}

func word(t tessera.Tile, n tessera.Numbers) string {
	if !available(t) {
		return t.State
	}
	if unit := t.Text("unit_of_measurement"); unit != "" {
		return n.WithUnit(n.Localize(t.State), unit)
	}
	if w := t.ExtraText("w"); w != "" {
		return w
	}
	if _, ok := weatherIcons[t.State]; ok && t.Domain() == "weather" {
		return say.T("tessera.weather." + t.State)
	}
	return pretty(t.State)
}
