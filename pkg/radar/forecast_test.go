package radar

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func TestForecastShotsStartAfterTheNewestScan(t *testing.T) {
	run := time.Date(2026, 10, 12, 13, 0, 0, 0, time.UTC)
	after := time.Date(2026, 10, 12, 15, 25, 0, 0, time.UTC)
	shots := forecastShots(run, after, 7)
	if len(shots) != 8 {
		t.Fatalf("got %d forecast shots, want 8 across two hours", len(shots))
	}
	if want := time.Date(2026, 10, 12, 15, 30, 0, 0, time.UTC); !shots[0].at.Equal(want) {
		t.Errorf("first forecast at %v, want %v", shots[0].at, want)
	}
	if last := shots[len(shots)-1].at; last.After(after.Add(ahead)) {
		t.Errorf("last forecast at %v, past %v", last, after.Add(ahead))
	}
	url := shots[0].url(4, 3, 5)
	if !strings.Contains(url, "REFD-F0150-202610121300/4/3/5.png") {
		t.Errorf("url %q does not name lead 150 of the 13:00 run", url)
	}
}

func TestForecastShotsWithoutARun(t *testing.T) {
	if shots := forecastShots(time.Time{}, time.Now(), 7); shots != nil {
		t.Errorf("got %d shots with no model run", len(shots))
	}
}

func TestFetchOrderNewestScanThenForecastThenHistory(t *testing.T) {
	got := fetchOrder(4, 2)
	want := []int{3, 4, 5, 2, 1, 0}
	if !slices.Equal(got, want) {
		t.Errorf("fetchOrder(4, 2) = %v, want %v", got, want)
	}
}
