package resample

import (
	"testing"
)

func TestHeadroomKeepsItsLevel(t *testing.T) {
	r := NewRational(16000, 48000, 1)
	r.Headroom(0.9)
	phases, _ := r.Ratio()
	for p := range phases {
		var sum float32
		for k := range ratioTaps {
			sum += r.filter[p+k*phases]
		}
		if diff := sum - 0.9; diff > 0.001 || diff < -0.001 {
			t.Errorf("phase %d sums to %.5f", p, sum)
		}
	}
}
