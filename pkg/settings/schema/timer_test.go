package schema

import "testing"

func TestTimersDefaultToABar(t *testing.T) {
	if got := DefaultScreen().Timer; got != TimerBar {
		t.Fatalf("default timer style %q, want %q", got, TimerBar)
	}
}

func TestEveryTimerStyleHasItsOwnLabel(t *testing.T) {
	seen := map[string]TimerStyle{}
	for _, s := range TimerStyles() {
		label := s.Label()
		if label == "" || label == string(s) || label == "timerstyle."+string(s) {
			t.Errorf("%s is labeled %q", s, label)
		}
		if other, dup := seen[label]; dup {
			t.Errorf("%s and %s are both labeled %q", other, s, label)
		}
		seen[label] = s
	}
}
