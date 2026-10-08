package sound

import (
	"context"
	"sync"
	"testing"
	"time"
)

type counted struct {
	mu   sync.Mutex
	down bool
}

func (b *counted) Stand(down bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.down = down
}

func (b *counted) held() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.down
}

func TestBackgroundPlaysAgainAfterOneSound(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	waitFor(t, d.Claim("one", func(context.Context, *fake) error { return nil }))

	if bg.held() {
		t.Fatal("still standing down after the only sound finished")
	}
}

func TestBackgroundPlaysAgainAfterOneSoundDisplacesAnother(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	first := d.Claim("first", func(ctx context.Context, _ *fake) error {
		<-ctx.Done()
		return nil
	})

	second := d.Claim("second", func(context.Context, *fake) error { return nil })

	waitFor(t, second)
	waitFor(t, first)

	if bg.held() {
		t.Fatal("still standing down after both sounds finished")
	}
}

func TestBackgroundPlaysAgainAfterSilence(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	c := d.Claim("held", func(ctx context.Context, _ *fake) error {
		<-ctx.Done()
		return nil
	})

	time.Sleep(50 * time.Millisecond)
	d.Silence()
	waitFor(t, c)

	if bg.held() {
		t.Fatal("still standing down after being silenced")
	}
}

func TestBackgroundSurvivesAStreamOfSounds(t *testing.T) {
	d := driver()
	bg := &counted{}
	d.Yields(bg)

	for range 20 {
		waitFor(t, d.Claim("one", func(context.Context, *fake) error { return nil }))
	}

	if bg.held() {
		t.Fatal("still standing down after twenty sounds")
	}
}
