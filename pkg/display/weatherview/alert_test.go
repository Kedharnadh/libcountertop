package weatherview

import "testing"

func TestHazardOf(t *testing.T) {
	for event, want := range map[string]hazard{
		"Tornado Warning":             hazardTornado,
		"Hurricane Warning":           hazardHurricane,
		"Tropical Storm Watch":        hazardHurricane,
		"Severe Thunderstorm Warning": hazardStorm,
		"Flash Flood Warning":         hazardFlood,
		"Storm Surge Warning":         hazardFlood,
		"Blizzard Warning":            hazardWinter,
		"Ice Storm Warning":           hazardWinter,
		"Extreme Heat Warning":        hazardHeat,
		"Red Flag Warning":            hazardFire,
		"High Wind Warning":           hazardWind,
		"Dense Fog Advisory":          hazardFog,
		"Civil Danger Warning":        hazardGeneric,
	} {
		if got := hazardOf(event); got != want {
			t.Errorf("hazardOf(%q) = %q, want %q", event, got, want)
		}
	}
}
