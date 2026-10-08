package noise

import (
	"math"
	"math/rand/v2"
)

type Gen struct {
	Rate int

	rand *rand.Rand
}

func newRand(seed uint64) *rand.Rand { return rand.New(rand.NewPCG(seed, seed>>32)) }

func (g *Gen) white() float32 { return g.rand.Float32()*2 - 1 }

func (g *Gen) between(lo, hi float32) float32 { return lo + g.rand.Float32()*(hi-lo) }

func (g *Gen) chance(perSecond float32) bool {
	return g.rand.Float32() < perSecond/float32(g.Rate)
}

const ctrl = 64

type walk struct {
	value, target float32
	lo, hi        float32
	rate          float32
	coef          float32
}

func newWalk(lo, hi, period float32, rate int) walk {
	mid := (lo + hi) / 2
	return walk{
		value: mid, target: mid,
		lo: lo, hi: hi,
		rate: 1 / period,
		coef: onePole(1/period, rate),
	}
}

func (w *walk) next(g *Gen) float32 {
	if g.chance(w.rate) {
		w.target = g.between(w.lo, w.hi)
	}
	w.value += (w.target - w.value) * w.coef
	return w.value
}

type lowpass float32

func (l *lowpass) run(x, coef float32) float32 {
	*l += lowpass((x - float32(*l)) * coef)
	return float32(*l)
}

type comb struct {
	buf      []float32
	at       int
	feedback float32
}

func newComb(delay, feedback float32, rate int) comb {
	return comb{buf: make([]float32, max(1, int(delay*float32(rate)))), feedback: feedback}
}

func (c *comb) run(x float32) float32 {
	y := x + c.buf[c.at]*c.feedback
	c.buf[c.at] = y
	c.at = (c.at + 1) % len(c.buf)
	return y
}

const sineSize = 1024

var sineTable = func() [sineSize + 1]float32 {
	var t [sineSize + 1]float32
	for i := range t {
		t[i] = float32(math.Sin(2 * math.Pi * float64(i) / sineSize))
	}
	return t
}()

func sine(phase float32) float32 {
	x := phase * sineSize
	i := int(x)
	f := x - float32(i)
	return sineTable[i] + (sineTable[i+1]-sineTable[i])*f
}

func onePole(freq float32, rate int) float32 {
	return 1 - float32(math.Exp(-2*math.Pi*float64(freq)/float64(rate)))
}

func corner(freq float32, rate int) float32 {
	return min(1, 2*math.Pi*freq/float32(rate))
}
