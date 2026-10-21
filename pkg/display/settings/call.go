package settings

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	ha "github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/runtime/safe"
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
	ToggleAnswerUser(string)
	SetRing(config.Chime)
	SetVideoRing(config.Chime)
	SetRingback(config.Chime)
	Tones() []setting.Option
	Preview(config.Chime)
}

const (
	personUser = "user_id"
	peopleWait = 10 * time.Second
)

type Directory interface {
	Entities(ctx context.Context, f ha.Filter) ([]ha.Entity, error)
}

type CallDependencies struct {
	Settings   storage.Section[config.Call]
	Shell      *shell.Shell
	Controller CallController
	Peers      func() []peer.Peer
	Home       Directory
}

type CallPages struct {
	options CallDependencies

	mu     sync.Mutex
	people []ha.Entity
}

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
					if p.options.Home != nil {
						rows = append(rows, menu.Row{
							Label: say.T("call.answer.users"),
							Kind:  menu.Chevron,
							Value: say.F("call.answer.count", map[string]any{"N": len(c.Users())}),
						})
						acts = append(acts, p.open(func() shell.View { return p.AnswerUsers() }))
					}
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

func (p *CallPages) AnswerUsers() *shell.Page {
	p.findPeople()
	return &shell.Page{
		Title: say.T("call.answer.users"),
		Build: func() ([]menu.Row, []func(int)) {
			p.mu.Lock()
			people := p.people
			p.mu.Unlock()
			if len(people) == 0 {
				return []menu.Row{{Label: say.T("call.users.empty")}}, []func(int){nil}
			}
			chosen := p.options.Settings.Read().Users()
			rows := make([]menu.Row, 0, len(people))
			acts := make([]func(int), 0, len(people))
			for _, person := range people {
				user := person.Attributes[personUser]
				rows = append(rows, menu.Row{
					Label: person.Name,
					Kind:  menu.Toggle,
					On:    slices.Contains(chosen, user),
				})
				acts = append(acts, func(int) { p.options.Controller.ToggleAnswerUser(user) })
			}
			return rows, acts
		},
	}
}

func (p *CallPages) findPeople() {
	safe.Go("call people", func() {
		ctx, cancel := context.WithTimeout(context.Background(), peopleWait)
		defer cancel()
		found, err := p.options.Home.Entities(
			ctx,
			ha.Filter{Domains: []string{"person"}, Attributes: []string{personUser}},
		)
		if err != nil {
			slog.Warn("listing Home Assistant people failed", "err", err)
			return
		}
		people := slices.DeleteFunc(
			found,
			func(e ha.Entity) bool { return e.Attributes[personUser] == "" },
		)
		p.mu.Lock()
		p.people = people
		p.mu.Unlock()
		p.options.Shell.Redraw()
	})
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
