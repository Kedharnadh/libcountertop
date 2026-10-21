package homecontrol

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/apps"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Full struct{ Sel Selection }

func (*Full) Covers() bool           { return true }
func (*Full) Timeout() time.Duration { return shell.SettingsTimeout }

var appOf = map[string]struct{ id, icon string }{
	"lights":   {config.AppLights, "mdi:lightbulb-group"},
	"switches": {config.AppSwitches, "mdi:toggle-switch"},
}

func (e *Engine) Apps() []apps.App {
	var out []apps.App
	for _, s := range e.selections {
		a, ok := appOf[s.Key]
		if !ok {
			continue
		}
		full := &Full{Sel: s}
		out = append(out, apps.App{
			ID:   a.id,
			Name: s.Name,
			Icon: a.icon,
			Open: func() { e.options.Shell.Push(full) },
		})
	}
	return out
}
