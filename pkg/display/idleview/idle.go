package idleview

import (
	"context"
	"log/slog"
	"math"
	"slices"
	"sync"
	"time"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

const check = 200 * time.Millisecond

type Activity func() (since time.Duration, known bool)

type Options struct {
	Settings storage.Section[schema.Idle]
	DeviceID uint32
	Shell    *shell.Shell
	View     *View
	Ready    func() bool
	Activity []Activity
	Free     func() bool
}

type Idle struct {
	options Options

	after, face, place, align, size *esphome.Select
	kinds, sources                  [2]*esphome.Select

	mu   sync.Mutex
	hold *shell.Hold
}

func NewIdle(options Options) *Idle {
	i := &Idle{options: options}
	i.build()
	return i
}

func (i *Idle) Name() string { return "idle" }

func (i *Idle) View() *View { return i.options.View }

func (i *Idle) Entities() []esphome.Entity {
	return []esphome.Entity{
		i.after, i.face, i.place, i.align, i.size,
		i.kinds[0], i.sources[0], i.kinds[1], i.sources[1],
	}
}

func (i *Idle) Restore(c schema.Idle) {
	i.after.Set(c.After.Label())
	i.face.Set(c.Face.Label())
	i.place.Set(c.Position.Label())
	i.align.Set(c.Align.Label())
	i.size.Set(c.Size.Label())
	for n, v := range []schema.IdleVisual{c.First, c.Second} {
		i.kinds[n].Set(KindLabel(v.Kind))
		i.sources[n].Set(v.Source.Label())
	}
}

func (i *Idle) Run(ctx context.Context) error {
	t := time.NewTicker(check)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
		}
		i.step()
	}
}

func (i *Idle) step() {
	if !i.options.Ready() {
		return
	}
	i.covered()
	if due(i.options.Settings.Read().After.After(), i.quiet()) && i.free() {
		i.Show()
	}
}

func (i *Idle) Lock() {
	i.options.Shell.Close()
	i.Show()
}

func (i *Idle) Show() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.hold.Held() {
		return
	}
	i.hold = i.options.Shell.Hold(i.options.View)
}

func (i *Idle) Wake() {
	i.mu.Lock()
	hold := i.hold
	i.mu.Unlock()
	if hold.Held() && i.options.Shell.Visible(i.options.View) {
		hold.Release()
	}
}

func (i *Idle) covered() {
	i.mu.Lock()
	hold := i.hold
	i.mu.Unlock()
	if hold.Held() && !i.options.Shell.Visible(i.options.View) {
		hold.Release()
	}
}

func due(after, since time.Duration) bool { return after > 0 && since > after }

func (i *Idle) quiet() time.Duration {
	since := time.Duration(math.MaxInt64)
	for _, a := range i.options.Activity {
		if d, ok := a(); ok {
			since = min(since, d)
		}
	}
	return since
}

func (i *Idle) free() bool { return !i.options.Shell.Open() && i.options.Free() }

func KindLabel(kind string) string {
	if kind == "" {
		return say.T("idle.none")
	}
	return visual.Kind(kind).Label()
}

func KindLabels() []string {
	out := []string{say.T("idle.none")}
	for _, k := range visual.Built() {
		out = append(out, k.Label())
	}
	return out
}

func kindByLabel(label string) (string, bool) {
	if label == say.T("idle.none") {
		return "", true
	}
	for _, k := range visual.Built() {
		if k.Label() == label {
			return string(k), true
		}
	}
	return "", false
}

func (i *Idle) redraw() {
	i.mu.Lock()
	held := i.hold.Held()
	i.mu.Unlock()
	if held {
		i.options.Shell.Redraw()
	}
}

func (i *Idle) save(what string, change func(*schema.Idle)) {
	if err := i.options.Settings.Update(change); err != nil {
		slog.Error("saving a setting failed", "setting", what, "err", err)
	}
}

func (i *Idle) SetAfter(v schema.Delay) {
	i.after.Set(v.Label())
	i.save("idle_after", func(c *schema.Idle) { c.After = v })
}

func (i *Idle) SetFace(v schema.Face) {
	i.face.Set(v.Label())
	i.save("idle_face", func(c *schema.Idle) { c.Face = v })
	i.redraw()
}

func (i *Idle) SetPosition(v schema.Position) {
	i.place.Set(v.Label())
	i.save("idle_position", func(c *schema.Idle) { c.Position = v })
	i.redraw()
}

func (i *Idle) SetAlign(v schema.Align) {
	i.align.Set(v.Label())
	i.save("idle_align", func(c *schema.Idle) { c.Align = v })
	i.redraw()
}

func (i *Idle) SetSize(v schema.Size) {
	i.size.Set(v.Label())
	i.save("idle_size", func(c *schema.Idle) { c.Size = v })
	i.redraw()
}

func slot(c *schema.Idle, n int) *schema.IdleVisual {
	if n == 0 {
		return &c.First
	}
	return &c.Second
}

func (i *Idle) SetKind(n int, kind string) {
	if kind != "" && !slices.Contains(visual.Built(), visual.Kind(kind)) {
		return
	}
	i.kinds[n].Set(KindLabel(kind))
	i.save("idle_visual", func(c *schema.Idle) { slot(c, n).Kind = kind })
	i.redraw()
}

func (i *Idle) SetSource(n int, s schema.Source) {
	i.sources[n].Set(s.Label())
	i.save("idle_visual_source", func(c *schema.Idle) { slot(c, n).Source = s })
	i.redraw()
}

type labeled interface{ Label() string }

func labels[T labeled](values []T) []string {
	out := make([]string, len(values))
	for n, v := range values {
		out[n] = v.Label()
	}
	return out
}

func choose[T labeled](values []T, set func(T)) func(string) {
	return func(label string) {
		for _, v := range values {
			if v.Label() == label {
				set(v)
				return
			}
		}
	}
}

func (i *Idle) build() {
	sel := func(id, name, icon string, options []string) *esphome.Select {
		return &esphome.Select{
			Base: esphome.Base{ObjectID: id, Name: name, Icon: icon,
				Category: esphome.CategoryConfig, DeviceID: i.options.DeviceID},
			Options: options,
		}
	}

	i.after = sel("idle_after", "Idle screen after", "mdi:timer-outline",
		labels(schema.Delays()))
	i.after.OnCommand = choose(schema.Delays(), i.SetAfter)

	i.face = sel("idle_face", "Idle clock face", "mdi:clock-digital",
		labels(schema.IdleFaces()))
	i.face.OnCommand = choose(schema.IdleFaces(), i.SetFace)

	i.place = sel("idle_position", "Idle clock vertical position", "mdi:align-vertical-center",
		labels(schema.Positions()))
	i.place.OnCommand = choose(schema.Positions(), i.SetPosition)

	i.align = sel("idle_align", "Idle clock horizontal position", "mdi:align-horizontal-center",
		labels(schema.Aligns()))
	i.align.OnCommand = choose(schema.Aligns(), i.SetAlign)

	i.size = sel("idle_size", "Idle clock size", "mdi:resize", labels(schema.Sizes()))
	i.size.OnCommand = choose(schema.Sizes(), i.SetSize)

	for n, id := range []string{"1", "2"} {
		i.kinds[n] = sel("idle_visual_"+id, "Idle visual "+id, "mdi:waveform", KindLabels())
		i.kinds[n].OnCommand = func(l string) {
			if k, ok := kindByLabel(l); ok {
				i.SetKind(n, k)
			}
		}
		i.sources[n] = sel(
			"idle_visual_"+id+"_source",
			"Idle visual "+id+" listens to",
			"mdi:microphone-settings",
			labels(schema.Sources()),
		)
		i.sources[n].OnCommand = choose(
			schema.Sources(),
			func(s schema.Source) { i.SetSource(n, s) },
		)
	}
}
