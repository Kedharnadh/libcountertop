package upsample

import (
	"math"

	"github.com/ygelfand/libcountertop/pkg/audio/resample"
)

const VoiceRate = 16000

const headroom = 0.9

type Resampler interface {
	Run(mono []int16, out []int16) []int16

	Reset()

	Clipped() uint64
}

type Kind string

const (
	Sinc   Kind = "sinc"
	Linear Kind = "linear"
	Hold   Kind = "hold"
)

func Kinds() []Kind { return []Kind{Sinc, Linear, Hold} }

func New(k Kind, rate int) (Resampler, Kind) {
	factor := max(rate/VoiceRate, 1)
	switch k {
	case Linear:
		return &linear{factor: factor}, k
	case Hold:
		return hold{factor: factor}, k
	}
	return newSinc(rate), Sinc
}

type sinc struct{ up *resample.Rational }

func newSinc(rate int) *sinc {
	up := resample.NewRational(VoiceRate, rate, 1)
	up.Headroom(headroom)
	return &sinc{up: up}
}

func (u *sinc) Reset()          { u.up.Reset() }
func (u *sinc) Clipped() uint64 { return u.up.Clipped() }

func (u *sinc) Run(mono []int16, out []int16) []int16 {
	for _, v := range u.up.Run(mono) {
		out = append(out, v, v)
	}
	return out
}

type linear struct {
	factor int
	prev   int16
}

func (l *linear) Reset()        { l.prev = 0 }
func (*linear) Clipped() uint64 { return 0 }

func (l *linear) Run(mono []int16, out []int16) []int16 {
	for _, s := range mono {
		step := (float32(s) - float32(l.prev)) / float32(l.factor)
		for p := range l.factor {
			v := int16(float32(l.prev) + step*float32(p+1))
			out = append(out, v, v)
		}
		l.prev = s
	}
	return out
}

type hold struct{ factor int }

func (hold) Reset()          {}
func (hold) Clipped() uint64 { return 0 }

func (h hold) Run(mono []int16, out []int16) []int16 {
	for _, s := range mono {
		for range h.factor {
			out = append(out, s, s)
		}
	}
	return out
}

const (
	sweepMs   = 1500
	sweepFrom = 300.0
	sweepTo   = 7000.0
)

func Sweep() []int16 {
	frames := VoiceRate * sweepMs / 1000
	out := make([]int16, frames)

	var phase float64
	for i := range out {
		hz := sweepFrom + (sweepTo-sweepFrom)*float64(i)/float64(frames)
		phase += 2 * math.Pi * hz / VoiceRate

		env := math.Min(1, math.Min(float64(i), float64(frames-i))/float64(VoiceRate/50))
		out[i] = int16(0.5 * env * math.MaxInt16 * math.Sin(phase))
	}
	return out
}
