package noise

const NameRain = "Rain"

const (
	rainFall = 55

	rainLow, rainHigh   = 900, 4500
	rainQuick, rainSlow = 0.004, 0.014

	rainRing = 0.005

	rainBed  = 500
	rainWash = 0.5

	rainSoft, rainHard = 0.55, 1.0
	rainGust           = 7
)

var rainSound = Sound{
	Name: NameRain,
	RMS:  0.45993,
	New: func(g *Gen) Fill {
		var (
			bed  pink
			low  lowpass
			gust = newWalk(rainSoft, rainHard, rainGust, g.Rate)
			near = newBursts(48)
			cut  = onePole(rainBed, g.Rate)
		)

		return func(dst []float32) {
			for i := range dst {
				p := bed.next(g)
				out := (p - low.run(p, cut)) * gust.next(g) * rainWash

				if g.chance(rainFall) {
					amp := g.between(0.45, 1)
					near.add(newBurst(
						g.between(rainLow, rainHigh),
						rainRing,
						g.between(rainQuick, rainSlow),
						amp*amp,
						g.Rate,
					))
				}
				dst[i] = out + near.sum(g)
			}
		}
	},
}
