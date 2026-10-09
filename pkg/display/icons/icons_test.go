package icons

import "testing"

func TestNamed(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want rune
	}{
		{"doorbell", 0xF12E6},
		{"mdi:doorbell", 0xF12E6},
		{" MDI:Alert-Outline ", 0xF002A},
		{"F0335", 0xF0335},
		{"f0335", 0xF0335},
		{"no-such-icon", 0},
		{"", 0},
	} {
		if got := Named(tt.in); got != tt.want {
			t.Errorf("Named(%q) = %X, want %X", tt.in, got, tt.want)
		}
	}
}
