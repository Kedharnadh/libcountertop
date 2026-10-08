package visualinput

import (
	"math"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type rig struct {
	frames  chan []int16
	stopped atomic.Int32
	muted   atomic.Bool

	mu  sync.Mutex
	tap Tap
}

func newRig(t *testing.T) (*rig, *Visuals) {
	t.Helper()
	st, err := storage.Load(
		filepath.Join(t.TempDir(), "state.json"),
		schema.DefaultVisualSettings(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{frames: make(chan []int16, 64)}
	v := New(Options{
		Settings: storage.Bind(
			func() *storage.Store[schema.Visual] { return st },
			func(c *schema.Visual) *schema.Visual { return c },
		),
		MicRate:         16000,
		MicChannels:     2,
		SpeakerRate:     48000,
		SpeakerChannels: 2,
		Listen: func() (<-chan []int16, func()) {
			return r.frames, func() { r.stopped.Add(1) }
		},
		Muted: r.muted.Load,
		Tap:   func(t Tap) { r.mu.Lock(); r.tap = t; r.mu.Unlock() },
	})
	return r, v
}

func (r *rig) tapped() Tap {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tap
}

func tone(n int, amp float64) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = int16(amp * math.MaxInt16 * math.Sin(2*math.Pi*300*float64(i)/16000))
	}
	return out
}

func eventually(t *testing.T, what string, ok func() bool) {
	t.Helper()
	until := time.Now().Add(2 * time.Second)
	for !ok() {
		if time.Now().After(until) {
			t.Fatalf("never: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestTheTapsRunOnlyWhileHeld(t *testing.T) {
	r, v := newRig(t)
	if r.tapped() != nil {
		t.Fatal("tapped before anything held")
	}

	a := v.Hold()
	b := v.Hold()
	if r.tapped() == nil {
		t.Fatal("not tapped while held")
	}
	a()
	a()
	if r.tapped() == nil {
		t.Fatal("one release of two stopped the tap")
	}
	b()
	if r.tapped() != nil {
		t.Error("still tapped after the last release")
	}
	eventually(t, "the mic listener stopped", func() bool { return r.stopped.Load() == 1 })
}

func TestTheMicIsHeard(t *testing.T) {
	r, v := newRig(t)
	defer v.Hold()()

	for range 20 {
		r.frames <- tone(320, 0.5)
	}
	eventually(t, "the mic level rose", func() bool { return v.Input().Mic.Level > 0.1 })
}

func TestAMutedMicIsNotHeard(t *testing.T) {
	r, v := newRig(t)
	r.muted.Store(true)
	defer v.Hold()()

	for range 20 {
		r.frames <- tone(320, 0.5)
	}
	time.Sleep(100 * time.Millisecond)
	if got := v.Input().Mic.Level; got != 0 {
		t.Errorf("a muted mic read level %v", got)
	}
}

func TestMutingDropsTheLevelInsteadOfHoldingIt(t *testing.T) {
	r, v := newRig(t)
	defer v.Hold()()

	for range 20 {
		r.frames <- tone(320, 0.5)
	}
	eventually(t, "the mic level rose", func() bool { return v.Input().Mic.Level > 0.1 })

	r.muted.Store(true)
	for range 200 {
		r.frames <- tone(320, 0.5)
	}
	eventually(t, "the level fell once muted", func() bool { return v.Input().Mic.Level < 0.01 })
}

func TestTheSpeakerIsHeardAndCountsAsReplying(t *testing.T) {
	r, v := newRig(t)
	defer v.Hold()()

	stereo := make([]int16, 0, 9600)
	for _, s := range tone(4800, 0.5) {
		stereo = append(stereo, s, s)
	}
	for range 5 {
		r.tapped().Offer(stereo)
		time.Sleep(25 * time.Millisecond)
	}
	eventually(t, "the speaker level rose", func() bool {
		x := v.Input()
		return x.Speaker.Level > 0.1 && x.Replying
	})
}

func TestRestoreFallsBackFromAnUnbuiltVisual(t *testing.T) {
	_, v := newRig(t)
	kind := func() *esphome.Select { return v.Entities()[0].(*esphome.Select) }
	for _, saved := range []string{"no-such-visual", "pac-man"} {
		cfg := schema.DefaultVisualSettings()
		cfg.Kind = saved
		v.Restore(cfg)
		if v.Kind() != visual.Kind(schema.DefaultVisual) {
			t.Errorf("restoring %q chose %q, want %q", saved, v.Kind(), schema.DefaultVisual)
		}
		if kind().Get() != v.Kind().Label() {
			t.Errorf("restoring %q left the select on %q", saved, kind().Get())
		}
	}

	cfg := schema.DefaultVisualSettings()
	cfg.Kind = string(visual.LCARS)
	v.Restore(cfg)
	if v.Kind() != visual.LCARS || kind().Get() != visual.LCARS.Label() {
		t.Errorf("restoring lcars gave %q / %q", v.Kind(), kind().Get())
	}
}

func TestTheSelectOffersEveryBuiltVisual(t *testing.T) {
	_, v := newRig(t)
	options := v.Entities()[0].(*esphome.Select).Options
	if len(options) != len(visual.Built()) {
		t.Fatalf("%d options for %d visuals", len(options), len(visual.Built()))
	}
	seen := map[string]bool{}
	for _, o := range options {
		if seen[o] {
			t.Errorf("%q offered twice", o)
		}
		seen[o] = true
	}
}
