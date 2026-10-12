package vad

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"math"
	"sync"

	"github.com/ygelfand/libcountertop/pkg/audio/fft"
	"github.com/ygelfand/libcountertop/pkg/inference/vec"
)

const (
	SampleRate = 16000
	Chunk      = 512

	context = 64
	window  = 256
	hop     = 128
	bins    = window/2 + 1
	steps   = (Chunk+2*context-window)/hop + 1
	hidden  = 128
)

//go:embed model/silero.bin
var weights []byte

type layer struct {
	w      []float32
	b      []float32
	out    int
	in     int
	stride int
}

type model struct {
	encoder       [4]layer
	inputWeights  []float32
	hiddenWeights []float32
	inputBias     []float32
	hiddenBias    []float32
	dw            []float32
	db            float32
	hann          [window]float32
}

var load = sync.OnceValues(func() (*model, error) {
	if len(weights)%4 != 0 {
		return nil, errors.New("vad: weights are not float32")
	}
	all := make([]float32, len(weights)/4)
	for i := range all {
		all[i] = math.Float32frombits(binary.LittleEndian.Uint32(weights[i*4:]))
	}
	take := func(n int) []float32 {
		if len(all) < n {
			return nil
		}
		v := all[:n:n]
		all = all[n:]
		return v
	}
	m := &model{}
	shapes := [4][3]int{{128, bins, 1}, {64, 128, 2}, {64, 64, 2}, {128, 64, 1}}
	for i, s := range shapes {
		m.encoder[i] = layer{
			w:      take(s[0] * s[1] * 3),
			b:      take(s[0]),
			out:    s[0],
			in:     s[1],
			stride: s[2],
		}
	}
	m.inputWeights, m.hiddenWeights = take(4*hidden*hidden), take(4*hidden*hidden)
	m.inputBias, m.hiddenBias = take(4*hidden), take(4*hidden)
	m.dw = take(hidden)
	db := take(1)
	if db == nil || len(all) != 0 {
		return nil, errors.New("vad: weights do not match the model")
	}
	m.db = db[0]
	for n := range m.hann {
		m.hann[n] = float32(0.5 - 0.5*math.Cos(2*math.Pi*float64(n)/window))
	}
	return m, nil
})

type Detector struct {
	m       *model
	fft     *fft.FFT
	context [context]float32
	h       [hidden]float32
	c       [hidden]float32

	padded [Chunk + 2*context]float32
	frame  [window]complex64
	spec   []float32
	work   [2][]float32
	cols   []float32
	gates  [4 * hidden]float32
}

func New() (*Detector, error) {
	m, err := load()
	if err != nil {
		return nil, err
	}
	return &Detector{
		m:    m,
		fft:  fft.New(window),
		spec: make([]float32, bins*steps),
		work: [2][]float32{make([]float32, 128*steps), make([]float32, 128*steps)},
		cols: make([]float32, steps*bins*3),
	}, nil
}

func (d *Detector) Reset() {
	d.context = [context]float32{}
	d.h = [hidden]float32{}
	d.c = [hidden]float32{}
}

func (d *Detector) Speech(chunk []int16) (float32, error) {
	if len(chunk) != Chunk {
		return 0, errors.New("vad: a chunk is 512 samples")
	}
	x := d.padded[:]
	copy(x, d.context[:])
	for i, s := range chunk {
		x[context+i] = float32(s) / 32768
	}
	last := Chunk + context - 1
	for i := range context {
		x[last+1+i] = x[last-1-i]
	}
	copy(d.context[:], x[Chunk:Chunk+context])

	d.spectrum(x)
	in, length := d.spec, steps
	for i := range d.m.encoder {
		l := &d.m.encoder[i]
		out := d.work[i%2]
		length = convolve(l, in, length, out, d.cols)
		in = out
	}
	d.lstm(in)

	sum := d.m.db
	for i, w := range d.m.dw {
		sum += w * max(d.h[i], 0)
	}
	return sigmoid(sum), nil
}

func (d *Detector) spectrum(x []float32) {
	for t := range steps {
		for n := range window {
			d.frame[n] = complex(x[t*hop+n]*d.m.hann[n], 0)
		}
		d.fft.Forward(d.frame[:])
		for k := range bins {
			re, im := real(d.frame[k]), imag(d.frame[k])
			d.spec[k*steps+t] = float32(math.Sqrt(float64(re*re + im*im)))
		}
	}
}

func convolve(l *layer, in []float32, length int, out []float32, cols []float32) int {
	n := (length+2-3)/l.stride + 1
	k := l.in * 3
	for t := range n {
		col := cols[t*k : (t+1)*k]
		start := t*l.stride - 1
		for c := range l.in {
			row := in[c*length : (c+1)*length]
			for j := range 3 {
				v := float32(0)
				if p := start + j; p >= 0 && p < length {
					v = row[p]
				}
				col[c*3+j] = v
			}
		}
	}
	row := func(o int) []float32 { return l.w[o*k : (o+1)*k] }
	for o := 0; o < l.out; o += 4 {
		for t := range n {
			s := vec.Dot4(cols[t*k:(t+1)*k], row(o), row(o+1), row(o+2), row(o+3))
			for j, v := range s {
				out[(o+j)*n+t] = max(l.b[o+j]+v, 0)
			}
		}
	}
	return n
}

func (d *Detector) lstm(x []float32) {
	m := d.m
	x = x[:hidden]
	in := func(g int) []float32 { return m.inputWeights[g*hidden : (g+1)*hidden] }
	rec := func(g int) []float32 { return m.hiddenWeights[g*hidden : (g+1)*hidden] }
	for g := 0; g < len(d.gates); g += 4 {
		a := vec.Dot4(x, in(g), in(g+1), in(g+2), in(g+3))
		b := vec.Dot4(d.h[:], rec(g), rec(g+1), rec(g+2), rec(g+3))
		for j := range 4 {
			d.gates[g+j] = m.inputBias[g+j] + m.hiddenBias[g+j] + a[j] + b[j]
		}
	}
	for j := range hidden {
		i := sigmoid(d.gates[j])
		f := sigmoid(d.gates[hidden+j])
		g := float32(math.Tanh(float64(d.gates[2*hidden+j])))
		o := sigmoid(d.gates[3*hidden+j])
		d.c[j] = f*d.c[j] + i*g
		d.h[j] = o * float32(math.Tanh(float64(d.c[j])))
	}
}

func sigmoid(v float32) float32 { return float32(1 / (1 + math.Exp(-float64(v)))) }
