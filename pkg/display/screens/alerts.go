package screens

import (
	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/alertview"
)

const priorityAlert = 30

func (a *App) alert(w *gogui.Window) gogui.View {
	m, ok := a.o.Messages.Current()
	if !ok {
		return nil
	}
	return a.alerts.Build(w, m, a.o.Settings().Screen.Alert)
}

func (a *App) alertCovers() bool {
	_, ok := a.o.Messages.Current()
	return ok && alertview.Covers(a.o.Settings().Screen.Alert)
}
