package noise

const NamePink = "Pink"

var pinkNoise = Sound{
	Name: NamePink,
	RMS:  1.76,
	New: func(g *Gen) Fill {
		var f pink
		return func(dst []float32) {
			for i := range dst {
				dst[i] = f.next(g)
			}
		}
	},
}

// Paul Kellett's pink filter: within 0.05 dB of 3 dB/octave from 9 Hz up, derived at 44.1 kHz.
type pink struct{ b [7]float32 }

func (p *pink) next(g *Gen) float32 {
	w := g.white()
	p.b[0] = 0.99886*p.b[0] + w*0.0555179
	p.b[1] = 0.99332*p.b[1] + w*0.0750759
	p.b[2] = 0.96900*p.b[2] + w*0.1538520
	p.b[3] = 0.86650*p.b[3] + w*0.3104856
	p.b[4] = 0.55000*p.b[4] + w*0.5329522
	p.b[5] = -0.7616*p.b[5] - w*0.0168980

	out := p.b[0] + p.b[1] + p.b[2] + p.b[3] + p.b[4] + p.b[5] + p.b[6] + w*0.5362
	p.b[6] = w * 0.115926
	return out
}
