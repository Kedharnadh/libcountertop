package settings

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type CallController interface {
	SetStream(config.CallStream)
	SetIncoming(bool)
	SetAutoAnswer(bool)
	SetAutoVideo(bool)
	SetPauseWake(bool)
	SetAnswerFrom(config.CallAudience)
	ToggleAnswerPeer(string)
	SetRing(config.Chime)
	SetVideoRing(config.Chime)
	SetRingback(config.Chime)
	Tones() []setting.Option
	Preview(config.Chime)
}
type CallDependencies struct {
	Settings   storage.Section[config.Call]
	Shell      *shell.Shell
	Controller CallController
	Peers      func() []peer.Peer
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
					SegmentRow(say.T("call.answer.from"), config.CallAudiences(), audience(c)),
				)
				acts = append(
					acts,
					SegmentPick(config.CallAudiences(), p.options.Controller.SetAnswerFrom),
				)
				if c.AnswerFrom == config.AnswerSelected {
					rows = append(rows, menu.Row{
						Label: say.T("call.answer.peers"),
						Kind:  menu.Chevron,
						Value: say.F("call.answer.count", map[string]any{"N": len(c.Peers())}),
					})
					acts = append(acts, p.open(func() shell.View { return p.AnswerPeers() }))
				}
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
			choices := chimeChoices(config.CallChimes(), p.options.Controller.Tones())
			tone := func(label string, read func(config.Call) config.Chime, set func(config.Chime)) {
				rows = append(rows, menu.Row{
					Label: label,
					Kind:  menu.Chevron,
					Value: chimeChoiceOf(choices, read(c)).label,
				})
				acts = append(acts, p.open(func() shell.View {
					return Choose(label, choices,
						func() chimeChoice {
							return chimeChoiceOf(choices, read(p.options.Settings.Read()))
						},
						func(c chimeChoice) {
							set(c.chime)
							p.options.Controller.Preview(c.chime)
						},
						nil,
					)
				}))
			}
			tone(say.T("call.ring"), func(c config.Call) config.Chime { return c.Ring },
				p.options.Controller.SetRing)
			tone(say.T("call.videoring"), func(c config.Call) config.Chime { return c.VideoRing },
				p.options.Controller.SetVideoRing)
			tone(say.T("call.ringback"), func(c config.Call) config.Chime { return c.Ringback },
				p.options.Controller.SetRingback)
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
func (p *CallPages) AnswerPeers() *shell.Page {
	return &shell.Page{
		Title: say.T("call.answer.peers"),
		Build: func() ([]menu.Row, []func(int)) {
			var peers []peer.Peer
			if p.options.Peers != nil {
				peers = p.options.Peers()
			}
			if len(peers) == 0 {
				return []menu.Row{{Label: say.T("call.empty")}}, []func(int){nil}
			}
			chosen := p.options.Settings.Read().Peers()
			rows := make([]menu.Row, 0, len(peers))
			acts := make([]func(int), 0, len(peers))
			for _, pr := range peers {
				on := slices.Contains(chosen, pr.ID)
				rows = append(
					rows,
					menu.Row{Label: pr.Name, Hint: pr.Model, Kind: menu.Toggle, On: on},
				)
				acts = append(acts, func(int) { p.options.Controller.ToggleAnswerPeer(pr.ID) })
			}
			return rows, acts
		},
	}
}

func audience(c config.Call) config.CallAudience {
	if c.AnswerFrom == config.AnswerSelected {
		return config.AnswerSelected
	}
	return config.AnswerEveryone
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
