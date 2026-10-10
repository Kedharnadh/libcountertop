package hci

import "testing"

func TestTheScanWindowIsAShareOfTheInterval(t *testing.T) {
	for _, tc := range []struct {
		percent int
		want    uint16
	}{
		{9, 28},
		{5, 16},
		{2, 6},
		{1, scanWindowLeast},
		{0, scanWindowLeast},
		{100, scanInterval},
		{150, scanInterval},
	} {
		if got := (Scanning{Percent: tc.percent}).window(); got != tc.want {
			t.Errorf("%d%% gives a window of %d, want %d", tc.percent, got, tc.want)
		}
	}
}
