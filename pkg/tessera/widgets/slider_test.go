package widgets

import "testing"

func TestASliderSendsWhatEachDomainTakes(t *testing.T) {
	for _, tt := range []struct {
		name   string
		attrs  map[string]any
		entity string
		state  string
		value  float64
		action string
		key    string
		sent   string
		wish   string
	}{
		{"a dim light floors at 1%", nil, "light.lamp", "on", 0, "light.turn_on", "brightness", "3", ""},
		{"an off light turns on", nil, "light.lamp", "off", 0.5, "light.turn_on", "brightness", "128", "on"},
		{"a fan", nil, "fan.ceiling", "on", 0.4, "fan.set_percentage", "percentage", "40", ""},
		{"a blind's bar is its closed part", nil, "cover.blind", "open", 0.25, "cover.set_cover_position", "position", "75", ""},
		{"a number snaps to its step", map[string]any{"min": 70.0, "max": 90.0, "step": 2.0}, "input_number.pool", "70", 0.52, "input_number.set_value", "value", "80", ""},
	} {
		act, data, wish := levelAction(dressed(tt.entity, tt.state, tt.attrs, nil), tt.value)
		if act != tt.action || data[tt.key] != tt.sent || wish != tt.wish {
			t.Errorf("%s: %s %v wish %q", tt.name, act, data, wish)
		}
	}
}

func TestASliderReadsEachDomain(t *testing.T) {
	if v, _ := level(
		dressed("light.lamp", "on", map[string]any{"brightness": 51.0}, nil),
	); v != 0.2 {
		t.Errorf("light %v", v)
	}
	if v, _ := level(
		dressed("cover.blind", "open", map[string]any{"current_position": 30.0}, nil),
	); v != 0.7 {
		t.Errorf("cover %v", v)
	}
	if v, _ := level(
		dressed("number.n", "75", map[string]any{"min": 50.0, "max": 100.0}, nil),
	); v != 0.5 {
		t.Errorf("number %v", v)
	}
}
