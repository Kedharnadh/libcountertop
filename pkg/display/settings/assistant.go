package settings

import (
	"log/slog"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/assistant/turn"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/inference/wake"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type AssistantDependencies struct {
	Slots       int
	Preferences Preferences
	Models      func() []wake.Model
	Requested   *hook.Hook[int]
	Words       turn.Words
	Look        *LookPages
	Shell       *shell.Shell
}
type AssistantPages struct{ options AssistantDependencies }

func NewAssistantPages(
	o AssistantDependencies,
) *AssistantPages {
	return &AssistantPages{options: o}
}
func (p *AssistantPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Shell.Push(makePage()) }
}

// Below 0.5 a model fires on anything; at 1 nothing reaches it.
const (
	LeastSure = 50
	MostSure  = 99
)

func (p *AssistantPages) Page() *shell.Page {
	return &shell.Page{
		Title: say.T("assistants.title"),
		Build: func() ([]menu.Row, []func(int)) {
			var (
				rows []menu.Row
				acts []func(int)
			)
			for slot := range p.options.Slots {
				rows = append(rows, menu.Row{
					Label: say.F("assistants.slot", map[string]any{"N": slot + 1}),
					Kind:  menu.Chevron,
					Value: p.Listening(slot),

					Hint:   p.Hearing(slot),
					Chosen: false,
				})
				acts = append(acts, p.open(func() shell.View { return p.Slot(slot) }))
			}
			return rows, acts
		},
	}
}

func (p *AssistantPages) Listening(slot int) string {
	if id := p.options.Preferences.Wake.Read().Slot(slot).ID; id != "" {
		return wake.Pick(p.options.Models(), id).Phrase
	}
	return say.T("assistants.off")
}

func (p *AssistantPages) Hearing(slot int) string {
	id := p.options.Preferences.Wake.Read().Slot(slot).ID
	if id == "" {
		return say.T("assistants.unset")
	}
	if _, ok := wake.Find(p.options.Models(), id); !ok {
		return say.T("assistants.waiting")
	}
	return ""
}

func (p *AssistantPages) Slot(slot int) *shell.Page {
	return &shell.Page{
		Title: say.F("assistants.slot", map[string]any{"N": slot + 1}),
		Build: func() ([]menu.Row, []func(int)) {
			saved := p.options.Preferences.Wake.Read().Slot(slot)

			return []menu.Row{
					{Label: say.T("assistants.talk"), Hint: say.T("assistants.talk.hint")},

					{
						Label: say.T("assistants.threshold"),
						Kind:  menu.Slider,
						Level: Sensitivity(saved.Threshold),
						Hint:  say.T("assistants.threshold.hint"),
					},

					{
						Label: say.T("assistants.tone"),
						Kind:  menu.Chevron,
						Value: saved.Tone.Label(),
					},
					{
						Label: say.T("assistants.reply"),
						Kind:  menu.Chevron,
						Value: saved.Delivery.Label(),
					},
					{
						Label: say.T("look.title"),
						Kind:  menu.Chevron,
						Value: saved.Look.Place.Label(),
					},
				}, []func(int){
					func(int) { p.options.Requested.Emit(slot) },
					p.Threshold(slot),
					p.open(func() shell.View { return p.Tone(slot) }),
					p.open(func() shell.View { return p.Delivery(slot) }),
					p.open(func() shell.View { return p.options.Look.Page(slot) }),
				}
		},
	}
}

func Sensitivity(v float64) int {
	return LeastSure + MostSure - min(max(int(v*100+0.5), LeastSure), MostSure)
}

func (p *AssistantPages) Threshold(slot int) func(int) {
	return func(level int) {
		v := float64(LeastSure+MostSure-min(max(level, LeastSure), MostSure)) / 100
		if err := p.options.Preferences.Word(
			slot,
			func(w *config.WakeWord) { w.Threshold = v },
		); err != nil {
			slog.Error("saving the wake threshold failed", "slot", slot+1, "err", err)
		}
	}
}

func (p *AssistantPages) Phrases() string {
	var names []string
	for slot := range p.options.Slots {
		if p.options.Preferences.Wake.Read().Slot(slot).ID != "" {
			names = append(names, p.Listening(slot))
		}
	}
	if len(names) == 0 {
		return say.T("assistants.off")
	}
	return strings.Join(names, ", ")
}

func (p *AssistantPages) Tone(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("tone.title"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Preferences.Wake.Read().Slot(slot).Tone

			var (
				rows []menu.Row
				acts []func(int)
			)
			for _, c := range config.Chimes() {
				rows = append(rows, menu.Row{Label: c.Label(), Chosen: c == now})
				acts = append(acts, p.UseTone(slot, c))
			}
			return rows, acts
		},
	}
}

func (p *AssistantPages) UseTone(slot int, c config.Chime) func(int) {
	return func(int) {
		if err := p.options.Preferences.Word(
			slot,
			func(w *config.WakeWord) { w.Tone = c },
		); err != nil {
			slog.Error("saving the wake tone failed", "slot", slot+1, "err", err)
			return
		}
		p.options.Words.Chime(slot)
	}
}

func (p *AssistantPages) Delivery(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("reply.title"),
		Build: func() ([]menu.Row, []func(int)) {
			now := p.options.Preferences.Wake.Read().Slot(slot).Delivery

			var (
				rows []menu.Row
				acts []func(int)
			)
			for _, d := range config.Deliveries() {
				rows = append(rows, menu.Row{
					Label: d.Label(), Chosen: d == now, Hint: Delivering(d),
				})
				acts = append(acts, p.UseDelivery(slot, d))
			}
			return rows, acts
		},
	}
}

func Delivering(d config.Delivery) string {
	if d == config.DeliveryStream {
		return say.T("reply.stream")
	}
	return say.T("reply.download")
}

func (p *AssistantPages) UseDelivery(slot int, d config.Delivery) func(int) {
	return func(int) {
		if err := p.options.Preferences.Word(
			slot,
			func(w *config.WakeWord) { w.Delivery = d },
		); err != nil {
			slog.Error("saving the reply delivery failed", "slot", slot+1, "err", err)
		}
	}
}
