package aec

import (
	"errors"
	"testing"
)

func TestCancellerCancelsEachChannelAndGoesIdle(t *testing.T) {
	c, err := NewCanceller(2, Taps, 16000)
	if err != nil {
		t.Fatal(err)
	}

	near, far := make([]float64, 60), make([]float64, 90)
	near[57], far[83] = 0.5, 0.3

	ref := noiseAt(48000, 8000, 1)
	mics := [][]int16{echoed(ref, near), echoed(ref, far)}

	afters := 0
	out := [][]int16{make([]int16, frame), make([]int16, frame)}
	for at := 0; at+frame <= len(ref); at += frame {
		in := [][]int16{mics[0][at : at+frame], mics[1][at : at+frame]}
		err := c.Process(ref[at:at+frame], in, out, func(int, []int16) { afters++ })
		if err != nil {
			t.Fatal(err)
		}
	}

	if !c.Active() {
		t.Error("not active while processing")
	}
	if want := 2 * len(ref) / Block; afters != want {
		t.Errorf("after ran %d times, want %d", afters, want)
	}
	for ch := range 2 {
		if got := c.ERLE(ch); got < 10 {
			t.Errorf("channel %d: erle %.1f dB, want at least 10", ch, got)
		}
	}

	c.Idle()
	if c.Active() || c.ERLE(0) != 0 {
		t.Errorf("idle left active=%t erle=%.1f", c.Active(), c.ERLE(0))
	}
}

func TestCancellerRejectsMismatchedFrames(t *testing.T) {
	c, err := NewCanceller(1, Taps, 16000)
	if err != nil {
		t.Fatal(err)
	}
	ref := make([]int16, frame)
	if err := c.Process(
		ref,
		[][]int16{make([]int16, frame-1)},
		[][]int16{make([]int16, frame)},
		nil,
	); !errors.Is(
		err,
		ErrLength,
	) {
		t.Errorf("a short mic gave %v, want ErrLength", err)
	}
	if err := c.Process(ref[:100], [][]int16{ref[:100]}, [][]int16{ref[:100]}, nil); err == nil {
		t.Error("a partial block was accepted")
	}
	if c.Active() {
		t.Error("a rejected frame made the canceller active")
	}
}
