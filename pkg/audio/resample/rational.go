package resample

import (
	"math"
)

const ratioTaps = 16

type Rational struct {
	up   int
	down int

	channels int

	filter []float32

	at int

	tail []int16

	skew, owed float64

	clipped uint64
}

func NewRational(from, to, channels int) *Rational {
	if channels < 1 {
		channels = 1
	}

	g := gcd(from, to)
	r := &Rational{up: to / g, down: from / g, channels: channels}

	if from != to {
		r.filter = lowpass(r.up, ratioTaps)
	}
	r.Reset()
	return r
}

const skewGrid = 160

func NewSkewable(from, to, channels int) *Rational {
	if from != to {
		return NewRational(from, to, channels)
	}
	r := &Rational{
		up:       skewGrid,
		down:     skewGrid,
		channels: max(channels, 1),
		filter:   lowpass(skewGrid, ratioTaps),
	}
	r.Reset()
	return r
}

func (r *Rational) Headroom(g float32) {
	for i := range r.filter {
		r.filter[i] *= g
	}
}

func (r *Rational) Ratio() (up, down int) { return r.up, r.down }

func (r *Rational) Run(in []int16) []int16 {
	if r.filter == nil || len(in) == 0 {
		return in
	}

	ch := r.channels
	buf := append(r.tail, in...)
	frames := len(buf) / ch

	out := make([]int16, 0, (len(in)*r.up/r.down)+ch)

	for {
		idx := r.at / r.up
		if idx >= frames {
			break
		}
		if idx < ratioTaps-1 {
			r.at += r.down
			continue
		}

		phase := r.at % r.up
		for c := range ch {
			var sum float32
			for k := range ratioTaps {
				sum += r.filter[phase+k*r.up] * float32(buf[(idx-k)*ch+c])
			}
			if sum > 32767 || sum < -32768 {
				r.clipped++
			}
			out = append(out, clip(sum))
		}
		r.at += r.step()
	}

	idx := min(r.at/r.up, frames)
	drop := max(idx-(ratioTaps-1), 0)

	r.at -= drop * r.up
	r.tail = append(r.tail[:0], buf[drop*ch:]...)

	return out
}

func (r *Rational) Reset() {
	r.at = 0
	r.skew, r.owed = 0, 0

	r.tail = append(r.tail[:0], make([]int16, (ratioTaps-1)*r.channels)...)
	r.at = (ratioTaps - 1) * r.up
}

func (r *Rational) Clipped() uint64 { return r.clipped }

func (r *Rational) Skew(ppm float64) { r.skew = ppm * 1e-6 * float64(r.down) }

func (r *Rational) step() int {
	r.owed += r.skew
	extra := int(r.owed)
	r.owed -= float64(extra)
	return r.down + extra
}

func lowpass(phases, taps int) []float32 {
	n := phases * taps
	h := make([]float32, n)

	center := float64(n-1) / 2
	for i := range h {
		x := (float64(i) - center) / float64(phases)

		s := 1.0
		if x != 0 {
			s = math.Sin(math.Pi*x) / (math.Pi * x)
		}

		w := 0.54 - 0.46*math.Cos(2*math.Pi*float64(i)/float64(n-1))
		h[i] = float32(s * w)
	}

	for p := range phases {
		var sum float32
		for k := range taps {
			sum += h[p+k*phases]
		}
		if sum == 0 {
			continue
		}
		for k := range taps {
			h[p+k*phases] /= sum
		}
	}
	return h
}

func clip(v float32) int16 {
	switch {
	case v > 32767:
		return 32767
	case v < -32768:
		return -32768
	}
	return int16(v)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}
