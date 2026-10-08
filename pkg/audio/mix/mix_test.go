package mix

import (
	"bytes"
	"context"
	"errors"
	"math"
	"slices"
	"sync/atomic"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/audio/alsa"
	"github.com/ygelfand/libcountertop/pkg/audio/tone"
)

const (
	rate     = 48000
	channels = 2
)

func TestMixSumsIntoWhatIsQueued(t *testing.T) {
	got := Mix([]int16{100, 200, 300}, []int16{10, 20, 30})

	if want := []int16{110, 220, 330}; !slices.Equal(got, want) {
		t.Errorf("mix = %v, want %v", got, want)
	}
}

func TestMixExtendsPastTheQueue(t *testing.T) {
	got := Mix([]int16{100}, []int16{10, 20, 30})

	if want := []int16{110, 20, 30}; !slices.Equal(got, want) {
		t.Errorf("mix = %v, want %v", got, want)
	}
}

func TestMixClampsInsteadOfWrapping(t *testing.T) {
	got := Mix([]int16{math.MaxInt16 - 10, math.MinInt16 + 10}, []int16{1000, -1000})

	if want := []int16{math.MaxInt16, math.MinInt16}; !slices.Equal(got, want) {
		t.Errorf("mix = %v, want %v", got, want)
	}
}

func TestMixIntoNothing(t *testing.T) {
	got := Mix(nil, []int16{10, 20})

	if want := []int16{10, 20}; !slices.Equal(got, want) {
		t.Errorf("mix = %v, want %v", got, want)
	}
}

func TestNextTakesAtMostWhatIsAskedFor(t *testing.T) {
	var q Queue
	q.Push([]int16{1, 2, 3})

	if got := q.Next(2); !slices.Equal(got, []int16{1, 2}) {
		t.Errorf("first = %v, want [1 2]", got)
	}
	if got := q.Next(2); !slices.Equal(got, []int16{3}) {
		t.Errorf("second = %v, want [3]", got)
	}
	if q.Len() != 0 {
		t.Errorf("%d left, want none", q.Len())
	}
}

func TestOverlaySumsWithWhatIsQueued(t *testing.T) {
	var q Queue
	q.Push([]int16{100, 200})
	q.Overlay([]int16{1, 2, 3})

	if got := q.Take(); !slices.Equal(got, []int16{101, 202, 3}) {
		t.Errorf("queued = %v, want [101 202 3]", got)
	}
}

func TestARestIsSilenceOfTheRightLength(t *testing.T) {
	got := Tone(rate, channels, tone.Note{Freq: 0, Ms: 10}, 1)

	if want := rate * 10 / 1000 * channels; len(got) != want {
		t.Fatalf("a rest is %d samples, want %d", len(got), want)
	}
	for i, v := range got {
		if v != 0 {
			t.Fatalf("a rest has %d at sample %d", v, i)
		}
	}
}

func TestAToneIsOnBothChannels(t *testing.T) {
	got := Tone(rate, channels, tone.Note{Freq: 440, Ms: 50}, 1)

	var heard bool
	for i := 0; i+1 < len(got); i += channels {
		if got[i] != got[i+1] {
			t.Fatalf("channels differ at frame %d: %d and %d", i/channels, got[i], got[i+1])
		}
		if got[i] != 0 {
			heard = true
		}
	}
	if !heard {
		t.Error("the tone is silent")
	}
}

func TestAToneRampsInAndOut(t *testing.T) {
	got := Tone(rate, channels, tone.Note{Freq: 440, Ms: 100}, 1)

	var peak int16
	for _, v := range got {
		peak = max(peak, v)
	}
	if peak == 0 {
		t.Fatal("the tone is silent")
	}

	quiet := peak / 100
	if first := got[0]; first > quiet || first < -quiet {
		t.Errorf("the tone starts at %d against a peak of %d", first, peak)
	}
	if last := got[len(got)-1]; last > quiet || last < -quiet {
		t.Errorf("the tone ends at %d against a peak of %d", last, peak)
	}
}

type flaky struct {
	fails int
	got   [][]byte
}

func (f *flaky) Write(p []byte) (int, error) {
	f.got = append(f.got, bytes.Clone(p))
	if f.fails > 0 {
		f.fails--
		return 0, alsa.ErrUnderrun
	}
	return len(p), nil
}

func TestAnUnderrunRetriesTheSameAudio(t *testing.T) {
	var underruns atomic.Uint64
	to := &flaky{fails: 2}
	buf := []byte{1, 2, 3, 4}

	if err := Send(context.Background(), to, buf, &underruns); err != nil {
		t.Fatalf("send = %v", err)
	}

	if len(to.got) != 3 {
		t.Fatalf("wrote %d times, want 3", len(to.got))
	}
	for i, got := range to.got {
		if !slices.Equal(got, buf) {
			t.Errorf("attempt %d wrote %v, wanted the same %v", i, got, buf)
		}
	}
	if underruns.Load() != 2 {
		t.Errorf("underruns = %d, want 2", underruns.Load())
	}
}

func TestAWriteThatWorksIsNotCountedOrRepeated(t *testing.T) {
	var underruns atomic.Uint64
	to := &flaky{}

	if err := Send(context.Background(), to, []byte{1, 2}, &underruns); err != nil {
		t.Fatalf("send = %v", err)
	}
	if len(to.got) != 1 {
		t.Errorf("wrote %d times, want 1", len(to.got))
	}
	if underruns.Load() != 0 {
		t.Errorf("underruns = %d, want 0", underruns.Load())
	}
}

type writerFunc func([]byte) (int, error)

func (w writerFunc) Write(p []byte) (int, error) { return w(p) }

func TestRetryingStopsWhenTheContextDoes(t *testing.T) {
	var underruns atomic.Uint64
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dead := writerFunc(func([]byte) (int, error) { return 0, alsa.ErrUnderrun })
	if err := Send(ctx, dead, []byte{1, 2}, &underruns); !errors.Is(err, context.Canceled) {
		t.Errorf("send = %v, want context.Canceled", err)
	}
}

func TestAnythingOtherThanAnUnderrunIsReturned(t *testing.T) {
	var underruns atomic.Uint64
	want := errors.New("device gone")

	err := Send(context.Background(), writerFunc(func([]byte) (int, error) {
		return 0, want
	}), []byte{1, 2}, &underruns)

	if !errors.Is(err, want) {
		t.Errorf("send = %v, want %v", err, want)
	}
	if underruns.Load() != 0 {
		t.Errorf("underruns = %d, want 0", underruns.Load())
	}
}
