package upsample

import (
	"math"
	"testing"
)

const (
	rate     = 48000
	channels = 2
	factor   = rate / VoiceRate
)

func at(samples []int16, hz float64) float64 {
	var re, im float64
	for i := 0; i < len(samples); i += channels {
		p := 2 * math.Pi * hz * float64(i/channels) / rate
		re += float64(samples[i]) * math.Cos(p)
		im += float64(samples[i]) * math.Sin(p)
	}
	n := float64(len(samples) / channels)
	return math.Hypot(re, im) / n
}

func voice(hz float64, ms int) []int16 {
	out := make([]int16, VoiceRate*ms/1000)
	for i := range out {
		out[i] = int16(0.5 * math.MaxInt16 * math.Sin(2*math.Pi*hz*float64(i)/VoiceRate))
	}
	return out
}

func run(r Resampler, mono []int16) []int16 {
	return r.Run(mono, make([]int16, 0, len(mono)*factor*channels))
}

func made(k Kind) Resampler {
	r, _ := New(k, rate)
	return r
}

func TestTheFilterRejectsTheImageThatHoldingLeaves(t *testing.T) {
	const hz = 3000
	mono := voice(hz, 200)

	held := run(made(Hold), mono)
	filtered := run(made(Sinc), mono)

	image := rate/factor - hz

	heldRatio := at(held, float64(image)) / at(held, hz)
	sincRatio := at(filtered, float64(image)) / at(filtered, hz)

	if sincRatio >= heldRatio {
		t.Fatalf(
			"the filter left as much image as holding: %.4f against %.4f",
			sincRatio,
			heldRatio,
		)
	}
	if sincRatio > 0.01 {
		t.Errorf("the image is %.4f of the tone, want under 0.01", sincRatio)
	}
}

func TestResamplersRankByImageRejection(t *testing.T) {
	const hz = 3000
	mono := voice(hz, 300)

	image := map[Kind]float64{}
	for _, k := range Kinds() {
		out := run(made(k), mono)[512:]
		image[k] = at(out, 2*VoiceRate-hz)
	}

	if image[Sinc] >= image[Linear] {
		t.Errorf(
			"the filter leaves %.2f of the image, linear only %.2f",
			image[Sinc],
			image[Linear],
		)
	}
	if image[Linear] >= image[Hold] {
		t.Errorf("linear leaves %.2f of the image, holding only %.2f", image[Linear], image[Hold])
	}
}

func TestEveryResamplerKeepsTheTone(t *testing.T) {
	const hz = 1000
	mono := voice(hz, 200)

	for _, k := range Kinds() {
		t.Run(string(k), func(t *testing.T) {
			r, settled := New(k, rate)
			if settled != k {
				t.Fatalf("asked for %s and got %s", k, settled)
			}

			out := run(r, mono)
			if want := len(mono) * factor * channels; len(out) != want {
				t.Fatalf("%d samples out, want %d", len(out), want)
			}
			if got := at(out, hz); got < 0.2*math.MaxInt16 {
				t.Errorf("the tone came out at %.0f, which is most of the way to silence", got)
			}
		})
	}
}

func TestSilenceStaysSilent(t *testing.T) {
	for i, v := range run(made(Sinc), make([]int16, 512)) {
		if v != 0 {
			t.Fatalf("sample %d of silence is %d", i, v)
		}
	}
}

func TestChunksComeOutTheSameAsOnePiece(t *testing.T) {
	mono := voice(1000, 100)

	whole := run(made(Sinc), mono)

	piecewise := make([]int16, 0, len(whole))
	chunked := made(Sinc)
	for i := 0; i < len(mono); i += 137 {
		piecewise = chunked.Run(mono[i:min(i+137, len(mono))], piecewise)
	}

	if len(piecewise) != len(whole) {
		t.Fatalf("chunked gave %d samples, whole gave %d", len(piecewise), len(whole))
	}
	for i := range whole {
		if piecewise[i] != whole[i] {
			t.Fatalf(
				"chunked and whole differ at sample %d: %d against %d",
				i,
				piecewise[i],
				whole[i],
			)
		}
	}
}

func TestResetDropsTheTail(t *testing.T) {
	r := made(Sinc)
	run(r, voice(1000, 50))
	r.Reset()

	quiet := make([]int16, 160)
	got, want := run(r, quiet), run(made(Sinc), quiet)
	if len(got) != len(want) {
		t.Fatalf("%d samples against %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("a reset resampler still carried the last utterance at sample %d", i)
		}
	}
}

func TestVoiceLandsOnBothChannels(t *testing.T) {
	out := run(made(Sinc), voice(1000, 20))

	for i := 0; i+1 < len(out); i += channels {
		if out[i] != out[i+1] {
			t.Fatalf("channels differ at frame %d: %d and %d", i/channels, out[i], out[i+1])
		}
	}
}

func TestAnUnknownResamplerFallsBackToTheFilter(t *testing.T) {
	r, settled := New(Kind("bogus"), rate)

	if settled != Sinc {
		t.Errorf("settled on %s, want %s", settled, Sinc)
	}
	if r == nil {
		t.Fatal("no resampler came back")
	}
}

func TestTheSweepIsAVoiceSignalThatDoesNotClick(t *testing.T) {
	got := Sweep()

	if want := VoiceRate * sweepMs / 1000; len(got) != want {
		t.Fatalf("the sweep is %d samples, want %d", len(got), want)
	}

	var peak int16
	for _, v := range got {
		peak = max(peak, v)
	}
	if quiet := peak / 100; got[0] > quiet || got[len(got)-1] > quiet {
		t.Errorf(
			"the sweep starts at %d and ends at %d against a peak of %d",
			got[0],
			got[len(got)-1],
			peak,
		)
	}
}

func BenchmarkResample(b *testing.B) {
	mono := voice(1000, 1000)

	for _, k := range Kinds() {
		b.Run(string(k), func(b *testing.B) {
			r := made(k)
			out := make([]int16, 0, len(mono)*factor*channels)

			for b.Loop() {
				r.Run(mono, out[:0])
			}
			b.SetBytes(int64(len(mono) * 2))
		})
	}
}
