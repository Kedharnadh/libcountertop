package schema

import "testing"

func TestAlertsDefaultToABanner(t *testing.T) {
	if got := DefaultScreen().Alert; got != AlertBanner {
		t.Fatalf("default alert style %q, want %q", got, AlertBanner)
	}
}

func TestEveryAlertStyleHasItsOwnLabel(t *testing.T) {
	seen := map[string]AlertStyle{}
	for _, s := range AlertStyles() {
		label := s.Label()
		if label == "" || label == string(s) || label == "alertstyle."+string(s) {
			t.Errorf("%s is labeled %q", s, label)
		}
		if other, dup := seen[label]; dup {
			t.Errorf("%s and %s are both labeled %q", other, s, label)
		}
		seen[label] = s
	}
}
