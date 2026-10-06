package ducking

import (
	"math"
	"testing"
)

func TestSavedDepthAndLegacyLevel(t *testing.T) {
	if Gain(LegacyDB) != float32(0.2) {
		t.Fatal(Gain(LegacyDB))
	}
	if got := Gain(-15); math.Abs(float64(got)-math.Pow(10, -15.0/20)) > 1e-7 {
		t.Fatal(got)
	}
	if Gain(-20) != float32(0.1) {
		t.Fatal(Gain(-20))
	}
	if Valid(1) || Valid(math.NaN()) || Valid(math.Inf(-1)) {
		t.Fatal("invalid attenuation accepted")
	}
}
