package screens

import (
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"
)

func (a *App) again(w *gogui.Window, after time.Duration) {
	if a.ticking.Swap(true) {
		return
	}
	time.AfterFunc(after, func() {
		a.ticking.Store(false)
		w.QueueCommand(func(w *gogui.Window) { w.InvalidateLayout() })
	})
}
