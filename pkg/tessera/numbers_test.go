package tessera

import "testing"

func TestNumbersAreWrittenTheScreensWay(t *testing.T) {
	for _, tt := range []struct {
		n     Numbers
		state string
		want  string
	}{
		{Numbers{}, "1234.5", "1,234.5"},
		{Numbers{}, "123", "123"},
		{Numbers{}, "-1234567", "-1,234,567"},
		{Numbers{Style: 2}, "1234.5", "1.234,5"},
		{Numbers{Style: 3}, "1234.5", "1 234,5"},
		{Numbers{Style: 2, GroupMin: 2}, "1234", "1234"},
		{Numbers{Style: 2, GroupMin: 2}, "12345", "12.345"},
		{Numbers{}, "on", "on"},
		{Numbers{}, "1.", "1."},
		{Numbers{}, "-", "-"},
	} {
		if got := tt.n.Localize(tt.state); got != tt.want {
			t.Errorf("%+v.Localize(%q) = %q, want %q", tt.n, tt.state, got, tt.want)
		}
	}
	if got := (Numbers{Percent: 2}).WithUnit("54", "%"); got != "54 %" {
		t.Errorf("spaced percent = %q", got)
	}
	if got := (Numbers{}).WithUnit("21.5", "°C"); got != "21.5 °C" {
		t.Errorf("unit = %q", got)
	}
	if got := (Numbers{}).WithUnit("18", "°"); got != "18°" {
		t.Errorf("bare degree = %q", got)
	}
}
