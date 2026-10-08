package capture

import (
	"slices"
	"testing"
	"time"
)

func TestHistoryKeepsTheNewestSamples(t *testing.T) {
	h := NewHistory(1000, 10*time.Millisecond)

	if got := h.Recent(5 * time.Millisecond); len(got) != 0 {
		t.Fatalf("an empty history gave %v", got)
	}

	h.Add([]int16{1, 2, 3})
	if got := h.Recent(5 * time.Millisecond); !slices.Equal(got, []int16{1, 2, 3}) {
		t.Errorf("before filling: %v, want 1 2 3", got)
	}

	h.Add([]int16{4, 5, 6, 7, 8, 9, 10, 11, 12})
	if got := h.Recent(4 * time.Millisecond); !slices.Equal(got, []int16{9, 10, 11, 12}) {
		t.Errorf("after wrapping: %v, want 9 10 11 12", got)
	}
	if got := h.Recent(time.Second); len(got) != 10 || got[0] != 3 || got[9] != 12 {
		t.Errorf("asking past the span: %v, want 3 through 12", got)
	}
}
