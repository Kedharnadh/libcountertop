package homecontrol

import (
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type Selection struct {
	Key    string
	Filter homeassistant.Filter
	engine *Engine
}

func (s Selection) Name() string { return say.T("home.selection." + s.Key) }

func (s Selection) Pick() config.HomePick {
	return s.engine.options.Settings.Read().Picked(s.Key).Clone()
}

func Includes(p config.HomePick, e homeassistant.Entity) bool {
	return p.All ||
		slices.Contains(p.Entities, e.ID) ||
		(e.AreaID != "" && slices.Contains(p.Areas, e.AreaID)) ||
		slices.ContainsFunc(e.Labels, func(l string) bool { return slices.Contains(p.Labels, l) })
}

func Summary(p config.HomePick) string {
	switch {
	case p.All:
		return say.T("home.all")
	case p.Empty():
		return say.T("home.none")
	}
	var parts []string
	for _, part := range []struct {
		kind string
		n    int
	}{
		{"labels", len(p.Labels)},
		{"areas", len(p.Areas)},
		{"entities", len(p.Entities)},
	} {
		if part.n > 0 {
			parts = append(parts, say.F("home.picked."+part.kind, map[string]any{"N": part.n}))
		}
	}
	return strings.Join(parts, ", ")
}

func (s Selection) Controlled() bool {
	return s.engine.Enabled() && s.engine.options.Settings.Read().Control[s.Key]
}

func (e *Engine) Enabled() bool { return e.options.Settings.Read().Enabled }

func (e *Engine) SetEnabled(on bool) {
	if err := e.options.Settings.Update(func(c *config.Home) { c.Enabled = on }); err != nil {
		slog.Error("the home control setting could not be saved", "err", err)
	}
	e.options.Client.Enable(e.Enabled())
	e.cache.clear()
}

func (s Selection) GroupFrom() int { return s.engine.options.Settings.Read().Grouped(s.Key) }

func (s Selection) SetGroupFrom(n int) {
	if err := s.engine.options.Settings.Update(func(c *config.Home) {
		if c.Group == nil {
			c.Group = map[string]int{}
		}
		c.Group[s.Key] = min(max(n, 0), config.HomeGroupMost)
	}); err != nil {
		slog.Error("the grouping setting could not be saved", "selection", s.Key, "err", err)
	}
	s.engine.options.Shell.Redraw()
}

func (s Selection) SetControlled(on bool) {
	if err := s.engine.options.Settings.Update(func(c *config.Home) {
		if c.Control == nil {
			c.Control = map[string]bool{}
		}
		if on {
			c.Control[s.Key] = true
		} else {
			delete(c.Control, s.Key)
		}
	}); err != nil {
		slog.Error("the control setting could not be saved", "selection", s.Key, "err", err)
	}
	s.engine.options.Shell.Redraw()
}

func (e *Engine) Combined() bool {
	return e.options.Settings.Read().Combine && e.selections[0].Controlled()
}

func (e *Engine) SetCombined(on bool) {
	if err := e.options.Settings.Update(func(c *config.Home) { c.Combine = on }); err != nil {
		slog.Error("the combine setting could not be saved", "err", err)
	}
	e.options.Shell.Redraw()
}

func (e *Engine) SelectionPage(s Selection) *shell.Page {
	return &shell.Page{
		Title: s.Name(),
		Build: func() ([]widget.Row, []func(int)) {
			on, from := s.Controlled(), s.GroupFrom()
			rows := []widget.Row{
				{Label: say.T("home.enable"), Kind: widget.Toggle, On: on},
				{Label: say.T("home.devices"), Kind: widget.Chevron, Value: Summary(s.Pick())},
				{Label: say.T("home.group"), Hint: groupHint(from), Kind: widget.Slider, Level: groupLevel(from), Snap: groupSnap, Value: strconv.Itoa(from)},
			}
			taps := []func(int){
				func(int) { s.SetControlled(!on) },
				func(int) { e.options.Shell.Push(e.NewPicker(s)) },
				func(level int) { s.SetGroupFrom(groupOf(level)) },
			}
			if s.Key == e.selections[0].Key {
				combine := e.options.Settings.Read().Combine
				rows = append(rows, widget.Row{Label: say.T("home.combine"), Kind: widget.Toggle, On: combine})
				taps = append(taps, func(int) { e.SetCombined(!combine) })
			}
			return rows, taps
		},
	}
}

func groupOf(level int) int {
	return min(max((level*config.HomeGroupMost+50)/100, 0), config.HomeGroupMost)
}

func groupLevel(n int) int { return n * 100 / config.HomeGroupMost }

func groupSnap(level int) int { return groupLevel(groupOf(level)) }

func groupHint(n int) string {
	switch n {
	case 0:
		return say.T("home.group.none")
	case 1:
		return say.T("home.group.all")
	}
	return say.F("home.group.from", map[string]any{"N": n})
}

type Home struct{}

func (e *Engine) Page() *Home        { return e.home }
func (*Home) Covers() bool           { return true }
func (*Home) Timeout() time.Duration { return shell.SettingsTimeout }
