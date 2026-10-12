package widgets

import (
	"encoding/json"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func TestHourLabel(t *testing.T) {
	cases := []struct {
		hour    map[string]any
		clock24 bool
		want    string
	}{
		{map[string]any{"t": "14:00"}, false, "14:00"},
		{map[string]any{"o": 0.0}, false, "12 AM"},
		{map[string]any{"o": 15.0}, false, "3 PM"},
		{map[string]any{"o": 36.0}, false, "12 PM"},
		{map[string]any{"o": 31.0}, true, "07:00"},
		{map[string]any{}, true, ""},
	}
	for _, c := range cases {
		if got := hourLabel(c.hour, c.clock24); got != c.want {
			t.Errorf("hourLabel(%v, %v) = %q, want %q", c.hour, c.clock24, got, c.want)
		}
	}
}

func TestBearingAndWeekly(t *testing.T) {
	tile := func(attrs, extra string) tessera.Tile {
		return tessera.Tile{Attributes: json.RawMessage(attrs), Extra: json.RawMessage(extra)}
	}
	for attrs, want := range map[string]string{
		`{"wind_bearing": 0}`:    "N",
		`{"wind_bearing": 225}`:  "SW",
		`{"wind_bearing": 350}`:  "N",
		`{"wind_bearing": "NE"}`: "NE",
		`{}`:                     "",
	} {
		if got := bearing(tile(attrs, `{}`)); got != want {
			t.Errorf("bearing(%s) = %q, want %q", attrs, got, want)
		}
	}
	for extra, want := range map[string]bool{
		`{"days": [{"d": "Mon", "w": 1}]}`:                    true,
		`{"hours": [{"o": 5, "c": "sunny"}]}`:                 true,
		`{"days": [{"d": "Mon"}], "hours": [{"t": "14:00"}]}`: false,
	} {
		if got := weekly(tile(`{}`, extra)); got != want {
			t.Errorf("weekly(%s) = %v, want %v", extra, got, want)
		}
	}
}
