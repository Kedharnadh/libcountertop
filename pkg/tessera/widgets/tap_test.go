package widgets

import (
	"encoding/json"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func tile(entity, state, tapping string) tessera.Tile {
	return tessera.Tile{Entity: entity, State: state, Tap: tapping, Received: true}
}

func TestTapsGoWhereTheFirmwareSendsThem(t *testing.T) {
	act, _ := json.Marshal(
		map[string]any{
			"act": map[string]any{"s": "script.turn_on", "d": [][2]string{{"variables", "x"}}},
		},
	)
	scripted := tile("light.lamp", "on", "action")
	scripted.Options = act
	tests := []struct {
		name string
		t    tessera.Tile
		hold bool
		want tap
	}{
		{
			"a light toggles",
			tile("light.lamp", "on", "auto"),
			false,
			tap{route: service, service: "light.toggle"},
		},
		{"a held light opens its card", tile("light.lamp", "on", "auto"), true, tap{route: card}},
		{"none does nothing", tile("light.lamp", "on", "none"), false, tap{}},
		{
			"toggle on a cover",
			tile("cover.blind", "open", "toggle"),
			false,
			tap{route: service, service: "cover.toggle"},
		},
		{"a cover opens its card", tile("cover.blind", "open", "auto"), false, tap{route: card}},
		{
			"a scene runs",
			tile("scene.evening", "unknown", "auto"),
			false,
			tap{route: service, service: "scene.turn_on"},
		},
		{
			"a button presses",
			tile("button.bell", "unknown", "auto"),
			false,
			tap{route: service, service: "button.press"},
		},
		{
			"an automation toggles",
			tile("automation.porch", "on", "auto"),
			false,
			tap{route: service, service: "automation.toggle"},
		},
		{
			"a held automation runs",
			tile("automation.porch", "on", "auto"),
			true,
			tap{route: service, service: "automation.trigger"},
		},
		{
			"a run tile runs on a tap",
			tile("automation.porch", "on", "run"),
			false,
			tap{route: service, service: "automation.trigger"},
		},
		{"a lock asks first", tile("lock.door", "locked", "auto"), false, tap{route: guarded}},
		{
			"an idle timer starts",
			tile("timer.tea", "idle", "auto"),
			false,
			tap{route: service, service: "timer.start"},
		},
		{
			"a running timer pauses",
			tile("timer.tea", "active", "auto"),
			false,
			tap{route: service, service: "timer.pause"},
		},
		{"a sensor opens its card", tile("sensor.temp", "21", "auto"), false, tap{route: card}},
		{"a custom action", scripted, false, tap{route: custom, service: "script.turn_on"}},
	}
	for _, tt := range tests {
		if got := routeOf(tt.t, tt.hold); got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestActiveFollowsHomeAssistant(t *testing.T) {
	for _, tt := range []struct {
		t    tessera.Tile
		want bool
	}{
		{tile("light.lamp", "on", ""), true},
		{tile("light.lamp", "off", ""), false},
		{tile("cover.blind", "closed", ""), false},
		{tile("lock.door", "locked", ""), false},
		{tile("scene.evening", "2026-10-08T12:00:00", ""), true},
		{tile("button.bell", "unknown", ""), true},
		{tile("switch.fan", "unavailable", ""), false},
		{tile("vacuum.robot", "docked", ""), false},
	} {
		if got := active(tt.t); got != tt.want {
			t.Errorf("active(%s %s) = %v", tt.t.Entity, tt.t.State, got)
		}
	}
	if !available(tile("scene.evening", "unknown", "")) ||
		available(tile("light.lamp", "unknown", "")) {
		t.Error("only scenes and buttons are pressable while unknown")
	}
}
