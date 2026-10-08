package noise

import (
	"math"
	"math/rand/v2"
)

type Fill func(dst []float32)

type Sound struct {
	Name string

	RMS float32

	Peak float32

	New func(g *Gen) Fill
}

func (s Sound) gain() float32 {
	if s.Peak > 0 {
		return Loudest / s.Peak
	}
	return Level / s.RMS
}

const Level = 0.1

const Loudest = 0.6

var catalog = []Sound{
	whiteNoise,
	pinkNoise,
	brownNoise,
	rainSound,
	oceanSound,
	brookSound,
	windSound,
	fireSound,
	cricketSound,
	fanSound,
	cabinSound,
}

var byName = map[string]Sound{}

func init() {
	for _, s := range catalog {
		byName[s.Name] = s
	}
}

func Names() []string {
	out := make([]string, 0, len(catalog))
	for _, s := range catalog {
		out = append(out, s.Name)
	}
	return out
}

func Has(name string) bool { _, ok := byName[name]; return ok }

func New(name string, rate int) Fill { return Mix(rate, name) }

func Mix(rate int, names ...string) Fill {
	return mixSeeded(rate, rand.Uint64(), names...)
}

func mixSeeded(rate int, seed uint64, names ...string) Fill {
	var (
		fills  []Fill
		gains  []float32
		sparse []bool
		steady int
	)
	for i, name := range names {
		s, ok := byName[name]
		if !ok {
			continue
		}

		g := &Gen{Rate: rate, rand: newRand(seed + uint64(i))}
		fills = append(fills, s.New(g))
		gains = append(gains, s.gain())
		sparse = append(sparse, s.Peak > 0)
		if s.Peak == 0 {
			steady++
		}
	}

	switch len(fills) {
	case 0:
		return nil
	case 1:
		return scaled(fills[0], gains[0])
	}

	together := 1 / float32(math.Sqrt(float64(max(1, steady))))
	for i := range gains {
		if !sparse[i] {
			gains[i] *= together
		}
	}
	buf := make([]float32, 0)

	return func(dst []float32) {
		if len(buf) < len(dst) {
			buf = make([]float32, len(dst))
		}
		clear(dst)

		for i, fill := range fills {
			into := buf[:len(dst)]
			fill(into)

			for j, v := range into {
				dst[j] += v * gains[i]
			}
		}
		for i, v := range dst {
			dst[i] = clamp(v)
		}
	}
}

func scaled(fill Fill, gain float32) Fill {
	return func(dst []float32) {
		fill(dst)
		for i, v := range dst {
			dst[i] = clamp(v * gain)
		}
	}
}

func clamp(v float32) float32 { return min(1, max(-1, v)) }
