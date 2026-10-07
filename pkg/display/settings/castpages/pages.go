// Package castpages supplies settings pages for devices with Cast playback.
package castpages

import (
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/media/cast/protocols/youtube"
	"github.com/ygelfand/libcountertop/pkg/say"
	"slices"
	"strings"
)

type State struct {
	Receiver, OnDemand, PrimePersist, PrimeSkipIntro bool
	Skip                                             []string
	LiveDelay                                        int
}
type Options struct {
	Read                                                               func() State
	Push                                                               func(shell.View)
	SetReceiver, SetPrimePersist, SetPrimeSkipIntro, SetLoungeOnDemand func(bool)
	SetSkip                                                            func(string)
	SetLiveDelay                                                       func(int)
	PrimeRegistered                                                    func() bool
	ResetPrime                                                         func()
}
type Pages struct{ options Options }

func New(o Options) *Pages { return &Pages{options: o} }
func (p *Pages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Push(makePage()) }
}

const (
	LiveDelayLeast = 4
	LiveDelayMost  = 60
	delaySpan      = LiveDelayMost - LiveDelayLeast
)

func (p *Pages) Cast() *shell.Page {
	return &shell.Page{
		Title: say.T("features.cast"),
		Build: func() ([]widget.Row, []func(int)) {
			on := p.options.Read().Receiver
			return []widget.Row{
					{Label: say.T("cast.enable"), Kind: widget.Toggle, On: on},
					{Label: say.T("cast.apps"), Kind: widget.Chevron},
				}, []func(int){
					func(int) { p.options.SetReceiver(!on) },
					p.open(func() shell.View { return p.CastApps() }),
				}
		},
	}
}

func (p *Pages) CastApps() *shell.Page {
	return &shell.Page{
		Title: say.T("cast.apps"),
		Build: func() ([]widget.Row, []func(int)) {
			return []widget.Row{
					{Label: say.T("youtube.title"), Kind: widget.Chevron},
					{Label: say.T("cast.prime"), Kind: widget.Chevron},
				}, []func(int){
					p.open(func() shell.View { return p.YouTube() }),
					p.open(func() shell.View { return p.Prime() }),
				}
		},
	}
}

func (p *Pages) Sponsor() *shell.Page {
	return &shell.Page{
		Title: say.T("youtube.skip"),
		Build: func() ([]widget.Row, []func(int)) {
			chosen := p.options.Read().Skip
			rows := make([]widget.Row, 0, len(youtube.Categories))
			acts := make([]func(int), 0, len(youtube.Categories))
			for _, c := range youtube.Categories {
				on := slices.Contains(chosen, c)
				rows = append(rows, widget.Row{Label: say.T("youtube.category." + c), Kind: widget.Toggle, On: on})
				acts = append(acts, func(int) {
					next := slices.DeleteFunc(slices.Clone(chosen), func(x string) bool { return x == c })
					if !on {
						next = append(next, c)
					}
					p.options.SetSkip(strings.Join(next, ","))
				})
			}
			return rows, acts
		},
	}
}

func (p *Pages) Prime() *shell.Page {
	return &shell.Page{
		Title: say.T("cast.prime"),
		Build: func() ([]widget.Row, []func(int)) {
			persist, skip := p.options.Read().PrimePersist, p.options.Read().PrimeSkipIntro
			state := say.T("cast.prime.unregistered")
			if p.options.PrimeRegistered() {
				state = say.T("cast.prime.registered")
			}
			return []widget.Row{
					{Label: say.T("cast.prime.persist"), Hint: say.T("cast.prime.persist.hint"), Kind: widget.Toggle, On: persist},
					{Label: say.T("cast.prime.reset"), Kind: widget.Plain, Value: state},
					{Label: say.T("cast.prime.skipintro"), Hint: say.T("cast.prime.skipintro.hint"), Kind: widget.Toggle, On: skip},
				}, []func(int){
					func(int) { p.options.SetPrimePersist(!persist) },
					func(int) { p.options.ResetPrime() },
					func(int) { p.options.SetPrimeSkipIntro(!skip) },
				}
		},
	}
}

func (p *Pages) YouTube() *shell.Page {
	return &shell.Page{
		Title: say.T("youtube.title"),
		Build: func() ([]widget.Row, []func(int)) {
			c := p.options.Read()
			return []widget.Row{
					{
						Label: say.T("youtube.demand"),
						Hint:  say.T("youtube.demand.hint"),
						Kind:  widget.Toggle,
						On:    c.OnDemand,
					},
					{Label: say.T("youtube.skip"), Kind: widget.Chevron, Value: SkipSays(c.Skip)},
					{
						Label: say.T("youtube.delay"),
						Hint:  say.T("youtube.delay.hint"),
						Kind:  widget.Slider,
						Level: DelayLevel(
							c.LiveDelay,
						),
						Snap:  DelaySnap,
						Value: DelaySays(c.LiveDelay),
					},
				}, []func(int){
					func(int) { p.options.SetLoungeOnDemand(!c.OnDemand) },
					p.open(func() shell.View { return p.Sponsor() }),
					func(level int) { p.options.SetLiveDelay(DelayOf(level)) },
				}
		},
	}
}
func Toggled(skip []string, cat string, on bool) []string {
	out := slices.DeleteFunc(slices.Clone(skip), func(s string) bool { return s == cat })
	if on {
		out = append(out, cat)
	}
	return out
}
func SkipSays(skip []string) string {
	if len(skip) == 0 {
		return say.T("youtube.skip.none")
	}
	return say.F("youtube.skip.some", map[string]any{"N": len(skip)})
}
func DelayOf(level int) int {
	return LiveDelayLeast + (min(max(level, 0), 100)*delaySpan+50)/100
}
func DelayLevel(seconds int) int {
	return (min(max(seconds-LiveDelayLeast, 0), delaySpan)*100 + delaySpan/2) / delaySpan
}
func DelaySnap(level int) int { return DelayLevel(DelayOf(level)) }
func DelaySays(
	seconds int,
) string {
	return say.F("youtube.delay.value", map[string]any{"N": seconds})
}
