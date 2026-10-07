package camerapreview

import (
	"context"
	"errors"
	"github.com/ygelfand/libcountertop/pkg/camera/helper"
	"github.com/ygelfand/libcountertop/pkg/camera/session"
	"io"
	"log/slog"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/surface"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

const (
	camIdle  = 750 * time.Millisecond
	camHold  = 5 * time.Second
	camLayer = 900
	camZ     = 6
)

type Preview struct {
	options Options
	left    chan shell.Change
	listen  sync.Once
	cancel  context.CancelFunc
	mu      sync.Mutex
	pages   []*shell.Page
	want    time.Time
	live    bool
	shown   bool
	bad     time.Time
	said    string
	box     ui.Rect
}

type Options struct {
	Shell       *shell.Shell
	Helper      func() *surface.Client
	Select      func() (index, width, height int)
	Join        func(int) (*session.Session, <-chan helper.Frame, error)
	Leave       func(*session.Session, <-chan helper.Frame)
	Orientation func() int
	Place       func(ui.Rect, int, int) (float32, float32, surface.Matrix)
}

func New(options Options) *Preview {
	return &Preview{options: options, left: make(chan shell.Change, 1)}
}

func (p *Preview) Page(page *shell.Page) *shell.Page {
	p.listen.Do(func() {
		p.options.Shell.Changed.Listen(func(c shell.Change) {
			if !p.onScreen() {
				p.mu.Lock()
				cancel := p.cancel
				p.mu.Unlock()
				if cancel != nil {
					cancel()
				}
				select {
				case p.left <- c:
				default:
				}
			}
		})
	})
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pages = append(p.pages, page)
	return page
}

func (p *Preview) onScreen() bool {
	p.mu.Lock()
	pages := append([]*shell.Page(nil), p.pages...)
	p.mu.Unlock()
	for _, page := range pages {
		if p.options.Shell.Visible(page) {
			return true
		}
	}
	return false
}

func (p *Preview) Want(box ui.Rect) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.want, p.box = time.Now(), box
	if !p.live && time.Since(p.bad) > camHold {
		p.live = true
		ctx, cancel := context.WithCancel(context.Background())
		p.cancel = cancel
		go p.run(ctx)
	}
	return p.shown
}

func (p *Preview) watched() (ui.Rect, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.box, time.Since(p.want) < camIdle
}

func (p *Preview) run(ctx context.Context) {
	err := p.play(ctx)
	p.mu.Lock()
	p.live, p.shown = false, false
	if errors.Is(err, session.ErrRestarted) || errors.Is(err, context.Canceled) {
		err = nil
	}
	if err != nil {
		p.bad = time.Now()
		if err.Error() != p.said {
			slog.Warn("the camera preview stopped", "err", err)
		}
		p.said = err.Error()
	}
	p.mu.Unlock()
	p.options.Shell.Redraw()
}

func (p *Preview) place(c *surface.Client, box ui.Rect, w, h int, visible bool) error {
	fit := Fit(w, h, box)
	x, y, m := p.options.Place(fit, w, h)
	return c.VideoPlace(camLayer, x, y, m, camZ, visible)
}

func (p *Preview) play(ctx context.Context) error {
	c := p.options.Helper()
	if c == nil {
		return nil
	}
	at, width, height := p.options.Select()
	if at < 0 || width <= 0 || height <= 0 {
		return errors.New("camera preview has no stream")
	}
	s, frames, err := p.options.Join(at)
	if err != nil {
		return err
	}
	defer p.options.Leave(s, frames)

	if err := c.VideoOpen(camLayer, surface.AVC, width, height, camZ, 0, ""); err != nil {
		return err
	}
	defer c.VideoClose(camLayer)

	var config []byte
	keyed := false
	placed := ui.Rect{}
	orient := p.options.Orientation()
	check := time.NewTicker(250 * time.Millisecond)
	defer check.Stop()
	select {
	case <-p.left:
	default:
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-s.Done():
			return s.Err()
		case <-p.left:
			return nil
		case <-check.C:
			box, watched := p.watched()
			if !watched {
				return nil
			}
			p.options.Shell.Redraw()
			if o := p.options.Orientation(); keyed && (box != placed || o != orient) {
				placed, orient = box, o
				if err := p.place(c, box, width, height, true); err != nil {
					return err
				}
			}
		case f, ok := <-frames:
			if !ok {
				return io.EOF
			}
			if f.Config {
				config = f.Data
				continue
			}
			data := f.Data
			if !keyed {
				if !f.Key {
					continue
				}
				data = append(append([]byte{}, config...), f.Data...)
			}
			if err := p.feed(ctx, c, f.PTS, data, s.Done()); err != nil {
				return err
			}
			if err := c.VideoClock(camLayer, f.PTS-camLag, true); err != nil {
				return err
			}
			if !keyed {
				keyed = true
				placed, _ = p.watched()
				if err := p.place(c, placed, width, height, true); err != nil {
					return err
				}
				p.mu.Lock()
				p.shown = true
				p.mu.Unlock()
				p.options.Shell.Redraw()
			}
		}
	}
}

const (
	camRetry = 20 * time.Millisecond
	camLag   = 200 * time.Millisecond
)

func (p *Preview) feed(ctx context.Context, c *surface.Client, pts time.Duration, data []byte, done <-chan struct{}) error {
	for {
		_, err := c.VideoSample(camLayer, pts, 0, data)
		if !errors.Is(err, surface.Full) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return nil
		case <-time.After(camRetry):
			if _, watched := p.watched(); !watched {
				return context.Canceled
			}
		}
	}
}

func Fit(w, h int, box ui.Rect) ui.Rect {
	dw, dh := box.W, box.W*h/max(1, w)
	if dh > box.H {
		dw, dh = box.H*w/max(1, h), box.H
	}
	return ui.Rect{X: box.X + (box.W-dw)/2, Y: box.Y + (box.H-dh)/2, W: dw, H: dh}
}
