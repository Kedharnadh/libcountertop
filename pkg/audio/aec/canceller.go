package aec

import (
	"fmt"
	"log/slog"
	"math"
	"sync/atomic"
)

const (
	Taps  = 1024
	Block = 64

	echoSuppressActive = -40
)

func NewSuppressor(rate int, echo *MDF) (*Preprocessor, error) {
	p, err := NewPreprocessor(Block, rate, echo)
	if err != nil {
		return nil, err
	}
	p.EchoSuppressActive = echoSuppressActive
	return p, nil
}

type Canceller struct {
	rate    int
	taps    int
	filters []*MDF

	// speex averages ERLE over the last half second.
	erle, best []atomic.Int64

	active atomic.Bool

	refE, micE float64
	blocks     int
	samples    int
}

func NewCanceller(channels, taps, rate int) (*Canceller, error) {
	if channels < 1 {
		return nil, fmt.Errorf("aec: %d channels", channels)
	}
	c := &Canceller{
		rate:    rate,
		taps:    taps,
		filters: make([]*MDF, channels),
		erle:    make([]atomic.Int64, channels),
		best:    make([]atomic.Int64, channels),
	}
	for i := range c.filters {
		f, err := NewMDF(Block, taps, rate)
		if err != nil {
			return nil, err
		}
		c.filters[i] = f
	}
	return c, nil
}

func (c *Canceller) Filter(ch int) *MDF { return c.filters[ch] }

func (c *Canceller) SetAdapting(on bool) {
	for _, f := range c.filters {
		f.SetAdapting(on)
	}
}

func (c *Canceller) Reset() {
	for _, f := range c.filters {
		f.Reset()
	}
}

func (c *Canceller) Active() bool { return c.active.Load() }

func (c *Canceller) ERLE(ch int) float64 { return float64(c.erle[ch].Load()) / 1000 }

func (c *Canceller) Process(
	ref []int16,
	mics, out [][]int16,
	after func(ch int, block []int16),
) error {
	n := len(ref)
	if len(mics) != len(c.filters) || len(out) != len(c.filters) {
		return fmt.Errorf(
			"aec: %d mic and %d out channels for %d filters",
			len(mics),
			len(out),
			len(c.filters),
		)
	}
	for ch := range mics {
		if len(mics[ch]) != n || len(out[ch]) != n {
			return ErrLength
		}
	}
	if n%Block != 0 {
		return fmt.Errorf("aec: %d samples is not a whole number of %d sample frames", n, Block)
	}

	if !c.active.Swap(true) {
		slog.Debug("echo cancellation running", "engine", "speex", "taps", c.taps)
	}
	c.refE += power(ref)
	var mic float64
	for _, m := range mics {
		mic += power(m)
	}
	c.micE += mic / float64(len(mics))
	c.blocks++
	c.samples += n

	for k := 0; k < n; k += Block {
		for ch, f := range c.filters {
			got, err := f.Process(mics[ch][k:k+Block], ref[k:k+Block])
			if err != nil {
				return err
			}
			block := out[ch][k : k+Block]
			copy(block, got)
			if after != nil {
				after(ch, block)
			}
		}
	}

	for ch, f := range c.filters {
		erle := int64(f.ERLE() * 1000)
		c.erle[ch].Store(erle)
		if erle > c.best[ch].Load() {
			c.best[ch].Store(erle)
		}
	}
	return nil
}

func (c *Canceller) Idle() {
	if !c.active.Swap(false) {
		return
	}
	best := make([]float64, len(c.filters))
	last := make([]float64, len(c.filters))
	for ch := range c.filters {
		best[ch] = float64(c.best[ch].Swap(0)) / 1000
		last[ch] = float64(c.erle[ch].Swap(0)) / 1000
	}
	slog.Debug("echo cancellation idle",
		"best_db", best, "last_db", last,
		"ref_dbfs", meanDBFS(c.refE, c.blocks), "mic_dbfs", meanDBFS(c.micE, c.blocks),
		"seconds", round1(float64(c.samples)/float64(c.rate)))
	c.refE, c.micE, c.blocks, c.samples = 0, 0, 0, 0
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func meanDBFS(energy float64, blocks int) float64 {
	if blocks == 0 || energy <= 0 {
		return -120
	}
	return round1(10 * math.Log10(energy/float64(blocks)/(full*full)))
}

func power(s []int16) float64 {
	if len(s) == 0 {
		return 0
	}
	var sum float64
	for _, v := range s {
		sum += float64(v) * float64(v)
	}
	return sum / float64(len(s))
}
