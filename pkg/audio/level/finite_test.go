package level

import (
	"math"
	"testing"
)

func TestLevelerStaysFiniteUnderAdversarialInput(t *testing.T) {
	patterns := []struct {
		name  string
		frame func(i int) int16
	}{
		{"exact zeros", func(int) int16 { return 0 }},
		{"one lsb", func(int) int16 { return 1 }},
		{"dc at full scale", func(int) int16 { return 32767 }},
		{"negative full scale", func(int) int16 { return -32768 }},
		{"alternating extremes", func(i int) int16 {
			if i%2 == 0 {
				return 32767
			}
			return -32768
		}},
		{"full scale square", func(i int) int16 {
			if i/16%2 == 0 {
				return 32767
			}
			return -32768
		}},
	}

	const frames = 6000

	for _, p := range patterns {
		for _, settled := range []bool{false, true} {
			l := newLeveler()
			name := p.name
			if settled {
				settle(l, -57, 200)
				name += " (after a real room)"
			}

			buf := make([]int16, testFrame)
			for k := 0; k < frames; k++ {
				for i := range buf {
					buf[i] = p.frame(k*testFrame + i)
				}
				l.apply(buf, buf)
			}

			for what, v := range map[string]float64{
				"gain":      float64(l.gain),
				"floor":     float64(l.floor),
				"bandFloor": float64(l.bandFloor),
				"level":     float64(math.Float32frombits(l.level.Load())),
			} {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					t.Errorf("%s: %s is %v", name, what, v)
				}
			}

			var loudest int16
			for k := 0; k < 300; k++ {
				say := speech(-30)
				l.apply(say, say)
				for _, v := range say {
					if v > loudest {
						loudest = v
					}
				}
			}
			if loudest == 0 {
				t.Errorf("%s: speech afterwards comes out as pure zeros", name)
			}
			t.Logf("%-38s gain=%7.3f floor=%9.3f speech after=%6d", name, l.gain, l.floor, loudest)
		}
	}
}
