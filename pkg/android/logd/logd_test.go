package logd

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestPriorityMapsLevels(t *testing.T) {
	cases := []struct {
		level slog.Level
		want  byte
	}{
		{slog.LevelDebug, 3},
		{slog.LevelInfo, 4},
		{slog.LevelWarn, 5},
		{slog.LevelError, 6},
	}
	for _, c := range cases {
		if got := Priority(c.level); got != c.want {
			t.Errorf("Priority(%v) = %d, want %d", c.level, got, c.want)
		}
		if got := Level(c.want); got != c.level {
			t.Errorf("Level(%d) = %v, want %v", c.want, got, c.level)
		}
	}
}

func TestHandleFormatsAttrs(t *testing.T) {
	var buf bytes.Buffer
	h := &Handler{fallback: &buf, mu: new(sync.Mutex)}
	log := slog.New(h)

	log.Info("playback path up", "output", "speaker", "step", 15)

	got := buf.String()
	for _, want := range []string{"I ", "playback path up", "output=speaker", "step=15"} {
		if !strings.Contains(got, want) {
			t.Errorf("output %q is missing %q", got, want)
		}
	}
}

func TestHandleQuotesValuesWithSpaces(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(&Handler{fallback: &buf, mu: new(sync.Mutex)})

	log.Error("mixer write failed", "control", "HP Driver Gain Volume")

	if got := buf.String(); !strings.Contains(got, `control="HP Driver Gain Volume"`) {
		t.Errorf("output %q did not quote the value", got)
	}
}

func TestDebugIsDropped(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(&Handler{fallback: &buf, mu: new(sync.Mutex)})

	log.Debug("noise")

	if buf.Len() != 0 {
		t.Errorf("debug produced %q, want nothing", buf.String())
	}
}

func TestLevelOptionMovesTheThreshold(t *testing.T) {
	var buf bytes.Buffer
	var lv slog.LevelVar
	log := slog.New(&Handler{fallback: &buf, opts: Options{Level: &lv}, mu: new(sync.Mutex)})

	lv.Set(slog.LevelDebug)
	log.Debug("noise")

	if !strings.Contains(buf.String(), "D noise") {
		t.Errorf("output %q, want the debug line", buf.String())
	}
}

func TestUptimePrefixesTheLine(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{Uptime: func() float64 { return 12.5 }}
	slog.New(&Handler{fallback: &buf, opts: opts, mu: new(sync.Mutex)}).Info("up")

	if got, want := buf.String(), "I [   12.50] up\n"; got != want {
		t.Errorf("output %q, want %q", got, want)
	}
}

func TestStreamCountsDroppedLines(t *testing.T) {
	s := NewStream(1)
	log := slog.New(&Handler{opts: Options{Stream: s}, mu: new(sync.Mutex)})

	log.Info("one")
	log.Info("two")
	log.Info("three")

	l := <-s.Lines()
	if l.Text != "three" {
		t.Errorf("text = %q, want three", l.Text)
	}
	if l.Dropped != 1 {
		t.Errorf("dropped = %d, want 1", l.Dropped)
	}
}
