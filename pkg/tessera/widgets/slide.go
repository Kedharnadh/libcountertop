package widgets

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/tessera"
)

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

func (v *View) drag(w *gogui.Window, l tessera.Layout, dx, width float32) {
	v.mu.Lock()
	page := v.page
	excluded := l.Pages[page].Excluded
	v.mu.Unlock()
	if excluded {
		return
	}
	v.pager.Drag(
		w,
		dx,
		width,
		func(by int) int { return neighbour(l, page, by) },
		func(to int) int {
			v.mu.Lock()
			defer v.mu.Unlock()
			old := v.page
			v.page = to
			v.shown = l.Pages[to].ID
			return old
		},
	)
}

func (v *View) move(w *gogui.Window, change func()) {
	v.mu.Lock()
	old, width := v.page, v.width
	v.mu.Unlock()
	change()
	v.mu.Lock()
	to := v.page
	v.mu.Unlock()
	v.pager.Jump(w, old, to, width)
}
