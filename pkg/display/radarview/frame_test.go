package radarview

import (
	"slices"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/radar"
)

func TestAPickedFrameStaysPutWhileFramesArrive(t *testing.T) {
	base := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	at := func(minutes int) time.Time { return base.Add(time.Duration(minutes) * time.Minute) }
	arrival := []int{120, 135, 150, 165, 110, 100, 90, 80, 70}
	pick := at(80)

	var loaded []radar.Frame
	for _, m := range arrival {
		loaded = append(loaded, radar.Frame{At: at(m)})
		slices.SortFunc(loaded, func(a, b radar.Frame) int { return a.At.Compare(b.At) })
		got := loaded[frame(loaded, 0, true, pick, time.Time{})].At
		want := loaded[nearest(loaded, pick)].At
		if !got.Equal(want) {
			t.Fatalf(
				"after %d frames showing %v, want the nearest to the pick, %v",
				len(loaded),
				got,
				want,
			)
		}
	}
	if got := loaded[frame(loaded, 0, true, pick, time.Time{})].At; !got.Equal(pick) {
		t.Errorf("with every frame loaded showing %v, want the picked %v", got, pick)
	}
}

func TestNoPickShowsNow(t *testing.T) {
	frames := []radar.Frame{{}, {}, {}}
	if got := frame(frames, 1, true, time.Time{}, time.Time{}); got != 1 {
		t.Errorf("paused with no pick showed frame %d, want now (1)", got)
	}
}

func TestClosestSnapsToAPlannedTime(t *testing.T) {
	base := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	times := []time.Time{base, base.Add(10 * time.Minute), base.Add(25 * time.Minute)}
	if got := closest(times, base.Add(19*time.Minute)); !got.Equal(times[2]) {
		t.Errorf("closest = %v, want %v", got, times[2])
	}
}
