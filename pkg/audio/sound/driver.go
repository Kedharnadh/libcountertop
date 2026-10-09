package sound

import (
	"context"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/background"
	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Player interface {
	Drain()
	Queued() int
	Chime(s schema.Stream, level float64, notes ...tone.Note)
}

type Background interface {
	Stand(down bool)
}

type Driver[P Player] struct {
	p P

	mu  sync.Mutex
	now *Claim
	bg  Background
	arb *background.Arbiter

	yielded bool
}

func NewDriver[P Player](p P) *Driver[P] { return &Driver[P]{p: p} }

const HardwareTail = 150 * time.Millisecond

const dry = 400 * time.Millisecond

func (d *Driver[P]) Yields(b Background) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.bg = b
}

func (d *Driver[P]) Backgrounds() *background.Arbiter {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.arb == nil {
		d.arb = &background.Arbiter{}
		d.bg = d.arb
	}
	return d.arb
}

func (d *Driver[P]) Claim(name string, play func(ctx context.Context, p P) error) *Claim {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Claim{name: name, cancel: cancel, done: make(chan struct{})}

	d.mu.Lock()
	previous := d.now
	d.now = c
	d.mu.Unlock()

	d.settle()
	previous.preempt(d.p)

	go func() {
		defer close(c.done)
		defer d.release(c)

		if err := play(ctx, d.p); err != nil {
			c.fail(err)
			return
		}
		c.mark(d.await(ctx))
	}()
	return c
}

func (d *Driver[P]) Interject(play func(p P)) { play(d.p) }

func (d *Driver[P]) Chime(s schema.Stream, notes []tone.Note) {
	if d == nil || len(notes) == 0 {
		return
	}
	d.Interject(func(p P) { p.Chime(s, tone.Level, notes...) })
}

func (d *Driver[P]) Silence() {
	d.mu.Lock()
	c := d.now
	d.now = nil
	d.mu.Unlock()

	c.stop(d.p)
	d.p.Drain()
	d.settle()
}

func (d *Driver[P]) Busy() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.now != nil && !d.now.Finished()
}

func (d *Driver[P]) release(c *Claim) {
	d.mu.Lock()
	if d.now != c {
		d.mu.Unlock()
		return
	}
	d.now = nil
	d.mu.Unlock()

	d.settle()
}

func (d *Driver[P]) settle() {
	d.mu.Lock()
	want := d.now != nil
	if want == d.yielded {
		d.mu.Unlock()
		return
	}
	d.yielded = want
	bg := d.bg
	d.mu.Unlock()

	if bg == nil {
		return
	}
	bg.Stand(want)
}

func (d *Driver[P]) await(ctx context.Context) time.Time {
	const tick = 50 * time.Millisecond

	var empty time.Time
	for {
		select {
		case <-ctx.Done():
			return time.Now()
		case <-time.After(tick):
		}

		if d.p.Queued() > 0 {
			empty = time.Time{}
			continue
		}
		if empty.IsZero() {
			empty = time.Now()
		}
		if time.Since(empty) >= dry {
			time.Sleep(HardwareTail)
			return empty
		}
	}
}
