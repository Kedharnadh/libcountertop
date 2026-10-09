package boot

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/gpu"
	"github.com/ygelfand/libcountertop/pkg/display/panel"
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/input/touch"
	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
	"github.com/ygelfand/libcountertop/pkg/say"
)

const (
	sceneFPS  = 61
	viewEvery = 50 * time.Millisecond
	viewWait  = 10 * time.Second
)

type Splash struct {
	Claim    *panel.Claim
	Display  gpu.Display
	Layers   *gpu.Renderer
	Contacts *hook.Hook[touch.Contact]
	Reveal   *Reveal
	Theme    func() theme.Theme
	Version  string
	Progress func() []startup.Progress
	Ready    func() bool
	Blank    func(p *panel.Panel) error
}

func Viewed(ctx context.Context, d gpu.Display) (int, int) {
	for range int(viewWait / viewEvery) {
		fw, fh := d.Native()
		if w, h := d.Orientation().Size(fw, fh); w > 0 && h > 0 {
			return w, h
		}
		if !wait(ctx, viewEvery) {
			break
		}
	}
	return 0, 0
}

func OpenScene(d gpu.Display, w, h int, t theme.Theme, version string) *Scene {
	c := d.Helper()
	if c == nil || c.Err() != nil || w <= 0 || h <= 0 {
		return nil
	}
	nw, nh := d.Native()
	scene, err := NewScene(
		c,
		sceneFPS,
		nw,
		nh,
		w,
		h,
		int(d.Orientation()),
		t,
		version,
		say.T("boot.skip"),
	)
	if err != nil {
		slog.Error("starting boot GUI", "err", err)
		return nil
	}
	return scene
}

func (s Splash) Play(ctx context.Context, w, h int) bool {
	layer, err := s.Layers.Open(w, h)
	if err != nil {
		slog.Error(
			"the boot animation could not start",
			"err",
			err,
			"size",
			fmt.Sprintf("%dx%d", w, h),
		)
	} else {
		defer layer.Close()
		if err := layer.Place(ui.Rect{W: w, H: h}); err != nil {
			slog.Error("placing the boot animation", "err", err)
		}
	}

	var skipped, offering atomic.Bool
	skip := Skip(w, h)
	stopSkip := s.Contacts.Listen(func(c touch.Contact) {
		if c.Phase == touch.Down && offering.Load() && skip.Contains(c.X, c.Y) {
			slog.Info("boot screen skipped")
			skipped.Store(true)
		}
	})
	defer stopSkip()

	scene := OpenScene(s.Display, w, h, s.Theme(), s.Version)
	defer func() {
		if scene != nil {
			scene.Close()
		}
	}()

	fresh := true
	completed := startup.Run(ctx, startup.Hooks{
		Progress: s.Progress,
		Ready:    s.Ready, Skipped: skipped.Load, OfferSkip: offering.Store,
		Draw: func(progress []startup.Progress) {
			if scene == nil {
				return
			}
			if err := scene.Draw(progress, true); err != nil {
				slog.Error("drawing boot progress", "err", err)
				scene.Close()
				scene = nil
			}
		},
		Changed: func(text string) { slog.Info("coming up", "waiting", text) },
		Animate: func(m startup.Moment) bool {
			if layer == nil {
				return false
			}
			if err := s.Reveal.Shade(layer, fresh, w, h, m); err != nil {
				slog.Error("the boot animation stopped", "err", err)
				_ = layer.Close()
				layer = nil
				return false
			}
			fresh = false
			return true
		},
	})
	if !completed {
		return false
	}

	s.Claim.Show(s.Blank)
	time.Sleep(2 * startup.Frame)
	slog.Info("ready")
	return true
}

func wait(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
