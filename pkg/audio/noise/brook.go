package noise

import "math"

const NameBrook = "Brook"

// A bubble rings at Minnaert's frequency, 3.26 over its radius in meters.
const (
	brookFine, brookFat = 0.0008, 0.004
	minnaert            = 3.26

	brookRise             = 0.12
	brookQuick, brookSlow = 0.004, 0.018

	brookBubbles = 90
	brookMost    = 16

	brookBed    = 900
	brookHiss   = 1.0
	brookAccent = 0.35
)

type bubble struct {
	phase, step, rise float32
	amp, fade         float32
	left              int
}

func newBubble(g *Gen) bubble {
	radius := g.between(brookFine, brookFat)
	decay := g.between(brookQuick, brookSlow)
	life := int(decay * 2 * float32(g.Rate))
	amp := g.between(0, 1)

	return bubble{
		step: minnaert / radius / float32(g.Rate),
		rise: float32(math.Pow(1+brookRise, 1/float64(life))),
		amp:  amp * amp,
		fade: float32(math.Exp(-1 / float64(decay*float32(g.Rate)))),
		left: life,
	}
}

func (b *bubble) next() float32 {
	b.phase += b.step
	if b.phase >= 1 {
		b.phase--
	}
	b.step *= b.rise
	b.amp *= b.fade
	b.left--

	return b.amp * sine(b.phase)
}

var brookSound = Sound{
	Name: NameBrook,
	RMS:  0.98812,
	New: func(g *Gen) Fill {
		var (
			bed     pink
			low     lowpass
			cut     = onePole(brookBed, g.Rate)
			bubbles = make([]bubble, 0, brookMost)
		)

		return func(dst []float32) {
			for i := range dst {
				p := bed.next(g)
				out := (p - low.run(p, cut)) * brookHiss

				if g.chance(brookBubbles) && len(bubbles) < brookMost {
					bubbles = append(bubbles, newBubble(g))
				}

				for j := len(bubbles) - 1; j >= 0; j-- {
					out += bubbles[j].next() * brookAccent

					if bubbles[j].left <= 0 {
						bubbles[j] = bubbles[len(bubbles)-1]
						bubbles = bubbles[:len(bubbles)-1]
					}
				}
				dst[i] = out
			}
		}
	},
}
