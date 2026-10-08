package sound

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
)

type fake struct{}

func (*fake) Drain()                      {}
func (*fake) Queued() int                 { return 0 }
func (*fake) Chime(float64, ...tone.Note) {}

func driver() *Driver[*fake] { return NewDriver(&fake{}) }

func waitFor(t *testing.T, c *Claim) {
	t.Helper()
	select {
	case <-c.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("the claim never finished")
	}
}

func TestDriverPlays(t *testing.T) {
	d := driver()

	var ran atomic.Bool
	c := d.Claim("test", func(context.Context, *fake) error {
		ran.Store(true)
		return nil
	})

	waitFor(t, c)
	if !ran.Load() {
		t.Error("the errand never ran")
	}
	if c.Stopped() {
		t.Error("a claim that finished reports being stopped")
	}
}

func TestClaimTakesOverFromWhatWasPlaying(t *testing.T) {
	d := driver()

	first := d.Claim("first", func(ctx context.Context, _ *fake) error {
		<-ctx.Done()
		return nil
	})

	second := d.Claim("second", func(context.Context, *fake) error { return nil })
	waitFor(t, second)

	if !first.Stopped() {
		t.Error("the first claim was left holding the speaker")
	}
	if second.Stopped() {
		t.Error("the second claim was stopped by taking over")
	}
}

func TestSilenceStopsAClaimBeforeItPlays(t *testing.T) {
	d := driver()

	var played atomic.Bool
	c := d.Claim("fetch", func(ctx context.Context, p *fake) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
		played.Store(true)
		return nil
	})

	time.Sleep(20 * time.Millisecond)
	d.Silence()
	waitFor(t, c)

	if played.Load() {
		t.Error("the errand played after being silenced")
	}
	if !c.Stopped() {
		t.Error("a silenced claim does not report being stopped")
	}
	if d.Busy() {
		t.Error("the speaker is still busy after being silenced")
	}
}

func TestSilenceWithNothingPlaying(t *testing.T) {
	d := driver()
	d.Silence()

	if d.Busy() {
		t.Error("an idle speaker reports being busy")
	}
}

func TestClaimFailureIsKept(t *testing.T) {
	d := driver()
	want := errors.New("no such reply")

	c := d.Claim("broken", func(context.Context, *fake) error { return want })
	waitFor(t, c)

	if !errors.Is(c.Err(), want) {
		t.Errorf("Err = %v, want %v", c.Err(), want)
	}
}

func TestAnErrandThatWillNotStopDoesNotBlockForever(t *testing.T) {
	d := driver()

	d.Claim("stuck", func(context.Context, *fake) error {
		time.Sleep(3 * time.Second)
		return nil
	})

	start := time.Now()
	next := d.Claim("next", func(context.Context, *fake) error { return nil })
	waitFor(t, next)

	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("waited %s for a stuck errand", took.Round(time.Millisecond))
	}
}
