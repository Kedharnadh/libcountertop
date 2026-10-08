package homecontrol

import (
	"fmt"
	"log/slog"
	"maps"
	"strconv"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Setting = setting.Setting[config.Home]

const Group setting.Group = "Home"

const (
	enabled = "enabled"
	combine = "combine"
)

func (e *Engine) Controls() *setting.Controls[config.Home] { return e.knobs }

func (e *Engine) controls() *setting.Controls[config.Home] {
	return &setting.Controls[config.Home]{
		Table: e.table(),
		Want:  func(setting.Group) bool { return false },
		Read:  e.options.Settings.Read,
		Save:  e.store,
	}
}

func (e *Engine) table() *setting.Table[config.Home] {
	rows := []Setting{
		setting.Flag(
			Setting{Name: enabled, Group: Group, Icon: "mdi:home-automation"},
			func(h *config.Home) *bool { return &h.Enabled },
		),
		setting.Flag(
			Setting{Name: combine, Group: Group, Icon: "mdi:set-merge"},
			func(h *config.Home) *bool { return &h.Combine },
		),
	}
	for _, s := range e.selections {
		rows = append(rows, controlRow(s.Key), groupRow(s.Key))
	}
	return setting.NewTable("home", []setting.Group{Group}, rows,
		setting.Messages{Text: say.T, Missing: say.Missing})
}

func controlRow(key string) Setting {
	name := "control." + key
	return Setting{
		Name:  name,
		Group: Group,
		Kind:  setting.Toggle,
		Read:  func(h *config.Home) string { return setting.OnOff(h.Control[key]) },
		Write: func(h *config.Home, v string) error {
			on, ok := setting.Boolean(v)
			if !ok {
				return fmt.Errorf("%s takes on or off, not %q", name, v)
			}
			control := maps.Clone(h.Control)
			if control == nil {
				control = map[string]bool{}
			}
			if on {
				control[key] = true
			} else {
				delete(control, key)
			}
			h.Control = control
			return nil
		},
	}
}

func groupRow(key string) Setting {
	name := "group." + key
	return Setting{
		Name:  name,
		Group: Group,
		Kind:  setting.Number,
		Max:   config.HomeGroupMost,
		Read:  func(h *config.Home) string { return strconv.Itoa(h.Grouped(key)) },
		Write: func(h *config.Home, v string) error {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 || n > config.HomeGroupMost {
				return fmt.Errorf("%s takes 0 to %d, not %q", name, config.HomeGroupMost, v)
			}
			group := maps.Clone(h.Group)
			if group == nil {
				group = map[string]int{}
			}
			group[key] = n
			h.Group = group
			return nil
		},
	}
}

func (e *Engine) store(s Setting, v string) error {
	err := setting.Store(e.options.Settings, s, v)
	if s.Name == enabled {
		e.options.Client.Enable(e.Enabled())
		e.cache.clear()
	}
	e.options.Shell.Redraw()
	return err
}

func (e *Engine) set(name, v string) {
	if err := e.knobs.Change(name, v); err != nil {
		slog.Error("the home control setting could not be saved", "setting", name, "err", err)
	}
}
