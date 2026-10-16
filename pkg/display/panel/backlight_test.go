package panel

import "testing"

func TestBacklightFollowsLightness(t *testing.T) {
	for _, tc := range []struct{ percent, want int }{
		{0, 0},
		{2, 1},
		{8, 2},
		{50, 47},
		{92, 206},
		{100, 255},
		{140, 255},
	} {
		if got := perceived(tc.percent, 255); got != tc.want {
			t.Errorf("perceived(%d, 255) = %d, want %d", tc.percent, got, tc.want)
		}
	}
	last := 0
	for p := 1; p <= 100; p++ {
		got := perceived(p, 255)
		if got < last {
			t.Fatalf("perceived(%d) = %d fell below %d", p, got, last)
		}
		last = got
	}
}
