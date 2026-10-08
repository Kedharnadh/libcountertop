package noise

import (
	"math"
	"testing"
)

const rate = 48000

var bands = []float64{750, 1500, 3000, 6000, 12000}

const (
	blockSize = 4096
	blocks    = 200
)

func TestSlopes(t *testing.T) {
	for _, tc := range []struct {
		name string
		want float64
	}{
		{NameWhite, 0},
		{NamePink, -3},
		{NameBrown, -6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := slope(mixSeeded(rate, 1, tc.name))
			t.Logf("%s: %.2f dB/octave", tc.name, got)

			if math.Abs(got-tc.want) > 0.8 {
				t.Errorf("%s falls %.2f dB an octave, want %.0f", tc.name, got, tc.want)
			}
		})
	}
}

func TestLevelsMatch(t *testing.T) {
	for _, name := range Names() {
		s := byName[name]
		rms, peak := levels(mixSeeded(rate, 2, name), blockSize*blocks)

		t.Logf("%-9s rms %.4f (%5.1f dBFS)  peak %.3f  crest %.1f  RMS %.5f  Peak %.5f",
			name, rms, 20*math.Log10(float64(rms)), peak, peak/rms,
			s.RMS*rms/Level, s.Peak*peak/Loudest)

		got, want := rms, float32(Level)
		if s.Peak > 0 {
			got, want = peak, Loudest
		}
		if off := 20 * math.Log10(float64(got/want)); math.Abs(off) > 1 {
			t.Errorf("%s runs %.1f dB off where it should sit", name, off)
		}
		if peak >= 1 {
			t.Errorf("%s peaks at %.3f, which is clipped", name, peak)
		}
	}
}

func TestResonatorGainDoesNotDependOnPitch(t *testing.T) {
	g := &Gen{Rate: rate, rand: newRand(1)}

	for _, decay := range []float32{0.002, 0.02} {
		var low, high float32 = 1, 0

		for _, freq := range []float32{400, 1600, 6400} {
			r := newReson(freq, decay, rate)

			var sum float64
			const n = 2 * rate
			for range n {
				y := r.run(g.white())
				sum += float64(y) * float64(y)
			}

			rms := float32(math.Sqrt(sum / n))
			low, high = min(low, rms), max(high, rms)
		}

		if spread := 20 * math.Log10(float64(high/low)); spread > 1.5 {
			t.Errorf("ringing for %gs, gain ranges over %.1f dB across pitch", decay, spread)
		}
	}
}

func TestMixHoldsTheLevel(t *testing.T) {
	rms, peak := levels(mixSeeded(rate, 3, NameWind, NameCrickets), blockSize*blocks)
	t.Logf("wind+crickets: rms %.4f, peak %.3f", rms, peak)

	if off := 20 * math.Log10(float64(rms/Level)); math.Abs(off) > 1.5 {
		t.Errorf("a mix runs %.1f dB off the level one sound plays at", off)
	}
	if peak >= 1 {
		t.Errorf("a mix peaks at %.3f, which is clipped", peak)
	}
}

func TestUnknownSoundsPlayNothing(t *testing.T) {
	if New("Bagpipes", rate) != nil {
		t.Error("played a sound this build does not have")
	}
	if Mix(rate) != nil {
		t.Error("played nothing at all as though it were something")
	}
	if Mix(rate, "Bagpipes", NamePink) == nil {
		t.Error("one unknown name in a mix threw the mix away")
	}
}

func slope(fill Fill) float64 {
	power := make([]float64, len(bands))
	buf := make([]float32, blockSize)

	for range blocks {
		fill(buf)
		for i, f := range bands {
			power[i] += goertzel(buf, f)
		}
	}

	var sx, sy, sxx, sxy float64
	for i, p := range power {
		x, y := float64(i), 10*math.Log10(p/blocks)
		sx, sy, sxx, sxy = sx+x, sy+y, sxx+x*x, sxy+x*y
	}
	n := float64(len(power))
	return (n*sxy - sx*sy) / (n*sxx - sx*sx)
}

func goertzel(samples []float32, freq float64) float64 {
	k := math.Round(freq / rate * float64(len(samples)))
	w := 2 * math.Pi * k / float64(len(samples))
	coeff := 2 * math.Cos(w)

	var s1, s2 float64
	for _, v := range samples {
		s := float64(v) + coeff*s1 - s2
		s2, s1 = s1, s
	}
	return s1*s1 + s2*s2 - coeff*s1*s2
}

func levels(fill Fill, n int) (rms, peak float32) {
	buf := make([]float32, blockSize)
	var sum float64

	for range n / blockSize {
		fill(buf)
		for _, v := range buf {
			sum += float64(v) * float64(v)
			peak = max(peak, float32(math.Abs(float64(v))))
		}
	}
	return float32(math.Sqrt(sum / float64(n))), peak
}
