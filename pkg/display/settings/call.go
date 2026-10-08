package settings

import (
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type CallController interface {
	SetStream(config.CallStream)
	SetIncoming(bool)
	SetAutoAnswer(bool)
	SetAutoVideo(bool)
	SetPauseWake(bool)
}
type CallDependencies struct {
	Settings   storage.Section[config.Call]
	Shell      *shell.Shell
	Controller CallController
}
type CallPages struct{ options CallDependencies }

func NewCallPages(o CallDependencies) *CallPages { return &CallPages{options: o} }
func (p *CallPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(makePage()) }
}
func (p *CallPages) Calls() *shell.Page {
	return &shell.Page{
		Title: say.T("call.settings"),
		Build: func() ([]menu.Row, []func(int)) {
			c := p.options.Settings.Read()
			rows := []menu.Row{
				{Label: say.T("call.incoming.allow"), Kind: menu.Toggle, On: c.Incoming},
				{
					Label: say.T("call.auto"),
					Hint:  say.T("call.auto.hint"),
					Kind:  menu.Toggle,
					On:    c.AutoAnswer,
				},
			}
			acts := []func(int){
				func(int) { p.options.Controller.SetIncoming(!c.Incoming) },
				func(int) { p.options.Controller.SetAutoAnswer(!c.AutoAnswer) },
			}
			if c.AutoAnswer {
				rows = append(
					rows,
					menu.Row{
						Label: say.T("call.autovideo"),
						Kind:  menu.Toggle,
						On:    c.AutoVideo,
					},
				)
				acts = append(acts, func(int) { p.options.Controller.SetAutoVideo(!c.AutoVideo) })
			}
			rows = append(
				rows,
				menu.Row{Label: say.T("call.pausewake"), Kind: menu.Toggle, On: c.PauseWake},
			)
			acts = append(acts, func(int) { p.options.Controller.SetPauseWake(!c.PauseWake) })
			rows = append(
				rows,
				menu.Row{
					Label: say.T("call.stream"),
					Hint:  say.T("call.stream.hint"),
					Kind:  menu.Chevron,
					Value: c.Stream.Label(),
				},
			)
			acts = append(acts, p.open(func() shell.View { return p.CallStream() }))
			return rows, acts
		},
	}
}
func (p *CallPages) CallStream() *shell.Page {
	return &shell.Page{
		Title: say.T("call.stream"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Settings.Read().Stream
			var rows []menu.Row
			var acts []func(int)
			for _, v := range config.CallStreams() {
				rows = append(
					rows,
					menu.Row{Label: v.Label(), Kind: menu.Plain, Chosen: v == now},
				)
				acts = append(acts, func(int) { p.options.Controller.SetStream(v) })
			}
			return rows, acts
		},
	}
}
