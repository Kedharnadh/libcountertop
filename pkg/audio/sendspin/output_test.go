package sendspin

import (
	"testing"
	"time"
)

const tailFrames = int64(DefaultHardwareTail * Rate / time.Second)

func anchored(t *testing.T) *Output {
	t.Helper()

	o := NewOutput(nil)
	o.ready = true
	o.anchored = true
	o.frame = 1000
	o.at = 0
	o.played = 1000
	return o
}

func frames(n int) []int16 { return make([]int16, n*Channels) }

func tone(n int, v int16) []int16 {
	s := frames(n)
	for i := range s {
		s[i] = v
	}
	return s
}

func microsFor(n int64) int64 { return n * 1e6 / Rate }

func rendered(o *Output, from uint64, n int) []int16 {
	buf := frames(n)
	o.Render(from, buf)
	return buf
}

func TestAChunkLandsOnTheFrameItsTimestampAsksFor(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(480), tone(10, 500))

	if got := rendered(o, 1000, 10)[0]; got != 0 {
		t.Errorf("at the anchor = %d, want silence, the audio is 480 frames later", got)
	}
	if got := rendered(o, 1480, 10)[0]; got != 500 {
		t.Errorf("at the placed frame = %d, want 500", got)
	}
}

func TestOrderOfArrivalDoesNotMoveAudio(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(480), tone(10, 700))
	o.write(microsFor(0), tone(10, 300))

	if got := rendered(o, 1000, 10)[0]; got != 300 {
		t.Errorf("first chunk = %d, want 300", got)
	}
	if got := rendered(o, 1480, 10)[0]; got != 700 {
		t.Errorf("second chunk = %d, want 700", got)
	}
}

func TestAGapPlaysAsSilenceRatherThanShiftingWhatFollows(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(10, 400))
	o.write(microsFor(2000), tone(10, 900))

	if got := rendered(o, 1010, 10)[0]; got != 0 {
		t.Errorf("in the gap = %d, want silence", got)
	}
	if got := rendered(o, 3000, 10)[0]; got != 900 {
		t.Errorf("after the gap = %d, want 900", got)
	}
}

func TestAudioThatArrivesAfterItsFrameIsDropped(t *testing.T) {
	o := anchored(t)
	rendered(o, 5000, 10)

	o.write(microsFor(0), tone(10, 600))

	if o.late.Load() != 1 {
		t.Errorf("late = %d, want 1", o.late.Load())
	}
	if got := rendered(o, 5010, 10)[0]; got != 0 {
		t.Errorf("played = %d, want silence", got)
	}
}

func TestAPartlyLateChunkKeepsTheFramesStillToCome(t *testing.T) {
	o := anchored(t)
	rendered(o, 1005, 1)

	o.write(microsFor(0), tone(10, 800))

	if got := rendered(o, 1005, 5)[0]; got != 800 {
		t.Errorf("remainder = %d, want 800", got)
	}
}

func TestRenderingTheSameFrameTwiceGivesTheSameAudio(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(10, 250))

	first := rendered(o, 1000, 10)[0]
	second := rendered(o, 1000, 10)[0]

	if first != 250 || second != 250 {
		t.Errorf("got %d then %d, want 250 both times", first, second)
	}
}

func TestAbsurdlyDistantAudioIsRefused(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(holdMax+Rate), tone(10, 100))

	if o.dropped.Load() != 1 {
		t.Errorf("dropped = %d, want 1", o.dropped.Load())
	}
	if len(o.pcm) != 0 {
		t.Errorf("held %d samples, want none", len(o.pcm))
	}
}

func TestDuckingAppliesAsFramesAreRendered(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(10, 1000))
	o.gain = 0.5

	if got := rendered(o, 1000, 10)[0]; got != 500 {
		t.Errorf("ducked = %d, want 500", got)
	}
}

func TestSuspendedPlaysNothing(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(10, 1000))
	o.held = true

	if got := rendered(o, 1000, 10)[0]; got != 0 {
		t.Errorf("suspended = %d, want silence", got)
	}
}

func TestFlushDropsTheAnchor(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(10, 100))
	o.flush()

	if o.anchored {
		t.Error("still anchored after a flush")
	}
	if len(o.pcm) != 0 {
		t.Errorf("held %d samples, want none", len(o.pcm))
	}
}

func ramp(n int) []int16 {
	s := frames(n)
	for i := range s {
		s[i] = int16(i / Channels)
	}
	return s
}

const heard = 1000 + uint64(tailFrames)

func listening(t *testing.T) *Output {
	t.Helper()
	o := anchored(t)
	o.write(microsFor(0), ramp(int(tailFrames)+4800))
	o.played = heard
	return o
}

func TestDriftCorrectionRepeatsFramesWhenEarly(t *testing.T) {
	o := listening(t)

	early := int64(4 * driftBand)
	o.now = func() int64 { return microsFor(-early) }
	for range 8 * driftBand {
		o.Render(heard, frames(1))
	}

	if o.corrected < early-2*driftBand || o.corrected > early {
		t.Fatalf("corrected %d frames, want most of %d", o.corrected, early)
	}
	if int64(o.frame) != 1000+o.corrected {
		t.Fatalf("anchor frame %d did not move with the %d frames repeated", o.frame, o.corrected)
	}
	if got := len(o.pcm) / Channels; got != 4800+int(o.corrected) {
		t.Fatalf("queue holds %d frames, want %d", got, 4800+int(o.corrected))
	}
	first := int16(heard - 1000)
	for i := range int(o.corrected) + 1 {
		if o.pcm[i*Channels] != first {
			t.Fatalf("frame %d is %d, want a repeat of frame %d", i, o.pcm[i*Channels], first)
		}
	}
	if o.pcm[(int(o.corrected)+1)*Channels] != first+1 {
		t.Fatal("the original next frame did not follow the repeats")
	}
}

func TestDriftCorrectionSkipsFramesWhenLate(t *testing.T) {
	o := listening(t)

	late := int64(4 * driftBand)
	o.now = func() int64 { return microsFor(late) }
	for range 8 * driftBand {
		o.Render(heard, frames(1))
	}

	if o.corrected > -(late-2*driftBand) || o.corrected < -late {
		t.Fatalf("corrected %d frames, want most of %d", o.corrected, -late)
	}
	if int64(o.frame) != 1000+o.corrected {
		t.Fatalf("anchor frame %d did not move with the %d frames skipped", o.frame, -o.corrected)
	}
	if want := int16(heard-1000) + int16(-o.corrected); o.pcm[0] != want {
		t.Fatalf("queue head is frame %d, want %d", o.pcm[0], want)
	}
}

func TestDriftCorrectionSnapsLargeErrors(t *testing.T) {
	o := listening(t)
	o.now = func() int64 { return microsFor(-2400) }
	for range 800 {
		o.Render(heard, frames(1))
	}
	if o.corrected < 2400-2*driftBand || o.corrected > 2400+driftBand {
		t.Fatalf("corrected %d frames, want about 2400", o.corrected)
	}
	if o.pcm[0] != 0 || o.pcm[100*Channels] != 0 {
		t.Fatal("the snap did not insert silence at the head")
	}
	if int64(o.frame) != 1000+o.corrected {
		t.Fatalf("anchor frame %d did not move with the %d frames inserted", o.frame, o.corrected)
	}
}

func TestDriftCorrectionLeavesSmallErrorsAlone(t *testing.T) {
	o := listening(t)
	o.now = func() int64 { return microsFor(-10) }
	for range 400 {
		o.Render(heard, frames(1))
	}
	if o.corrected != 0 || o.frame != 1000 {
		t.Fatalf("corrected %d, frame %d", o.corrected, o.frame)
	}
}
