package widgets

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/say"
)

func TestEnergyRows(t *testing.T) {
	other, untracked := say.T("tessera.energy.other"), say.T("tessera.energy.untracked")
	devices := []map[string]any{
		{"n": "Oven", "w": 2000.0},
		{"n": "Fridge", "w": 100.0},
		{"n": "Lamp", "w": 20.0},
	}
	cases := []struct {
		name   string
		other  float64
		home   float64
		places int
		want   []energyRow
	}{
		{
			name: "all fit with other and untracked", other: 50, home: 2500, places: 5,
			want: []energyRow{
				{name: "Oven", watts: 2000},
				{name: "Fridge", watts: 100},
				{name: "Lamp", watts: 20},
				{name: other, watts: 50, rest: true},
				{name: untracked, watts: 330, rest: true},
			},
		},
		{
			name: "smallest fold into other", home: 2500, places: 3,
			want: []energyRow{
				{name: "Oven", watts: 2000},
				{name: other, watts: 120, rest: true},
				{name: untracked, watts: 380, rest: true},
			},
		},
		{
			name: "no untracked under a watt", home: 2120.5, places: 4,
			want: []energyRow{
				{name: "Oven", watts: 2000},
				{name: "Fridge", watts: 100},
				{name: "Lamp", watts: 20},
			},
		},
	}
	for _, c := range cases {
		got := energyRows(devices, c.other, c.home, c.places)
		if len(got) != len(c.want) {
			t.Fatalf("%s: energyRows gave %v, want %v", c.name, got, c.want)
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("%s: row %d = %v, want %v", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestBatteryGlyph(t *testing.T) {
	cases := []struct {
		soc   float64
		known bool
		want  rune
	}{
		{0, false, 0xF0079},
		{2, true, 0xF008E},
		{50, true, 0xF007E},
		{97, true, 0xF0079},
	}
	for _, c := range cases {
		if got := batteryGlyph(c.soc, c.known); got != c.want {
			t.Errorf("batteryGlyph(%v, %v) = %X, want %X", c.soc, c.known, got, c.want)
		}
	}
}
