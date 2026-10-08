package geometry

import (
	"math"
	"testing"
)

func TestDensityFollowsTheDiagonal(t *testing.T) {
	if got := DPI(1200, 1920, 10.03); math.Abs(got-225.7) > 0.5 {
		t.Errorf("a 10.03 inch 1200x1920 panel is %.1f dpi", got)
	}
	if DPI(1200, 1920, 0) != 0 {
		t.Error("an unknown diagonal has a density")
	}
}
