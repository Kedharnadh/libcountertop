package visualview

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/display/visual"
)

const shadeEvery = time.Second / 60

// View is the chosen visual across the whole screen. A tap puts it away.
type View struct {
	v       Source
	options Options

	mu      sync.Mutex
	vis     visual.Visual
	kind    visual.Kind
	layer   *gpu.Layer
	area    ui.Rect
	rot     int
	fresh   bool
	broken  bool
	running bool
	stop    context.CancelFunc
	done    chan struct{}

	began   time.Time
	frames  int
	spent   time.Duration
	slowest time.Duration
}

func New(source Source, options Options) *View { return &View{v: source, options: options} }

func (w *View) Covers() bool { return true }

func (w *View) Timeout() time.Duration { return shell.SettingsTimeout }

func (w *View) Keep(width, height int) {
	area := ui.Rect{W: width, H: height}
	kind := w.v.Kind()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.vis == nil || kind != w.kind {
		w.vis, w.kind, w.broken = visual.New(kind), kind, false
		w.close()
	}
	if w.broken {
		return
	}
	if w.layer != nil && (w.area.W != area.W || w.area.H != area.H || !w.layer.Alive()) {
		w.close()
	}
	if w.layer == nil {
		l, err := w.options.Open(area.W, area.H)
		if errors.Is(err, gpu.ErrNoHelper) {
			return
		}
		if err != nil {
			slog.Error("visuals: opening the GPU layer", "err", err)
			w.broken = true
			return
		}
		w.layer, w.fresh, w.area = l, true, ui.Rect{}
	}
	if rot := w.options.Orientation(); w.area != area || w.rot != rot {
		if err := w.layer.Place(area); err != nil {
			if w.layer.Alive() {
				slog.Error("visuals: placing the GPU layer", "err", err)
				w.broken = true
			}
			w.close()
			return
		}
		w.area, w.rot = area, rot
	}
	if !w.running {
		w.running = true
		w.began, w.frames, w.spent, w.slowest = time.Now(), 0, 0, 0
		ctx, cancel := context.WithCancel(context.Background())
		w.stop, w.done = cancel, make(chan struct{})
		go w.run(ctx)
	}
}

func (w *View) close() {
	if w.layer != nil {
		w.layer.Close()
		w.layer = nil
	}
}

func (w *View) run(ctx context.Context) {
	release := w.v.Hold()
	defer func() {
		release()
		w.mu.Lock()
		w.close()
		w.mu.Unlock()
		w.say()
		w.mu.Lock()
		w.running = false
		close(w.done)
		w.mu.Unlock()
	}()

	t := time.NewTicker(shadeEvery)
	defer t.Stop()
	var shaded, last time.Time
	for {
		var now time.Time
		select {
		case <-ctx.Done():
			return
		case now = <-t.C:
		}
		if !w.options.Visible(w) {
			return
		}
		if fps := w.options.MaxFPS(); fps > 0 && now.Sub(shaded) < time.Second/time.Duration(fps)-2*time.Millisecond {
			continue
		}
		shaded = now
		w.shade(now, last)
		last = now
	}
}

func (w *View) shade(now, last time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.layer == nil {
		return
	}
	x := w.v.Input()
	if !last.IsZero() {
		x.Dt = now.Sub(last)
	}
	start := time.Now()
	if err := w.vis.Shade(w.layer, w.fresh, w.area, x); err != nil {
		if w.layer.Alive() {
			slog.Error("visuals: GPU visual failed", "kind", string(w.kind), "err", err)
			w.broken = true
		}
		w.close()
		return
	}
	w.fresh = false
	took := time.Since(start)
	w.frames++
	w.spent += took
	w.slowest = max(w.slowest, took)
}

func (w *View) say() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.frames == 0 {
		return
	}
	held := time.Since(w.began)
	slog.Debug("visual shown",
		"kind", string(w.kind),
		"shaded", fmt.Sprintf("%.1f/s", float64(w.frames)/held.Seconds()),
		"shade", (w.spent / time.Duration(w.frames)).Round(100*time.Microsecond),
		"slowest", w.slowest.Round(100*time.Microsecond),
		"loudest mic", fmt.Sprintf("%.4f", w.v.Loudest()))
}

type Source interface {
	Kind() visual.Kind
	Input() visual.Input
	Hold() func()
	Loudest() float64
}
type Options struct {
	Open        func(int, int) (*gpu.Layer, error)
	Orientation func() int
	Visible     func(shell.View) bool
	MaxFPS      func() int
}

// Close stops drawing and waits for the view's input and GPU resources to be released.
func (w *View) Close() {
	w.mu.Lock()
	stop, done := w.stop, w.done
	w.mu.Unlock()
	if stop != nil {
		stop()
		<-done
	}
}
