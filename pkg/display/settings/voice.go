package settings

import (
	"log/slog"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/display/widgets"
	ha "github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/voice"
)

const HACSLink = "https://my.home-assistant.io/redirect/hacs_repository/" +
	"?owner=ygelfand&repository=echolocal-hacs&category=integration"

type Integrated interface{ Integration() ha.Integration }

func VoicePage(c *setting.Controls[schema.Voice], hacs Integrated, sh *shell.Shell) *shell.Page {
	return &shell.Page{Title: say.T("voice.title"), Build: func() ([]menu.Row, []func(int)) {
		at := c.Read()
		var (
			rows []menu.Row
			taps []func(int)
		)
		for _, g := range []setting.Group{voice.ActionsGroup, voice.CommandsGroup} {
			members := c.Table.In(g)
			if len(members) == 0 {
				continue
			}
			rows = append(
				rows,
				menu.Row{Label: say.T("voice.group." + groupKey(g)), Kind: menu.Plain, Dim: true},
			)
			taps = append(taps, nil)
			if g == voice.ActionsGroup {
				if row, tap, ok := integrationRow(hacs, sh); ok {
					rows, taps = append(rows, row), append(taps, tap)
				}
			}
			master := members[0].On(&at)
			for i, s := range members {
				on := s.On(&at)
				rows = append(rows, menu.Row{
					Label: s.Title(),
					Kind:  menu.Toggle,
					On:    on,
					Dim:   i > 0 && !master,
				})
				taps = append(taps, func(int) {
					if err := c.Change(s.Name, setting.OnOff(!on)); err != nil {
						slog.Error(
							"saving a setting failed",
							"setting",
							"voice."+s.Name,
							"err",
							err,
						)
					}
				})
			}
		}
		return rows, taps
	}}
}

func groupKey(g setting.Group) string {
	if g == voice.ActionsGroup {
		return "actions"
	}
	return "commands"
}

func integrationRow(hacs Integrated, sh *shell.Shell) (menu.Row, func(int), bool) {
	var value string
	switch hacs.Integration() {
	case ha.IntegrationReady:
		return menu.Row{}, nil, false
	case ha.IntegrationMissing:
		value = say.T("voice.hacs.missing")
	case ha.IntegrationOutdated:
		value = say.T("voice.hacs.outdated")
	default:
		value = say.T("voice.hacs.unknown")
	}
	return menu.Row{
			Glyph: gogui.IconWarning,
			Label: say.T("voice.hacs"),
			Kind:  menu.Chevron,
			Value: value,
		},
		func(int) { sh.Push(HACSPage()) },
		true
}

func HACSPage() *shell.Page {
	return &shell.Page{
		Title: say.T("voice.hacs"),
		Build: func() ([]menu.Row, []func(int)) {
			return []menu.Row{{Label: say.T("voice.hacs.hint"), Kind: menu.Plain}}, []func(int){nil}
		},
		Side: func(w *gogui.Window) gogui.View {
			vw, vh := w.WindowSize()
			side := min(float32(vw), float32(vh)) * 0.45
			return widgets.QR(HACSLink, side, side, gogui.RGB(0, 0, 0))
		},
	}
}
