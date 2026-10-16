package screens

import gogui "github.com/go-gui-org/go-gui/gui"

const priorityAlarm = priorityAlert + 5

func (a *App) alarm(w *gogui.Window) gogui.View {
	if a.o.Alarms == nil {
		return nil
	}
	return a.alarms.Build(w)
}

func (a *App) alarmRinging() bool {
	if a.o.Alarms == nil {
		return false
	}
	al, ok := a.o.Alarms.Current()
	return ok && al.Light <= 0
}
