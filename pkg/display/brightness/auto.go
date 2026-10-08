package brightness

import (
	"math"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const (
	settle = 0.25

	change = 0.05
	stale  = 5 * time.Minute
)

type Step struct {
	Ambient float64
	Report  bool

	Level int
	Apply bool
}

type Auto struct {
	leastLux float64

	mu        sync.Mutex
	following float64
	followed  bool
	held      float64
	shown     int
	applied   int
	told      Told
}

func NewAuto(leastLux float64) *Auto { return &Auto{leastLux: leastLux} }

func (a *Auto) Ambient() (float64, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.following, a.followed
}

func (a *Auto) Follow(lux float64, screen schema.Screen) Step {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.followed {
		a.following, a.followed, a.held = lux, true, lux
	}
	a.following += settle * (lux - a.following)
	step := Step{Ambient: a.following, Report: a.told.Worth(a.following, a.leastLux)}

	if screen.Mode != schema.ModeAuto {
		a.applied = 0
		return step
	}

	a.held = hold(a.held, a.following, lux)
	want := Brightness(a.held, screen.Backlight)
	if a.shown == 0 {
		a.shown = want
	}
	a.shown = ramp(a.shown, want)
	if a.shown != a.applied {
		a.applied = a.shown
		step.Level, step.Apply = a.shown, true
	}
	return step
}

type Told struct {
	value float64
	at    time.Time
	sent  bool
}

func (t *Told) Worth(v, least float64) bool {
	now := time.Now()

	switch {
	case !t.sent, now.Sub(t.at) >= stale:
	case math.Abs(v-t.value) >= max(least, math.Abs(t.value)*change):
	default:
		return false
	}

	t.value, t.at, t.sent = v, now, true
	return true
}
