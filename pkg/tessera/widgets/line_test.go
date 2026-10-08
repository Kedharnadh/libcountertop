package widgets

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func dressed(entity, state string, attrs, extras map[string]any) tessera.Tile {
	t := tile(entity, state, "auto")
	t.Attributes, _ = json.Marshal(attrs)
	t.Extra, _ = json.Marshal(extras)
	return t
}

func TestEachDomainSaysItsOwnLine(t *testing.T) {
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.Local)
	today := float64(time.Date(2026, 10, 8, 14, 32, 0, 0, time.Local).Unix())
	yesterday := float64(time.Date(2026, 10, 7, 9, 5, 0, 0, time.Local).Unix())
	earlier := float64(time.Date(2026, 9, 13, 9, 5, 0, 0, time.Local).Unix())
	l := tessera.Layout{Clock24: true}
	for _, tt := range []struct {
		name string
		t    tessera.Tile
		want string
	}{
		{"a dimmed light", dressed("light.lamp", "on", map[string]any{"brightness": 128.0}, nil), "50%"},
		{"a light running an effect", dressed("light.lamp", "on", map[string]any{"brightness": 128.0, "effect": "Rainbow"}, nil), "Rainbow"},
		{"a solid effect is no effect", dressed("light.lamp", "on", map[string]any{"effect": "Solid"}, nil), "On"},
		{"a fan says on", dressed("fan.ceiling", "on", map[string]any{"percentage": 40.0}, nil), "On"},
		{"a script ran today", dressed("script.night", "off", nil, map[string]any{"last": today}), "Last 14:32"},
		{"a scene ran yesterday", dressed("scene.evening", "off", nil, map[string]any{"last": yesterday}), "Yesterday 09:05"},
		{"a button pressed long ago", dressed("button.bell", "unknown", nil, map[string]any{"last": earlier}), "Last 13 Sep"},
		{"a script never ran", dressed("script.night", "off", nil, nil), "Never run"},
		{"a running script", dressed("script.night", "on", nil, nil), "Running..."},
		{"a door", dressed("binary_sensor.front", "on", map[string]any{"device_class": "door"}, nil), "Open"},
		{"a leak sensor", dressed("binary_sensor.sink", "off", map[string]any{"device_class": "moisture"}, nil), "Dry"},
		{"a plain binary sensor", dressed("binary_sensor.thing", "on", nil, nil), "On"},
		{"a paused timer", dressed("timer.tea", "paused", nil, map[string]any{"rem": "0:02:05"}), "Paused 2:05"},
		{"an idle timer", dressed("timer.tea", "idle", nil, map[string]any{"dur": "1:00:00"}), "1:00:00"},
		{"an idle timer of no length", dressed("timer.tea", "idle", nil, map[string]any{"dur": "0:00:00"}), "0:00"},
		{"a timer without a duration", dressed("timer.tea", "idle", nil, nil), "Off"},
		{"a running timer", dressed("timer.tea", "active", nil, map[string]any{"dur": "0:05:00", "end": float64(now.Unix() + 75)}), "1:15"},
		{"a lock in its own words", dressed("lock.door", "unlocked", nil, nil), "Unlocked"},
		{"a cover in HA's words", dressed("cover.blind", "open", nil, map[string]any{"w": "Open"}), "Open"},
	} {
		if got := domainLine(tt.t, l, now, false); got != tt.want {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
	if got := lastRun(yesterday, now, false, true); got != "Yest. 9:05 AM" {
		t.Errorf("compact yesterday on a 12-hour clock: %q", got)
	}
}

func TestIconsFollowTheState(t *testing.T) {
	if iconFor(tile("light.lamp", "off", ""), false) != 0xF0E4F ||
		iconFor(tile("light.lamp", "on", ""), false) != 0xF0335 {
		t.Error("a light's bulb does not follow its state")
	}
	if iconFor(tile("lock.door", "locked", ""), true) != 0xF0FC6 {
		t.Error("a lock asking for its second tap does not show the open lock")
	}
	chosen := tile("light.lamp", "off", "")
	chosen.Icon = "F0425"
	if iconFor(chosen, false) != 0xF0425 {
		t.Error("a chosen icon does not stay")
	}
}

func TestALampPaintsItsOwnColour(t *testing.T) {
	pal := theme.Default()
	if got := lampColor(0, 5, pal); got != pal.Roles.Warm {
		t.Errorf("a white bulb is %s, want the theme's warm", got)
	}
	if got := lampColor(0, 100, pal); got != theme.RGB(0xff0000) {
		t.Errorf("a red bulb is %s", got)
	}
	if got := lampColor(240, 10, pal); got != theme.RGB(0x9999ff) {
		t.Errorf("a pale blue bulb is %s, want saturation held at 40", got)
	}
}
