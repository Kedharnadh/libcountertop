package mix

import (
	"math"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
)

func Chime(rate, channels int, level float64, notes ...tone.Note) []int16 {
	var out []int16
	for _, n := range notes {
		out = append(out, Tone(rate, channels, n, level)...)
	}
	return out
}

func Tone(rate, channels int, n tone.Note, level float64) []int16 {
	frames := rate * n.Ms / 1000
	out := make([]int16, frames*channels)
	if n.Freq == 0 {
		return out
	}

	for i := range frames {
		env := math.Min(1, math.Min(float64(i), float64(frames-i))/float64(rate/200))
		v := int16(
			level * env * math.MaxInt16 * math.Sin(2*math.Pi*n.Freq*float64(i)/float64(rate)),
		)
		for c := range channels {
			out[i*channels+c] = v
		}
	}
	return out
}
