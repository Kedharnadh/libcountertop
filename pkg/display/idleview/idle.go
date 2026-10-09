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
	setting "github.com/ygelfand/libcountertop/pkg/settings"
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

	knobs *setting.Controls[schema.Idle]

	mu   sync.Mutex
	hold *shell.Hold
	woke time.Time
}

func NewIdle(options Options) *Idle {
	i := &Idle{options: options}
	i.knobs = &setting.Controls[schema.Idle]{
		Table:  NewTable(),
		Device: options.DeviceID,
		Read:   options.Settings.Read,
		Save:   i.save,
	}
	return i
}

func (i *Idle) Name() string { return "idle" }

func (i *Idle) View() *View { return i.options.View }

func (i *Idle) Entities() []esphome.Entity { return i.knobs.Entities() }

func (i *Idle) Settings() *setting.Controls[schema.Idle] { return i.knobs }

func (i *Idle) Restore(c schema.Idle) { i.knobs.PublishFrom(c) }

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
	i.woke = time.Now()
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
	i.mu.Lock()
	if !i.woke.IsZero() {
		since = time.Since(i.woke)
	}
	i.mu.Unlock()
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

func (i *Idle) redraw() {
	i.mu.Lock()
	held := i.hold.Held()
	i.mu.Unlock()
	if held {
		i.options.Shell.Redraw()
	}
}

func (i *Idle) save(s Setting, v string) error {
	if err := setting.Store(i.options.Settings, s, v); err != nil {
		return err
	}
	i.redraw()
	return nil
}

func (i *Idle) set(what string, change func(*schema.Idle)) {
	if err := i.options.Settings.Update(change); err != nil {
		slog.Error("saving a setting failed", "setting", what, "err", err)
	}
	i.knobs.Publish()
	i.redraw()
}

func (i *Idle) SetAfter(v schema.Delay) {
	i.set("after", func(c *schema.Idle) { c.After = v })
}

func (i *Idle) Saver() schema.HomeMode { return i.options.Settings.Read().Saver }

func (i *Idle) SetSaver(v schema.HomeMode) {
	i.set("saver", func(c *schema.Idle) { c.Saver = v })
}

func (i *Idle) SetFace(v schema.Face) {
	i.set("face", func(c *schema.Idle) { c.Face = v })
}

func (i *Idle) SetPosition(v schema.Position) {
	i.set("position", func(c *schema.Idle) { c.Position = v })
}

func (i *Idle) SetAlign(v schema.Align) {
	i.set("align", func(c *schema.Idle) { c.Align = v })
}

func (i *Idle) SetSize(v schema.Size) {
	i.set("size", func(c *schema.Idle) { c.Size = v })
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
	i.set("visual", func(c *schema.Idle) { slot(c, n).Kind = kind })
}

func (i *Idle) SetSource(n int, s schema.Source) {
	i.set("visual.source", func(c *schema.Idle) { slot(c, n).Source = s })
}
