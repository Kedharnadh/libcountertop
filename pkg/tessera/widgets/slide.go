package widgets

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	slideTime  = 280 * time.Millisecond
	jumpShare  = 0.35
	commitPart = 0.25
	edgeGive   = 0.3
)

type slide struct {
	offset float32
	other  int
}

func (v *View) sliding() slide {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.slide
}

func neighbour(l tessera.Layout, page, by int) int {
	shown := pages(l)
	for i, p := range shown {
		if p == page {
			if j := i + by; j >= 0 && j < len(shown) {
				return shown[j]
			}
			return -1
		}
	}
	return -1
}

func (v *View) drag(l tessera.Layout, dx, width float32) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if l.Pages[v.page].Excluded {
		return
	}
	v.slide.offset += dx
	by := 1
	if v.slide.offset > 0 {
		by = -1
	}
	v.slide.other = neighbour(l, v.page, by)
	if v.slide.other < 0 {
		v.slide.offset -= dx * (1 - edgeGive)
	}
	v.slide.offset = max(min(v.slide.offset, width), -width)
}

func (v *View) release(w *gogui.Window, l tessera.Layout, width, velocity float32) {
	v.mu.Lock()
	s := v.slide
	commit := s.other >= 0 && (abs(s.offset) > width*commitPart || abs(velocity) > swipeSpeed) &&
		(velocity == 0 || (velocity < 0) == (s.offset < 0))
	if commit {
		old := v.page
		v.page, v.slide.other = s.other, old
		v.shown = l.Pages[v.page].ID
		if s.offset < 0 {
			v.slide.offset = s.offset + width
		} else {
			v.slide.offset = s.offset - width
		}
	}
	from := v.slide.offset
	v.mu.Unlock()
	v.settle(w, from)
}

func (v *View) jump(w *gogui.Window, old, to int, width float32) {
	if old == to || w == nil {
		return
	}
	v.mu.Lock()
	v.slide.other = old
	v.slide.offset = width * jumpShare
	if to < old {
		v.slide.offset = -v.slide.offset
	}
	from := v.slide.offset
	v.mu.Unlock()
	v.settle(w, from)
}

func (v *View) settle(w *gogui.Window, from float32) {
	tw := gogui.NewTweenAnimation("tessera-slide", from, 0, func(at float32, w *gogui.Window) {
		v.mu.Lock()
		v.slide.offset = at
		v.mu.Unlock()
	})
	tw.Duration = slideTime
	tw.OnDone = func(w *gogui.Window) {
		v.mu.Lock()
		v.slide = slide{other: -1}
		v.mu.Unlock()
		w.InvalidateLayout()
	}
	w.AnimationAdd(tw)
}

func (v *View) move(w *gogui.Window, change func()) {
	v.mu.Lock()
	old, width := v.page, v.width
	v.mu.Unlock()
	change()
	v.mu.Lock()
	to := v.page
	v.mu.Unlock()
	v.jump(w, old, to, width)
}
