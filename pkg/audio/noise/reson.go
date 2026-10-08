package noise

import "math"

type reson struct {
	a1, a2 float32
	y1, y2 float32

	// A two-pole resonator's gain denominator factors to (1-r)·|1 - r·e^-2jω|.
	drive float32
}

func newReson(freq, decay float32, rate int) reson {
	r := math.Exp(-1 / float64(decay*float32(rate)))
	w := 2 * math.Pi * float64(freq) / float64(rate)

	return reson{
		a1:    float32(2 * r * math.Cos(w)),
		a2:    float32(-r * r),
		drive: float32((1 - r) * math.Sqrt(1-2*r*math.Cos(2*w)+r*r)),
	}
}

func (o *reson) run(x float32) float32 {
	y := o.a1*o.y1 + o.a2*o.y2 + x*o.drive
	o.y2, o.y1 = o.y1, y
	return y
}

type burst struct {
	reson
	env, fade float32
	left      int
}

func newBurst(freq, ring, decay, amp float32, rate int) burst {
	return burst{
		reson: newReson(freq, ring, rate),
		env:   amp,
		fade:  float32(math.Exp(-1 / float64(decay*float32(rate)))),
		left:  int(decay * 3 * float32(rate)),
	}
}

func (b *burst) run(w float32) float32 {
	y := b.reson.run(w * b.env)
	b.env *= b.fade
	b.left--
	return y
}

func (b *burst) done() bool { return b.left <= 0 }

type bursts struct {
	live []burst
	most int
}

func newBursts(most int) bursts { return bursts{live: make([]burst, 0, most), most: most} }

func (p *bursts) add(b burst) {
	if len(p.live) < p.most {
		p.live = append(p.live, b)
	}
}

func (p *bursts) sum(g *Gen) float32 {
	w := g.white()

	var out float32
	for i := len(p.live) - 1; i >= 0; i-- {
		out += p.live[i].run(w)

		if p.live[i].done() {
			p.live[i] = p.live[len(p.live)-1]
			p.live = p.live[:len(p.live)-1]
		}
	}
	return out
}
