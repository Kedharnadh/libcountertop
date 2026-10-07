package settings

import (
	"log/slog"

	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/inference/wake"
	"github.com/ygelfand/libcountertop/pkg/say"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type AssistantWord struct {
	ID        string
	Threshold float64
	Tone      config.Chime
	Delivery  config.Delivery
	Look      config.Look
}
type AssistantOptions struct {
	Slots            int
	Read             func(int) AssistantWord
	Models           func() []wake.Model
	Threshold        func(int, float64) error
	Tone             func(int, config.Chime) error
	Delivery         func(int, config.Delivery) error
	Requested, Chime func(int)
	Look             func(int) shell.View
	Push             func(shell.View)
}
type AssistantPages struct{ options AssistantOptions }

func NewAssistantPages(o AssistantOptions) *AssistantPages { return &AssistantPages{options: o} }
func (p *AssistantPages) open(makePage func() shell.View) func(int) {
	return func(int) { p.options.Push(makePage()) }
}

const (
	LeastSure = 50
	MostSure  = 99
)

func (p *AssistantPages) Page() *shell.Page {
	return &shell.Page{
		Title: say.T("assistants.title"),
		Build: func() ([]widget.Row, []func(int)) {
			var (
				rows []widget.Row
				acts []func(int)
			)
			for slot := range p.options.Slots {
				rows = append(rows, widget.Row{
					Label: say.F("assistants.slot", map[string]any{"N": slot + 1}),
					Kind:  widget.Chevron,
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
	if id := p.options.Read(slot).ID; id != "" {
		return wake.Pick(p.options.Models(), id).Phrase
	}
	return say.T("assistants.off")
}

func (p *AssistantPages) Hearing(slot int) string {
	id := p.options.Read(slot).ID
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
		Build: func() ([]widget.Row, []func(int)) {
			saved := p.options.Read(slot)

			return []widget.Row{
					// First, because it is the one row that does something rather than setting
					// something: a way to talk to this assistant without saying its word.
					{Label: say.T("assistants.talk"), Hint: say.T("assistants.talk.hint")},

					{Label: say.T("assistants.threshold"), Kind: widget.Slider, Level: Sensitivity(saved.Threshold),
						Hint: say.T("assistants.threshold.hint")},

					{Label: say.T("assistants.tone"), Kind: widget.Chevron, Value: saved.Tone.Label()},
					{Label: say.T("assistants.reply"), Kind: widget.Chevron, Value: saved.Delivery.Label()},
					{Label: say.T("look.title"), Kind: widget.Chevron, Value: saved.Look.Place.Label()},
				}, []func(int){
					func(int) { p.options.Requested(slot) },
					p.Threshold(slot),
					p.open(func() shell.View { return p.Tone(slot) }),
					p.open(func() shell.View { return p.Delivery(slot) }),
					p.open(func() shell.View { return p.options.Look(slot) }),
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
		if err := p.options.Threshold(slot, v); err != nil {
			slog.Error("saving the wake threshold failed", "slot", slot+1, "err", err)
		}
	}
}

func (p *AssistantPages) Tone(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("tone.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read(slot).Tone

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, c := range config.Chimes() {
				rows = append(rows, widget.Row{Label: c.Label(), Chosen: c == now})
				acts = append(acts, p.UseTone(slot, c))
			}
			return rows, acts
		},
	}
}

func (p *AssistantPages) UseTone(slot int, c config.Chime) func(int) {
	return func(int) {
		if err := p.options.Tone(slot, c); err != nil {
			slog.Error("saving the wake tone failed", "slot", slot+1, "err", err)
			return
		}
		p.options.Chime(slot)
	}
}

func (p *AssistantPages) Delivery(slot int) *shell.Page {
	return &shell.Page{
		Title: say.T("reply.title"),
		Build: func() ([]widget.Row, []func(int)) {
			now := p.options.Read(slot).Delivery

			var (
				rows []widget.Row
				acts []func(int)
			)
			for _, d := range config.Deliveries() {
				rows = append(rows, widget.Row{
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
		if err := p.options.Delivery(slot, d); err != nil {
			slog.Error("saving the reply delivery failed", "slot", slot+1, "err", err)
		}
	}
}
