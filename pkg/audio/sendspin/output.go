// Package sendspin plays a room's part of a synchronized stream.
//
// Wire format and clock filtering come from sendspin-go. This package owns buffering and
// placement; the product supplies its device output and control callbacks.
package sendspin

import (
	"fmt"
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	ssync "github.com/Sendspin/sendspin-go/pkg/sync"
)

// holdMax bounds what is held ahead. The server keeps within the buffer capacity we advertised, so
// reaching this means a timestamp we cannot believe rather than a server sending too much.
const holdMax = 60 * Rate

// Drift correction. The anchor maps server time to output frames at the nominal rate, but the speaker's
// clock is not the server's: measured on device, one Dot ran about 200 ppm fast and pulled ahead of
// another by 12 ms a minute. So every render period the frame being heard is compared with the frame
// the server clock says should be, the error is smoothed, and one frame is dropped or repeated per
// period while the smoothed error is outside the band. A frame is 21 us; the spec's own suggestion,
// and inaudible at the handful per second a real drift needs.
//
// driftGain is the smoothing: at 47 periods a second, a time constant of about a second, enough to
// take the scheduling jitter out of when a period happens to be rendered. driftBand is 2 ms either
// side, which is what that jitter leaves: off carries several milliseconds of it, and a second of
// smoothing brings the residual down to the order of one. A band inside that residual corrects
// against the noise instead of the drift, and spends most of its corrections undoing the last ones.
//
// An error past snapBand is not drift but a misplaced anchor, most often a period's worth of phase
// between the write counter and the card at the moment the stream started, or an underrun that moved
// the counter on without playing anything. That is put right in one step of silence or one skip, which
// the spec allows on a start, and the fine correction takes it from there.
//
// tailFrames is the hardware tail in frames: the write counter is that far ahead of what is heard, and
// the anchor was laid in terms of what is heard.
const (
	driftGain = 0.02
	driftBand = Rate / 500
	snapBand  = Rate / 100

	// maxSnap is the largest correction that can be one, above which the anchor predates a change of
	// clock rather than having drifted.
	maxSnap = 2 * Rate
)

// out places this room's audio by output frame index. Arrival order cannot line two rooms up: a burst
// of jitter on one of them shifts it against the other for good, because nothing says where the audio
// was meant to go. The server's timestamps say, so they decide.
type Output struct {
	p        Sink
	controls Controls
	clock    *ssync.ClockSync

	mu    sync.Mutex
	ready bool
	held  bool
	gain  float32

	// pcm holds frames from base onward. played is where the card has got to, which is what decides
	// whether a chunk is late — base only says what we happen to be holding.
	base   uint64
	played uint64
	pcm    []int16

	// The frame that carries server time at. Fixed once per stream: recomputing it per chunk would
	// feed the sampling jitter of "what is playing now" straight back into where audio lands. Drift
	// correction nudges frame by whole frames instead, in step with the audio it moves.
	anchored bool
	frame    uint64
	at       int64

	// drift is the smoothed error between the frame being heard and the frame the server clock wants,
	// in frames, positive when the room is playing early. corrected counts frames repeated (positive)
	// or dropped (negative) to hold it.
	drift     float64
	corrected int64

	// nextReport is the frame the next correction line is due at, one a second.
	nextReport uint64

	// now stands in for the clock, so a test can hold the server's time still.
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

// use points the renderer at this session's clock. One server at a time, so it changes only between
// sessions.
func (o *Output) use(clock *ssync.ClockSync) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.clock = clock
}

// open refuses anything the speaker cannot play, rather than playing it at the wrong speed.
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

// write places a decoded chunk at the frame its timestamp asks for.
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

	// A chunk that reaches us after its frames have played is late: whatever is left of it still
	// belongs where it was meant to go, and the rest is gone. Playing it anyway is what leaves a room
	// behind for good.
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

// frameFor converts a server timestamp to an output frame. Wants mu.
func (o *Output) frameFor(at int64) uint64 {
	if o.anchored {
		return uint64(int64(o.frame) + (at-o.at)*Rate/1e6)
	}

	// Frame Written() is going to the card now and is heard a hardware tail later, so this is where
	// the server's intended moment falls. The tail is a constant and the same on every Dot, so what it
	// costs in absolute accuracy it does not cost in lining rooms up.
	ahead := time.Until(o.clock.ServerToLocalTime(at)) - o.hardwareTail()
	o.frame = o.p.Written() + uint64(max(0, ahead.Seconds()*Rate))
	o.at = at
	o.anchored = true
	o.played = o.frame

	// Both conversions of the same timestamp: ahead_ms goes through local time, lead_ms stays in the
	// server's frame, and correct() holds the room to the second. They should agree.
	slog.Debug("sendspin anchored", "frame", o.frame, "ahead_ms", ahead.Milliseconds(),
		"lead_ms", (at-o.clock.ServerMicrosNow())/1000, "written", o.p.Written(),
		"quality", o.clock.CheckQuality())
	return o.frame
}

// Render is the speaker asking what this room plays next.
func (o *Output) Render(from uint64, buf []int16) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if from > o.played {
		o.played = from
	}

	// The card has passed these, and an underrun can ask for the same frames twice, so dropping has to
	// be idempotent rather than a consume.
	if from > o.base && len(o.pcm) > 0 {
		if gone := int((from - o.base) * Channels); gone >= len(o.pcm) {
			o.pcm = o.pcm[:0]
		} else {
			o.pcm = append(o.pcm[:0], o.pcm[gone:]...)
		}
	}
	if from > o.base {
		o.base = from
	}

	if o.held || !o.ready || from < o.base {
		return
	}
	o.correct(from)
	for i := range min(len(buf), len(o.pcm)) {
		buf[i] = scale(o.pcm[i], o.gain)
	}
}

// correct holds the room to the server clock. Wants mu, and pcm's head at from.
func (o *Output) correct(from uint64) {
	if !o.anchored || len(o.pcm) < 2*Channels {
		return
	}
	now := o.now
	if now == nil {
		if o.clock == nil {
			return
		}
		// Without a sync the clock reads the server's timestamps as local ones, decades out, and every
		// period would re-anchor against an answer it does not have. Hold what is queued: the anchor is
		// wrong until a sync lands, and the first correction after one puts it right in a single step.
		if o.clock.CheckQuality() == ssync.QualityLost {
			return
		}
		now = o.clock.ServerMicrosNow
	}

	// The frame the server clock says should be heard now, against the one that is: the frame being
	// rendered less the tail it has yet to travel.
	want := int64(o.frame) + (now()-o.at)*Rate/1e6

	// The loop has the hardware tail in it: frames already handed to the card still play at the old
	// alignment, so a correction at full gain overshoots and the next one swings back. driftGain is
	// what damps that, and a snap is bounded by it for the same reason.
	off := float64(int64(from) - int64(o.hardwareTail()*Rate/time.Second) - want)
	o.drift += (off - o.drift) * driftGain

	// What the correction is actually looking at, once a second: off should sit near zero and stay
	// there. Where it does not, these say whether the room, the anchor or the server clock is moving.
	if from >= o.nextReport {
		o.nextReport = from + Rate
		slog.Debug("sendspin correction", "off_ms", int64(off)*1000/Rate,
			"drift_ms", int64(o.drift)*1000/Rate, "from", from, "want", want,
			"anchor_frame", o.frame, "corrected", o.corrected,
			"queued_ms", len(o.pcm)/Channels*1000/Rate)
	}

	// Re-anchor rather than snap: the next chunk lays one against the clock the server is using now.
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
		o.pcm = append(o.pcm[:0], o.pcm[n*Channels:]...)
		o.frame -= uint64(n)
		o.corrected -= int64(n)
		o.drift = 0
		slog.Debug("sendspin snapped earlier", "ms", n*1000/Rate)
	case o.drift > driftBand:
		// Early: say the first frame twice, and move the anchor with it so what arrives next lands in
		// step with what is already queued.
		o.pcm = append(o.pcm, o.pcm[:Channels]...)
		copy(o.pcm[Channels:], o.pcm[:len(o.pcm)-Channels])
		o.frame++
		o.drift--
		o.corrected++
	case o.drift < -driftBand:
		// Late: skip the first frame.
		n := copy(o.pcm, o.pcm[Channels:])
		o.pcm = o.pcm[:n]
		o.frame--
		o.drift++
		o.corrected--
	}
}

// drifting reports the smoothed error in frames and the frames corrected so far, for the log.
func (o *Output) drifting() (drift float64, corrected int64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.drift, o.corrected
}

// flush drops what has not been heard yet, and the anchor with it: what comes next is a new timeline.
func (o *Output) flush() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.reset()
}

// reset wants mu.
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

// setVolume and setMuted go through the device's own levels rather than the speaker underneath
// them: those own the level, show it in Home Assistant and remember it across a restart.
//
// The group speaks percentages and so do we, so nothing is converted.
func (o *Output) setVolume(level int) {
	if o.controls.SetVolume != nil {
		o.controls.SetVolume(max(0, min(level, 100)))
	}
}

func (o *Output) setMuted(muted bool) {
	if o.controls.SetMuted != nil {
		o.controls.SetMuted(muted)
	}
}

// Stand only holds this room's output. The group continues, and releasing the speaker
// rejoins its current frame rather than replaying audio missed while another producer was heard.
func (o *Output) Stand(down bool) {
	o.mu.Lock()
	was := o.held
	o.held = down
	o.mu.Unlock()
	if was != down {
		slog.Info("sendspin standing", "down", down, "queued_ms", o.queuedMs())
	}
}

// ducked is how far the group drops under a voice turn. It always quietens and never pauses: a hole
// in one room of a whole-house stream is worse than a quiet patch, and everyone else carries on
// regardless of what this room does.

func (o *Output) Duck(on bool) {
	gain := float32(1)
	if on {
		if o.controls.DuckDB != nil {
			gain = ducking.Gain(o.controls.DuckDB())
		}
	}

	o.mu.Lock()
	o.gain = gain
	o.mu.Unlock()
}

// Requeue has nothing to do: the level is applied as frames are rendered, so none are waiting at the
// one they were decoded at.
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
		return max(0, min(o.controls.Volume(), 100))
	}
	return 100
}
