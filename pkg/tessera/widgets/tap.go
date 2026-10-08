package widgets

import (
	"strings"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

type route int

const (
	nothing route = iota
	service
	custom
	card
	guarded
)

type tap struct {
	route   route
	service string
}

var cardDomains = map[string]bool{
	"sensor": true, "binary_sensor": true, "weather": true, "number": true, "input_number": true,
	"select": true, "input_select": true, "media_player": true, "vacuum": true, "cover": true,
	"sun": true, "person": true, "timer": true, "climate": true, "humidifier": true,
	"alarm_control_panel": true, "remote": true,
}

func routeOf(t tessera.Tile, hold bool) tap {
	d := t.Domain()
	if t.Tap == "none" {
		return tap{}
	}
	if !hold && t.Tap == "action" && t.Action().Service != "" {
		return tap{route: custom, service: t.Action().Service}
	}
	if !hold && t.Tap == "toggle" {
		return tap{route: service, service: d + ".toggle"}
	}
	if d == "automation" && t.Tap != "detail" {
		if hold != runs(t) {
			return tap{route: service, service: "automation.trigger"}
		}
		return tap{route: service, service: "automation.toggle"}
	}
	if d == "lock" {
		if hold || t.Tap == "detail" {
			return tap{route: card}
		}
		return tap{route: guarded}
	}
	open := hold || t.Tap == "detail" || d == "vacuum" || d == "cover"
	if d == "timer" && !open {
		if t.State == "active" {
			return tap{route: service, service: "timer.pause"}
		}
		return tap{route: service, service: "timer.start"}
	}
	if cardDomains[d] || open {
		return tap{route: card}
	}
	switch d {
	case "light", "switch", "input_boolean", "fan":
		return tap{route: service, service: d + ".toggle"}
	case "scene", "script":
		return tap{route: service, service: d + ".turn_on"}
	case "button", "input_button":
		return tap{route: service, service: d + ".press"}
	}
	return tap{}
}

func flips(action string) bool {
	return strings.HasSuffix(action, ".toggle") && !strings.HasPrefix(action, "automation.")
}
