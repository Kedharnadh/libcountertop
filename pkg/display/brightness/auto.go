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
	night     schema.Night
	pinned    bool
	rising    func() (float64, bool)
}

func (a *Auto) Rising(share func() (float64, bool)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.rising = share
}

func NewAuto(leastLux float64) *Auto { return &Auto{leastLux: leastLux} }

func (a *Auto) Night(n schema.Night) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.night = n
}

func nightLevel(n schema.Night) (int, bool) {
	switch n {
	case schema.NightOff:
		return 0, true
	case schema.NightDim, schema.NightClock:
		return Floor, true
	}
	return 0, false
}

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

	if a.rising != nil {
		if share, ok := a.rising(); ok {
			lift := min(max(share, 0), 1)
			level := Floor + int(math.Round(lift*float64(max(screen.Backlight, Floor)-Floor)))
			if level != a.applied || !a.pinned {
				a.applied, a.pinned = level, true
				step.Level, step.Apply = level, true
			}
			return step
		}
	}

	dim, dark := nightLevel(a.night)
	if screen.Mode != schema.ModeAuto && !dark {
		if a.pinned {
			a.pinned = false
			step.Level, step.Apply = screen.Backlight, true
		}
		a.applied = 0
		return step
	}

	if screen.Mode == schema.ModeAuto {
		a.held = hold(a.held, a.following, lux)
		want := Brightness(a.held, screen.Backlight)
		if a.shown == 0 {
			a.shown = want
		}
		a.shown = ramp(a.shown, want)
	}
	level := a.shown
	if dark {
		level = dim
	}
	if level != a.applied || dark != a.pinned {
		a.applied, a.pinned = level, dark
		step.Level, step.Apply = level, true
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
