package sendspin

import (
	"fmt"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	ssync "github.com/Sendspin/sendspin-go/pkg/sync"

	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
)

const holdMax = 60 * Rate

// The spec suggests dropping or repeating one frame per period, and allows one silence step or skip on a start.
const (
	driftGain = 0.02
	driftBand = Rate / 500
	snapBand  = Rate / 100

	maxSnap = 2 * Rate
)

type Output struct {
	p        Sink
	controls Controls
	clock    *ssync.ClockSync

	mu    sync.Mutex
	ready bool
	held  bool
	gain  float32

	base   uint64
	played uint64
	pcm    []int16

	anchored bool
	frame    uint64
	at       int64

	drift     float64
	corrected int64

	nextReport uint64

	now func() int64

	late    atomic.Int64
	dropped atomic.Int64
}

var (
	_ Producer = (*Output)(nil)
	_ Renderer = (*Output)(nil)
)

func NewOutput(p Sink, controls ...Controls) *Output {
	o := &Output{p: p, gain: 1}
	if len(controls) > 0 {
		o.controls = controls[0]
	}
	return o
}

func (o *Output) use(clock *ssync.ClockSync) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.clock = clock
}

func (o *Output) open(sampleRate, channels, bitDepth int) error {
	if sampleRate != Rate || channels != Channels || bitDepth != Bits {
		return fmt.Errorf("sendspin: cannot play %d Hz/%dch/%d-bit, the speaker is %d/%d/%d",
			sampleRate, channels, bitDepth, Rate, Channels, Bits)
	}

	o.mu.Lock()
	o.ready = true
	o.mu.Unlock()

	o.p.Attach(o)
	return nil
}

func (o *Output) close() {
	o.p.Attach(nil)

	o.mu.Lock()
	defer o.mu.Unlock()
	o.ready = false
	o.reset()
}

func (o *Output) write(at int64, samples []int16) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.ready {
		o.dropped.Add(1)
		return
	}

	frame := o.frameFor(at)
	if frame > o.played && frame-o.played > holdMax {
		o.dropped.Add(1)
		return
	}

	span := uint64(len(samples) / Channels)
	if frame+span <= o.played {
		o.late.Add(1)
		return
	}
	if frame < o.played {
		samples = samples[(o.played-frame)*Channels:]
		frame = o.played
	}

	if len(o.pcm) == 0 {
		o.base = frame
	}
	if frame < o.base {
		o.pcm = append(make([]int16, (o.base-frame)*Channels), o.pcm...)
		o.base = frame
	}

	off := int(frame-o.base) * Channels
	if need := off + len(samples); need > len(o.pcm) {
		o.pcm = append(o.pcm, make([]int16, need-len(o.pcm))...)
	}
	copy(o.pcm[off:], samples)
}

func (o *Output) frameFor(at int64) uint64 {
	if o.anchored {
		return uint64(int64(o.frame) + (at-o.at)*Rate/1e6)
	}

	ahead := time.Until(o.clock.ServerToLocalTime(at)) - o.hardwareTail()
	o.frame = o.p.Written() + uint64(max(0, ahead.Seconds()*Rate))
	o.at = at
	o.anchored = true
	o.played = o.frame

	slog.Debug("sendspin anchored", "frame", o.frame, "ahead_ms", ahead.Milliseconds(),
		"lead_ms", (at-o.clock.ServerMicrosNow())/1000, "written", o.p.Written(),
		"quality", o.clock.CheckQuality())
	return o.frame
}

func (o *Output) Render(from uint64, buf []int16) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if from > o.played {
		o.played = from
	}

	var spent []int16
	if from > o.base && len(o.pcm) > 0 {
		if gone := int((from - o.base) * Channels); gone >= len(o.pcm) {
			o.pcm = o.pcm[:0]
		} else {
			spent, o.pcm = o.pcm[:gone], o.pcm[gone:]
		}
	}
	if from > o.base {
		o.base = from
	}

	if o.held || !o.ready || from < o.base {
		return
	}
	o.correct(from, spent)
	for i := range min(len(buf), len(o.pcm)) {
		buf[i] = scale(o.pcm[i], o.gain)
	}
}

func (o *Output) correct(from uint64, spent []int16) {
	if !o.anchored || len(o.pcm) < 2*Channels {
		return
	}
	now := o.now
	if now == nil {
		if o.clock == nil {
			return
		}
		if o.clock.CheckQuality() == ssync.QualityLost {
			return
		}
		now = o.clock.ServerMicrosNow
	}

	want := int64(o.frame) + (now()-o.at)*Rate/1e6

	off := float64(int64(from) - int64(o.hardwareTail()*Rate/time.Second) - want)
	o.drift += (off - o.drift) * driftGain

	if from >= o.nextReport {
		o.nextReport = from + Rate
		slog.Debug("sendspin correction", "off_ms", int64(off)*1000/Rate,
			"drift_ms", int64(o.drift)*1000/Rate, "from", from, "want", want,
			"anchor_frame", o.frame, "corrected", o.corrected,
			"queued_ms", len(o.pcm)/Channels*1000/Rate)
	}

	if off > maxSnap || off < -maxSnap {
		slog.Warn("sendspin re-anchoring", "off_s", int64(off)/Rate)
		o.anchored = false
		o.drift = 0
		return
	}

	switch {
	case o.drift > snapBand:
		n := int(o.drift)
		o.pcm = append(make([]int16, n*Channels), o.pcm...)
		o.frame += uint64(n)
		o.corrected += int64(n)
		o.drift = 0
		slog.Debug("sendspin snapped later", "ms", n*1000/Rate)
	case o.drift < -snapBand:
		n := min(int(-o.drift), len(o.pcm)/Channels-1)
		o.pcm = o.pcm[n*Channels:]
		o.frame -= uint64(n)
		o.corrected -= int64(n)
		o.drift = 0
		slog.Debug("sendspin snapped earlier", "ms", n*1000/Rate)
	case o.drift > driftBand:
		if k := len(spent); k >= Channels {
			o.pcm = spent[k-Channels : k+len(o.pcm)]
			copy(o.pcm, o.pcm[Channels:2*Channels])
		} else {
			o.pcm = append(o.pcm, o.pcm[:Channels]...)
			copy(o.pcm[Channels:], o.pcm[:len(o.pcm)-Channels])
		}
		o.frame++
		o.drift--
		o.corrected++
	case o.drift < -driftBand:
		o.pcm = o.pcm[Channels:]
		o.frame--
		o.drift++
		o.corrected--
	}
}

func (o *Output) drifting() (drift float64, corrected int64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.drift, o.corrected
}

func (o *Output) flush() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.reset()
}

func (o *Output) reset() {
	o.pcm = o.pcm[:0]
	o.base = 0
	o.anchored = false
	o.drift = 0
}

func (o *Output) misses() (late, dropped int64) { return o.late.Load(), o.dropped.Load() }

func (o *Output) queuedMs() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.pcm) / Channels * 1000 / Rate
}

func (o *Output) setVolume(level int) {
	if o.controls.Volume != nil {
		o.controls.Volume.Set(max(0, min(level, 100)))
	}
}

func (o *Output) setMuted(muted bool) {
	if o.controls.Media != nil {
		o.controls.Media.Mute(muted)
	}
}

func (o *Output) Stand(down bool) {
	o.mu.Lock()
	was := o.held
	o.held = down
	o.mu.Unlock()
	if was != down {
		slog.Info("sendspin standing", "down", down, "queued_ms", o.queuedMs())
	}
}

func (o *Output) Duck(on bool) {
	gain := float32(1)
	if on {
		if o.controls.Settings != nil {
			gain = ducking.Gain(o.controls.Settings.Read().DuckDB)
		}
	}

	o.mu.Lock()
	o.gain = gain
	o.mu.Unlock()
}

func (o *Output) Requeue() {}

func scale(s int16, gain float32) int16 {
	if gain == 1 {
		return s
	}
	return int16(max(math.MinInt16, min(float64(s)*float64(gain), math.MaxInt16)))
}

func (o *Output) hardwareTail() time.Duration {
	if o.controls.HardwareTail > 0 {
		return o.controls.HardwareTail
	}
	return DefaultHardwareTail
}
func (o *Output) volume() int {
	if o.controls.Volume != nil {
		return max(0, min(o.controls.Volume.Level(), 100))
	}
	return 100
}
