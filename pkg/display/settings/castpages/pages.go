package castpages

import (
	"slices"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/media/cast/protocols/youtube"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Switch interface {
	SetReceiver(on bool)
}

type Lounge interface {
	SetLoungeOnDemand(on bool)
	SetSkip(categories string)
	SetLiveDelay(seconds int)
}

type PrimeOptions interface {
	SetPrimePersist(on bool)
	SetPrimeSkipIntro(on bool)
}

type PrimeRegistration interface {
	PrimeRegistered() bool
	ResetPrime()
}

type Receiver interface {
	Switch
	Lounge
	PrimeOptions
	PrimeRegistration
}

type Dependencies struct {
	Settings storage.Section[schema.Cast]
	Shell    *shell.Shell
	Receiver Receiver
}
type Pages struct{ options Dependencies }

func New(o Dependencies) *Pages { return &Pages{options: o} }
func (p *Pages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(makePage()) }
}

const (
	LiveDelayLeast = 4
	LiveDelayMost  = 60
	delaySpan      = LiveDelayMost - LiveDelayLeast
)

func (p *Pages) Cast() *shell.Page {
	return &shell.Page{
		Title: say.T("features.cast"),
		Build: func() ([]menu.Row, []func(int)) {
			on := p.options.Settings.Read().Receiver
			return []menu.Row{
					{Label: say.T("cast.enable"), Kind: menu.Toggle, On: on},
					{Label: say.T("cast.apps"), Kind: menu.Chevron},
				}, []func(int){
					func(int) { p.options.Receiver.SetReceiver(!on) },
					p.open(func() shell.View { return p.CastApps() }),
				}
		},
	}
}

func (p *Pages) CastApps() *shell.Page {
	return &shell.Page{
		Title: say.T("cast.apps"),
		Build: func() ([]menu.Row, []func(int)) {
			return []menu.Row{
					{Label: say.T("youtube.title"), Kind: menu.Chevron},
					{Label: say.T("cast.prime"), Kind: menu.Chevron},
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
		Build: func() ([]menu.Row, []func(int)) {
			chosen := p.options.Settings.Read().YouTube.Skip
			rows := make([]menu.Row, 0, len(youtube.Categories))
			acts := make([]func(int), 0, len(youtube.Categories))
			for _, c := range youtube.Categories {
				on := slices.Contains(chosen, c)
				rows = append(
					rows,
					menu.Row{Label: say.T("youtube.category." + c), Kind: menu.Toggle, On: on},
				)
				acts = append(acts, func(int) {
					next := slices.DeleteFunc(
						slices.Clone(chosen),
						func(x string) bool { return x == c },
					)
					if !on {
						next = append(next, c)
					}
					p.options.Receiver.SetSkip(strings.Join(next, ","))
				})
			}
			return rows, acts
		},
	}
}

func (p *Pages) Prime() *shell.Page {
	return &shell.Page{
		Title: say.T("cast.prime"),
		Build: func() ([]menu.Row, []func(int)) {
			persist, skip := p.options.Settings.Read().Prime.Persist, p.options.Settings.Read().Prime.SkipIntro
			state := say.T("cast.prime.unregistered")
			if p.options.Receiver.PrimeRegistered() {
				state = say.T("cast.prime.registered")
			}
			return []menu.Row{
					{
						Label: say.T("cast.prime.persist"),
						Hint:  say.T("cast.prime.persist.hint"),
						Kind:  menu.Toggle,
						On:    persist,
					},
					{Label: say.T("cast.prime.reset"), Kind: menu.Plain, Value: state},
					{
						Label: say.T("cast.prime.skipintro"),
						Hint:  say.T("cast.prime.skipintro.hint"),
						Kind:  menu.Toggle,
						On:    skip,
					},
				}, []func(int){
					func(int) { p.options.Receiver.SetPrimePersist(!persist) },
					func(int) { p.options.Receiver.ResetPrime() },
					func(int) { p.options.Receiver.SetPrimeSkipIntro(!skip) },
				}
		},
	}
}

func (p *Pages) YouTube() *shell.Page {
	return &shell.Page{
		Title: say.T("youtube.title"),
		Build: func() ([]menu.Row, []func(int)) {
			c := p.options.Settings.Read().YouTube
			return []menu.Row{
					{
						Label: say.T("youtube.demand"),
						Hint:  say.T("youtube.demand.hint"),
						Kind:  menu.Toggle,
						On:    c.OnDemand,
					},
					{Label: say.T("youtube.skip"), Kind: menu.Chevron, Value: SkipSays(c.Skip)},
					{
						Label: say.T("youtube.delay"),
						Hint:  say.T("youtube.delay.hint"),
						Kind:  menu.Slider,
						Level: DelayLevel(
							c.LiveDelay,
						),
						Snap:  DelaySnap,
						Value: DelaySays(c.LiveDelay),
					},
				}, []func(int){
					func(int) { p.options.Receiver.SetLoungeOnDemand(!c.OnDemand) },
					p.open(func() shell.View { return p.Sponsor() }),
					func(level int) { p.options.Receiver.SetLiveDelay(DelayOf(level)) },
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
