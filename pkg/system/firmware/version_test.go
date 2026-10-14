package firmware

import "testing"

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		offered, running string
		want             bool
	}{
		{"0.0.3", "0.0.2", true},
		{"0.0.2", "0.0.3", false},
		{"0.1.0", "0.0.9", true},
		{"0.0.2", "0.0.2", false},
		{"", "0.0.2", false},
		{"0.0.2", "0.0.2-dev.1791769487_08dfcf8", true},
		{"0.0.2-dev.1791769487_08dfcf8", "0.0.2", false},
		{"0.0.2-dev.1791769487_08dfcf8", "0.0.3-dev.1791973755_dirty", false},
		{"0.0.3-dev.1791973755_abc", "0.0.3-dev.1791769487_08dfcf8", true},
		{"0.0.3-dev.9_a", "0.0.3-dev.10_a", false},
		{"v1.2.3", "1.2.2", true},
		{"1.1", "1.0.9", true},
		{"nightly", "1.0", true},
	} {
		if got := Newer(c.offered, c.running); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.offered, c.running, got, c.want)
		}
	}
}
