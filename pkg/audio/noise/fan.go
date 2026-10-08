package noise

import "math"

const NameFan = "Fan"

const (
	fanBlade = 62.0

	fanMotor = 120.0
	fanHum   = 0.35

	fanAir   = 1300.0
	fanSweep = 0.3
	fanSway  = 0.2

	fanGrille   = 0.005
	fanFeedback = 0.45
)

var fanSound = Sound{
	Name: NameFan,
	RMS:  2.0833,
	New: func(g *Gen) Fill {
		var (
			bed   pink
			air   lowpass
			motor = newReson(fanMotor, 0.05, g.Rate)
			above = newReson(fanMotor*2, 0.05, g.Rate)
			grill = newComb(fanGrille, fanFeedback, g.Rate)
			drift = newWalk(0.94, 1.06, 15, g.Rate)

			phase float32
			step  = fanBlade / float32(g.Rate)
			swing float32
			n     int
		)

		return func(dst []float32) {
			for i := range dst {
				phase += step
				if phase >= 1 {
					phase--
				}
				if n%ctrl == 0 {
					swing = float32(math.Sin(2 * math.Pi * float64(phase)))
				}
				n++

				out := air.run(bed.next(g), corner(fanAir*(1+fanSweep*swing), g.Rate))
				out = grill.run(out * (1 + fanSway*swing) * drift.next(g))

				w := g.white()
				dst[i] = out + fanHum*(motor.run(w)+above.run(w))
			}
		}
	},
}
