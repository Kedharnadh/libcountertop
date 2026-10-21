package widgets

import (
	"log/slog"
	"sync"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"
)

const (
	slideTime  = 280 * time.Millisecond
	flickSpeed = 300
	jumpShare  = 0.35
	commitPart = 0.25
	edgeGive   = 0.3
)

type Slide struct {
	Offset float32
	Other  int
}

func (s Slide) Moving() bool { return s.Offset != 0 && s.Other >= 0 }

func (s Slide) OtherShift(width float32) float32 {
	if s.Offset < 0 {
		return s.Offset + width
	}
	return s.Offset - width
}

type Pager struct {
	id    string
	mu    sync.Mutex
	slide Slide
	held  Drag
}

func NewPager(id string) *Pager { return &Pager{id: id, slide: Slide{Other: -1}} }

func (p *Pager) Now() Slide {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.slide
}

func (p *Pager) Drag(
	w *gogui.Window,
	dx, width float32,
	neighbour func(by int) int,
	turn func(to int) (from int),
) {
	if !p.held.Active() {
		p.held.Begin(w, func(w *gogui.Window, velocity float32) {
			p.release(w, width, velocity, turn)
		})
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.slide.Offset == 0 {
		slog.Debug("pager drag", "pager", p.id, "dx", dx, "width", width)
	}
	p.slide.Offset += dx
	by := 1
	if p.slide.Offset > 0 {
		by = -1
	}
	p.slide.Other = neighbour(by)
	if p.slide.Other < 0 {
		p.slide.Offset -= dx * (1 - edgeGive)
	}
	p.slide.Offset = max(min(p.slide.Offset, width), -width)
}

func (p *Pager) Lift(w *gogui.Window, velocity float32) { p.held.End(w, velocity) }

func (p *Pager) release(w *gogui.Window, width, velocity float32, turn func(to int) (from int)) {
	p.mu.Lock()
	s := p.slide
	speed := velocity
	if speed < 0 {
		speed = -speed
	}
	far := s.Offset
	if far < 0 {
		far = -far
	}
	commit := s.Other >= 0 && (far > width*commitPart || speed > flickSpeed) &&
		(velocity == 0 || (velocity < 0) == (s.Offset < 0))
	p.mu.Unlock()
	slog.Debug("pager release", "pager", p.id, "offset", s.Offset, "other", s.Other,
		"velocity", velocity, "width", width, "commit", commit, "window", w != nil)
	if commit {
		old := turn(s.Other)
		p.mu.Lock()
		p.slide.Other = old
		p.slide.Offset = s.OtherShift(width)
		p.mu.Unlock()
	}
	p.settle(w)
}

func (p *Pager) Jump(w *gogui.Window, from, to int, width float32) {
	if from == to || w == nil {
		return
	}
	p.mu.Lock()
	p.slide.Other = from
	p.slide.Offset = width * jumpShare
	if to < from {
		p.slide.Offset = -p.slide.Offset
	}
	p.mu.Unlock()
	p.settle(w)
}

func (p *Pager) Slid(w *gogui.Window, from, to int, forward bool, width float32) {
	if from == to || w == nil {
		return
	}
	p.mu.Lock()
	p.slide.Other = from
	p.slide.Offset = width
	if !forward {
		p.slide.Offset = -width
	}
	p.mu.Unlock()
	p.settle(w)
}

func (p *Pager) settle(w *gogui.Window) {
	if w == nil {
		p.mu.Lock()
		p.slide = Slide{Other: -1}
		p.mu.Unlock()
		return
	}
	p.mu.Lock()
	from := p.slide.Offset
	p.mu.Unlock()
	tw := gogui.NewTweenAnimation(p.id, from, 0, func(at float32, w *gogui.Window) {
		p.mu.Lock()
		p.slide.Offset = at
		p.mu.Unlock()
	})
	tw.Duration = slideTime
	tw.OnDone = func(w *gogui.Window) {
		p.mu.Lock()
		left := p.slide.Offset
		p.slide = Slide{Other: -1}
		p.mu.Unlock()
		slog.Debug("pager settled", "pager", p.id, "left", left)
		w.InvalidateLayout()
	}
	w.AnimationAdd(tw)
}
