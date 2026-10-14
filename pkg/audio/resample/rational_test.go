package resample

import (
	"math"
	"math/rand"
	"testing"
)

func TestTheRatioReduces(t *testing.T) {
	for _, c := range []struct {
		from, to int
		up, down int
	}{
		{44100, 48000, 160, 147},
		{48000, 44100, 147, 160},
		{16000, 48000, 3, 1},
		{48000, 48000, 1, 1},
		{22050, 48000, 320, 147},
	} {
		up, down := NewRational(c.from, c.to, 2).Ratio()
		if up != c.up || down != c.down {
			t.Errorf("%d to %d reduced to %d:%d, want %d:%d", c.from, c.to, up, down, c.up, c.down)
		}
	}
}

func TestMatchingRatesPassStraightThrough(t *testing.T) {
	in := sine(48000, 1000, 480, 2)

	got := NewRational(48000, 48000, 2).Run(in)
	if len(got) != len(in) {
		t.Fatalf("%d samples out of %d in", len(got), len(in))
	}
	for i := range in {
		if got[i] != in[i] {
			t.Fatalf("sample %d came back as %d, want %d", i, got[i], in[i])
		}
	}
}

func TestASecondComesOutASecond(t *testing.T) {
	const frames = 44100

	got := NewRational(44100, 48000, 2).Run(sine(44100, 1000, frames, 2))

	want := frames * 160 / 147

	if out := len(got) / 2; out != want {
		t.Errorf("%d frames out of %d, want %d", out, frames, want)
	}
}

func TestNothingIsSwallowedAtTheStart(t *testing.T) {
	r := NewRational(16000, 48000, 1)

	got := r.Run([]int16{10000})

	if len(got) != 3 {
		t.Fatalf("%d samples out of one, want three", len(got))
	}
	for i, v := range got {
		if v == 0 {
			t.Errorf("sample %d of the first input came out silent", i)
		}
	}
}

func TestAToneKeepsItsPitchAndItsLevel(t *testing.T) {
	const (
		hz   = 1000
		peak = 10000
	)

	in := sine(44100, hz, 22050, 2)
	got := NewRational(44100, 48000, 2).Run(in)

	got = got[48*2:]

	amp := level(got, 0, 2, 48000, hz)
	if math.Abs(amp-peak)/peak > 0.02 {
		t.Errorf("the tone came out at %.0f, want about %d", amp, peak)
	}

	if image := level(got, 0, 2, 48000, 44100-hz); image > peak*0.01 {
		t.Errorf("an image at %d Hz came out at %.0f", 44100-hz, image)
	}

	rest := math.Sqrt(math.Max(power(got, 0, 2)-amp*amp/2, 0))
	if rest > peak*0.02 {
		t.Errorf("%.0f rms of everything that is not the tone, against a tone of %d", rest, peak)
	}
}

func TestDownsamplingDoesNotFoldHighTonesBack(t *testing.T) {
	const peak = 10000
	for _, c := range []struct {
		from, to int
		hz       float64
	}{
		{22050, 16000, 9000},
		{22050, 16000, 10000},
		{48000, 16000, 12000},
		{44100, 16000, 20000},
	} {
		in := sine(c.from, c.hz, c.from, 1)
		got := NewRational(c.from, c.to, 1).Run(in)[c.to/10:]
		if rms := math.Sqrt(power(got, 0, 1)); rms > peak*0.01 {
			t.Errorf("%.0f Hz from %d to %d came out at %.0f rms, want under 1%%",
				c.hz, c.from, c.to, rms)
		}
	}
}

func TestDownsamplingKeepsSpeech(t *testing.T) {
	const peak = 10000
	for _, hz := range []float64{300, 1000, 3000, 6000} {
		in := sine(22050, hz, 22050, 1)
		got := NewRational(22050, 16000, 1).Run(in)[1600:]
		if amp := level(got, 0, 1, 16000, hz); math.Abs(amp-peak)/peak > 0.05 {
			t.Errorf("%.0f Hz came out at %.0f, want about %d", hz, amp, peak)
		}
	}
}

func TestBuffersJoinUpWhereTheyMeet(t *testing.T) {
	in := sine(44100, 1000, 8192, 2)

	whole := NewRational(44100, 48000, 2).Run(append([]int16(nil), in...))

	chunked := &Rational{}
	*chunked = *NewRational(44100, 48000, 2)

	var pieces []int16
	rng := rand.New(rand.NewSource(1))
	for at := 0; at < len(in); {
		n := (1 + rng.Intn(128)) * 2
		n = min(n, len(in)-at)

		pieces = append(pieces, chunked.Run(in[at:at+n])...)
		at += n
	}

	if len(pieces) != len(whole) {
		t.Fatalf("%d samples in pieces against %d in one go", len(pieces), len(whole))
	}
	for i := range whole {
		if pieces[i] != whole[i] {
			t.Fatalf("sample %d is %d in pieces and %d in one go", i, pieces[i], whole[i])
		}
	}
}

func TestResetStartsOver(t *testing.T) {
	in := sine(44100, 1000, 2048, 2)

	r := NewRational(44100, 48000, 2)
	first := append([]int16(nil), r.Run(in)...)

	r.Reset()
	second := r.Run(in)

	if len(first) != len(second) {
		t.Fatalf("%d samples then %d from the same input", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("sample %d is %d after a reset, want %d", i, second[i], first[i])
		}
	}
}

func TestTheEarsStaySeparate(t *testing.T) {
	const frames = 4096

	in := make([]int16, frames*2)
	for i := range frames {
		in[i*2] = int16(10000 * math.Sin(2*math.Pi*1000*float64(i)/44100))
	}

	got := NewRational(44100, 48000, 2).Run(in)

	if left := level(got[96:], 0, 2, 48000, 1000); left < 9000 {
		t.Errorf("the tone came out of the left at %.0f, want about 10000", left)
	}
	if right := math.Sqrt(power(got[96:], 1, 2)); right > 100 {
		t.Errorf("%.0f rms leaked into the silent ear", right)
	}
}

func TestTooLittleToWorkWith(t *testing.T) {
	r := NewRational(44100, 48000, 2)

	if got := r.Run(nil); len(got) != 0 {
		t.Errorf("%d samples out of nothing", len(got))
	}

	in := sine(44100, 1000, 64, 2)
	var out []int16
	for at := 0; at < len(in); at += 2 {
		out = append(out, r.Run(in[at:at+2])...)
	}

	if want := (64 - ratioTaps) * 160 / 147; len(out)/2 < want {
		t.Errorf("%d frames from 64 fed one at a time, want at least %d", len(out)/2, want)
	}
}

func TestFullScaleSaturates(t *testing.T) {
	in := make([]int16, 4096)
	for i := range len(in) / 2 {
		v := int16(math.MaxInt16)
		if (i/16)%2 == 1 {
			v = math.MinInt16
		}
		in[i*2], in[i*2+1] = v, v
	}

	got := NewRational(44100, 48000, 2).Run(in)

	var pinned int
	for _, v := range got {
		if v == math.MaxInt16 || v == math.MinInt16 {
			pinned++
		}
	}
	if pinned == 0 {
		t.Fatal("a full scale square came back with nothing at full scale")
	}
}

func TestClipSaturates(t *testing.T) {
	for _, c := range []struct {
		in   float32
		want int16
	}{
		{0, 0},
		{32766.6, 32766},
		{32767, 32767},
		{40000, math.MaxInt16},
		{1e9, math.MaxInt16},
		{-32768, math.MinInt16},
		{-40000, math.MinInt16},
		{-1e9, math.MinInt16},
	} {
		if got := clip(c.in); got != c.want {
			t.Errorf("clip(%v) is %d, want %d", c.in, got, c.want)
		}
	}
}

func sine(rate int, hz float64, frames, channels int) []int16 {
	out := make([]int16, frames*channels)
	for i := range frames {
		v := int16(10000 * math.Sin(2*math.Pi*hz*float64(i)/float64(rate)))
		for c := range channels {
			out[i*channels+c] = v
		}
	}
	return out
}

func level(x []int16, ch, channels, rate int, hz float64) float64 {
	n := len(x) / channels

	var re, im float64
	for i := range n {
		w := 2 * math.Pi * hz * float64(i) / float64(rate)
		v := float64(x[i*channels+ch])

		re += v * math.Cos(w)
		im -= v * math.Sin(w)
	}
	return 2 * math.Hypot(re, im) / float64(n)
}

func power(x []int16, ch, channels int) float64 {
	n := len(x) / channels

	var sum float64
	for i := range n {
		v := float64(x[i*channels+ch])
		sum += v * v
	}
	return sum / float64(n)
}
