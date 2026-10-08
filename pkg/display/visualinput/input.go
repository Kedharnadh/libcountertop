package visualinput

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/audio/analysis"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const (
	drainEvery = 20 * time.Millisecond
	replyFloor = 0.02

	LabelMost = 32
)

type Visuals struct {
	lifecycle sync.Mutex
	workers   sync.WaitGroup
	options   Options
	mic, spk  *stereo
	heard     loudest
	ring      *analysis.Ring

	mu    sync.Mutex
	holds int
	stop  context.CancelFunc
	began time.Time
	last  time.Time

	listen func() (<-chan []int16, func())
	muted  func() bool
	tap    func(Tap)

	kind   *esphome.Select
	chosen visual.Kind
	label  *esphome.Text
	named  string
}

func (v *Visuals) Name() string { return "visuals" }

func (v *Visuals) Entities() []esphome.Entity { return []esphome.Entity{v.kind, v.label} }

func (v *Visuals) Restore(cfg config.Visual) {
	k := visual.Kind(cfg.Kind)
	if !slices.Contains(visual.Built(), k) {
		k = visual.Kind(config.DefaultVisual)
	}
	v.mu.Lock()
	v.chosen = k
	v.named = cfg.Label
	v.mu.Unlock()
	v.kind.Set(k.Label())
	v.label.Set(cfg.Label)
	visual.SetSeed(cfg.Seed)
}

func (v *Visuals) SetMaxFPS(fps int) {
	if !slices.Contains(config.MaxFPSSteps, fps) {
		fps = config.DefaultMaxFPS
	}
	if err := v.options.Settings.Update(func(c *config.Visual) { c.MaxFPS = fps }); err != nil {
		slog.Error("saving the visual frame cap failed", "err", err)
	}
}

func (v *Visuals) SetSeed(n int) {
	n = max(0, min(n, visual.SeedMost))
	if err := v.options.Settings.Update(func(c *config.Visual) { c.Seed = n }); err != nil {
		slog.Error("saving the visual seed failed", "err", err)
	}
	v.Restore(v.options.Settings.Read())
}

func (v *Visuals) SetLabel(s string) {
	r := []rune(strings.TrimSpace(s))
	if len(r) > LabelMost {
		r = r[:LabelMost]
	}
	if err := v.options.Settings.Update(
		func(c *config.Visual) { c.Label = string(r) },
	); err != nil {
		slog.Error("saving a setting failed", "setting", v.label.ObjectID, "err", err)
	}
	v.Restore(v.options.Settings.Read())
}

func (v *Visuals) Kind() visual.Kind {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.chosen
}

func (v *Visuals) SetKind(k visual.Kind) {
	if err := v.options.Settings.Update(func(c *config.Visual) { c.Kind = string(k) }); err != nil {
		slog.Error("saving a setting failed", "setting", v.kind.ObjectID, "err", err)
	}
	v.Restore(v.options.Settings.Read())
}

func (v *Visuals) build() {
	v.chosen = visual.Kind(config.DefaultVisual)
	v.kind = &esphome.Select{
		Base: esphome.Base{ObjectID: "visual", Name: "Visual", Icon: "mdi:waveform",
			Category: esphome.CategoryConfig, DeviceID: v.options.DeviceID},
		Options: labels(visual.Built()),
	}
	v.kind.OnCommand = func(label string) {
		if k, ok := byLabel(visual.Built(), label); ok {
			v.SetKind(k)
		}
	}
	v.named = config.DefaultLabel
	v.label = &esphome.Text{
		Base: esphome.Base{ObjectID: "visual_label", Name: "Visual label", Icon: "mdi:format-text",
			Category: esphome.CategoryConfig, DeviceID: v.options.DeviceID},
		MaxLength: LabelMost,
	}
	v.label.OnCommand = v.SetLabel
}

func New(options Options) *Visuals {
	v := &Visuals{
		options: options,
		mic:     newStereo(options.MicRate, liftGain(options.Lift)),
		spk:     newStereo(options.SpeakerRate, 1),
		ring:    analysis.NewRing(options.SpeakerRate * options.SpeakerChannels / 5),
		listen:  options.Listen, muted: options.Muted, tap: options.Tap,
	}
	v.build()
	return v
}

func (v *Visuals) Hold() (release func()) {
	v.lifecycle.Lock()
	defer v.lifecycle.Unlock()
	v.mu.Lock()
	v.holds++
	if v.holds == 1 {
		ctx, stop := context.WithCancel(context.Background())
		v.stop, v.last = stop, time.Now()
		v.mic.reset()
		v.spk.reset()
		v.heard.reset()
		v.tap(v.ring)
		v.workers.Add(2)
		go func() { defer v.workers.Done(); v.drain(ctx) }()
		go func() { defer v.workers.Done(); v.hear(ctx) }()
	}
	v.mu.Unlock()

	var done sync.Once
	return func() { done.Do(v.release) }
}

func (v *Visuals) release() {
	v.lifecycle.Lock()
	defer v.lifecycle.Unlock()
	v.mu.Lock()
	if v.holds--; v.holds > 0 {
		v.mu.Unlock()
		return
	}
	v.tap(nil)
	v.stop()
	v.stop = nil
	v.mu.Unlock()
	v.workers.Wait()
}

func (v *Visuals) SetLift(db int) { v.mic.setGain(liftGain(min(max(db, 0), LiftMax))) }

func (v *Visuals) Loudest() float64 {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.heard.most
}

func (v *Visuals) Input() visual.Input {
	v.mu.Lock()
	now := time.Now()
	if v.began.IsZero() {
		v.began, v.last = now, now
	}
	x := visual.Input{Now: now.Sub(v.began), Dt: now.Sub(v.last), Label: v.named}
	v.last = now
	v.mu.Unlock()

	x.Mic, x.Speaker = v.mic.mix.Latest(), v.spk.mix.Latest()
	x.MicLeft, x.MicRight = v.mic.left.Latest(), v.mic.right.Latest()
	x.SpeakerLeft, x.SpeakerRight = v.spk.left.Latest(), v.spk.right.Latest()
	x.Replying = x.Speaker.Level > replyFloor && x.Speaker.Level >= x.Mic.Level
	return x
}

func (v *Visuals) drain(ctx context.Context) {
	t := time.NewTicker(drainEvery)
	defer t.Stop()
	var buf []int16
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		buf = v.ring.Drain(buf)
		v.spk.write(buf)
	}
}

func (v *Visuals) hear(ctx context.Context) {
	frames, stop := v.listen()
	defer stop()
	var silent []int16
	for {
		select {
		case <-ctx.Done():
			return
		case f, ok := <-frames:
			if !ok {
				return
			}
			if v.muted() {
				if cap(silent) < len(f) {
					silent = make([]int16, len(f))
				}
				f = silent[:len(f)]
			}
			v.mu.Lock()
			v.heard.hear(f)
			v.mu.Unlock()
			v.mic.write(f)
		}
	}
}

type Tap interface{ Offer([]int16) }
type Options struct {
	MicRate, SpeakerRate, SpeakerChannels, Lift int
	Settings                                    storage.Section[config.Visual]
	DeviceID                                    uint32
	Listen                                      func() (<-chan []int16, func())
	Muted                                       func() bool
	Tap                                         func(Tap)
}

func labels(values []visual.Kind) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = v.Label()
	}
	return out
}
func byLabel(values []visual.Kind, label string) (visual.Kind, bool) {
	for _, v := range values {
		if v.Label() == label {
			return v, true
		}
	}
	return "", false
}
