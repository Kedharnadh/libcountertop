package assistant

import (
	"math"
	"time"

	voice "github.com/ygelfand/libcountertop/pkg/assistant/turn"
)

type Showing struct {
	voice.Showing

	Down float64

	Level float64

	At time.Time

	Reveal float64
}

func Revealed(text string, part float64) string {
	switch runes := []rune(text); {
	case part <= 0:
		return ""
	case part >= 1:
		return text
	default:
		return string(runes[:int(float64(len(runes))*part)])
	}
}

const (
	waveBars = 21

	waveFloor = 0.30

	waveRest = 0.65

	waveHz = 0.8
)

func Bars(show Showing) []float64 {
	t := seconds(show.At)
	level := min(max(show.Level, 0), 1)
	bars := make([]float64, waveBars)
	for i := range bars {
		at := float64(i) / float64(waveBars-1)
		bell := math.Sin(at * math.Pi)
		travel := 0.5 + 0.5*math.Sin(2*math.Pi*(waveHz*t-at*1.5))
		rest := waveFloor * (1 - waveRest + waveRest*travel)
		bars[i] = rest + (1-waveFloor)*level*bell*travel
	}
	return bars
}

const (
	thinkDots  = 3
	thinkCycle = 1.1
)

func Dots(show Showing) []float64 {
	t := seconds(show.At)
	lifts := make([]float64, thinkDots)
	for i := range lifts {
		lifts[i] = max(math.Sin(2*math.Pi*(t/thinkCycle-float64(i)/float64(thinkDots)/2)), 0)
	}
	return lifts
}

func seconds(at time.Time) float64 {
	return float64(at.UnixNano()) / float64(time.Second)
}
