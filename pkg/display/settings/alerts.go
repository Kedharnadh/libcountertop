package settings

import (
	"log/slog"
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/dawnview"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/previews"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func (p *BasicPages) AlertsPage() *shell.Page {
	return &shell.Page{
		Title: say.T("alerts.page"),
		Build: func() ([]menu.Row, []func(int)) {
			cfg := p.state()
			a := p.options.Alerts.Current()
			rings := config.RingDelays()
			ring := menu.Steps(len(rings))
			choices := chimeChoices(config.AlertChimes(), p.options.Alerts.Tones())
			chime := func(label string, c config.Chime) menu.Row {
				return menu.Row{
					Label: label,
					Kind:  menu.Chevron,
					Value: chimeChoiceOf(choices, c).label,
				}
			}
			rows := []menu.Row{
				{
					Label: say.T("display.alerts"),
					Kind:  menu.Chevron,
					Value: cfg.Screen.Alert.Label(),
				},
				chime(say.T("alerts.setting.info"), a.Info),
				chime(say.T("alerts.setting.success"), a.Success),
				chime(say.T("alerts.setting.warning"), a.Warning),
				chime(say.T("alerts.setting.alert"), a.Alert),
				{
					Label: say.T("display.timer"),
					Kind:  menu.Chevron,
					Value: cfg.Screen.Timer.Label(),
				},
				chime(say.T("alerts.setting.timer"), a.Timer),
				{
					Label: say.T("alerts.setting.ring_for"),
					Kind:  menu.Slider,
					Level: ring.Level(slices.Index(rings, a.RingFor)),
					Snap:  ring.Snap,
					Value: a.RingFor.Label(),
				},
				{
					Label: say.T("alerts.setting.wake"),
					Kind:  menu.Chevron,
					Value: a.Wake.Label(),
				},
			}
			pick := func(name string, read func(config.Alerts) config.Chime) func(int) {
				return p.open(func() shell.View {
					return Choose(
						say.T("alerts.setting."+name),
						choices,
						func() chimeChoice {
							return chimeChoiceOf(choices, read(p.options.Alerts.Current()))
						},
						func(c chimeChoice) { p.setAlertChime(name, c.chime) },
						nil,
					)
				})
			}
			acts := []func(int){
				p.open(func() shell.View { return p.AlertStylePage() }),
				pick("info", func(a config.Alerts) config.Chime { return a.Info }),
				pick("success", func(a config.Alerts) config.Chime { return a.Success }),
				pick("warning", func(a config.Alerts) config.Chime { return a.Warning }),
				pick("alert", func(a config.Alerts) config.Chime { return a.Alert }),
				p.open(func() shell.View { return p.TimerStylePage() }),
				pick("timer", func(a config.Alerts) config.Chime { return a.Timer }),
				func(level int) {
					p.setAlertSetting("ring_for", string(rings[ring.Index(level)]))
				},
				p.open(func() shell.View { return p.WakePage() }),
			}
			return rows, acts
		},
	}
}

func (p *BasicPages) setAlertChime(name string, c config.Chime) {
	p.setAlertSetting(name, string(c))
	p.options.Alerts.Preview(c)
}

func (p *BasicPages) setAlertSetting(name, value string) {
	if err := p.options.Alerts.Change(name, value); err != nil {
		slog.Error("saving an alert setting failed", "name", name, "err", err)
	}
}

func (p *BasicPages) WakePage() *shell.Page {
	return Tiles(
		say.T("alerts.setting.wake"),
		config.WakeLooks(),
		func() config.WakeLook { return p.options.Alerts.Current().Wake },
		func(w config.WakeLook) { p.setAlertSetting("wake", string(w)) },
		func(w config.WakeLook) menu.Preview { return dawnview.Preview(w) },
	)
}

func (p *BasicPages) AlertStylePage() *shell.Page {
	return Tiles(
		say.T("display.alerts"),
		config.AlertStyles(),
		func() config.AlertStyle { return p.state().Screen.Alert },
		p.setAlert,
		func(s config.AlertStyle) menu.Preview { return previews.Alert(s) },
	)
}

func (p *BasicPages) TimerStylePage() *shell.Page {
	return Tiles(
		say.T("display.timer"),
		config.TimerStyles(),
		func() config.TimerStyle { return p.state().Screen.Timer },
		p.setTimer,
		func(s config.TimerStyle) menu.Preview { return previews.Timer(s) },
	)
}
