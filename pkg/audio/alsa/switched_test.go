package alsa

import "testing"

func TestASwitchTakesTheSameWordsAsAnOnOffEnum(t *testing.T) {
	for _, tc := range []struct {
		item string
		want uint32
		ok   bool
	}{
		{"On", 1, true},
		{"off", 0, true},
		{"OFF", 0, true},
		{"Auto", 0, false},
	} {
		got, ok := switched(tc.item)
		if got != tc.want || ok != tc.ok {
			t.Errorf("switched(%q) = %d, %v; want %d, %v", tc.item, got, ok, tc.want, tc.ok)
		}
	}
}
