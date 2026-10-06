// Package startup coordinates the display boot reveal and progress handover.
package startup

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

const Frame = time.Second / 60

type Progress struct {
	Name, Doing  string
	Done, Failed bool
}
type Moment struct {
	At                   time.Duration
	Trace, Header, Ready float64
}

func Settled(p []Progress) bool {
	for _, r := range p {
		if !r.Done && !r.Failed {
			return false
		}
	}
	return true
}
func Summary(p []Progress) string {
	var b strings.Builder
	for _, r := range p {
		fmt.Fprintf(&b, "%s:%t:%t:%s ", r.Name, r.Done, r.Failed, r.Doing)
	}
	return b.String()
}

// Controller advances the reveal from elapsed time and current component readiness.
// Skip dismisses optional waiting; Ready must still permit handover.
type Controller struct {
	Moment
	last, listed, done, finishing, stepped time.Duration
	shown                                  int
	waited                                 bool
}

func (c *Controller) Tick(at time.Duration, p []Progress, ready, skip bool) {
	dt := at - c.last
	c.last = at
	c.At = at
	if at > 2*time.Second {
		up := 0
		for _, r := range p {
			if r.Done || r.Failed {
				up++
			}
		}
		if c.shown < up && (c.stepped == 0 || at-c.stepped >= 350*time.Millisecond) {
			c.shown++
			c.stepped = at
		}
		want := 1.0
		if len(p) > 0 && !skip {
			want = float64(c.shown) / float64(len(p))
		}
		c.Trace = math.Min(want, c.Trace+dt.Seconds()/6)
	}
	if at >= 3*time.Second {
		c.Header = math.Min(1, c.Header+dt.Seconds()/0.7)
	}
	if c.Header >= 1 && c.listed == 0 {
		c.listed = at
	}
	if c.listed != 0 && !Settled(p) {
		c.waited = true
	}
	if !ready {
		c.done = 0
		c.finishing = 0
		c.Ready = 0
	} else if c.done == 0 {
		c.done = at
	}
	if c.finishing == 0 && c.listed != 0 && c.done != 0 && at-c.done >= 500*time.Millisecond && (!c.waited || at-c.listed >= 3*time.Second) && (Settled(p) || skip) {
		c.finishing = at
	}
	if c.finishing != 0 && c.Trace >= 1 {
		c.Ready = math.Min(1, c.Ready+dt.Seconds()/1.1)
	}
}

type Hooks struct {
	Progress       func() []Progress
	Ready, Skipped func() bool
	OfferSkip      func(bool)
	Draw           func([]Progress)
	Animate        func(Moment) bool
	Changed        func(string)
}

// Run owns the boot clock. Animate reports whether the reveal layer remains usable.
func Run(ctx context.Context, h Hooks) bool {
	began := time.Now()
	ticker := time.NewTicker(Frame)
	defer ticker.Stop()
	var c Controller
	said := "\x00"
	for {
		select {
		case <-ctx.Done():
			return false
		case now := <-ticker.C:
			p := h.Progress()
			c.Tick(now.Sub(began), p, h.Ready(), h.Skipped())
			if c.Header >= 1 {
				h.OfferSkip(!Settled(p))
				if s := Summary(p); s != said {
					said = s
					h.Draw(p)
					if h.Changed != nil {
						h.Changed(s)
					}
				}
			}
			animated := h.Animate(c.Moment)
			if c.Ready >= 1 || (!animated && c.finishing != 0) {
				return true
			}
		}
	}
}

// Settled reports whether the component no longer delays boot.
func (p Progress) Settled() bool { return p.Done || p.Failed }

// Display returns an ordered copy with consistent names for equivalent board components.
// Components absent on a board are omitted by its registry rather than shown as pretend work.
func Display(progress []Progress) []Progress {
	out := slices.Clone(progress)
	order := map[string]int{"display": 1, "touch": 2, "touchscreen": 2, "buttons": 3, "speaker": 4, "amplifier": 5, "capture": 6, "microphones": 6, "wifi": 7, "dhcp": 8, "bluetooth": 9, "drm": 10, "chromecast": 11, "api": 12, "homeassistant": 13}
	slices.SortStableFunc(out, func(a, b Progress) int {
		ai, ok := order[a.Name]
		if !ok {
			ai = 100
		}
		bi, ok := order[b.Name]
		if !ok {
			bi = 100
		}
		return ai - bi
	})
	names := map[string]string{"display": "Display", "touch": "Touchscreen", "touchscreen": "Touchscreen", "buttons": "Buttons", "speaker": "Speaker", "amplifier": "Amplifier", "capture": "Microphones", "microphones": "Microphones", "wifi": "Wi-Fi", "dhcp": "Network address", "bluetooth": "Bluetooth", "drm": "Protected media", "chromecast": "Chromecast", "api": "Home Assistant API", "homeassistant": "Home Assistant"}
	for i := range out {
		if name, ok := names[out[i].Name]; ok {
			out[i].Name = name
		}
	}
	return out
}
