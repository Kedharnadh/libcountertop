package turn

import (
	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Effects interface {
	Begin()
	End()
	Clear()
	Play(string, bool)
	Elsewhere()
}

type Presentation struct {
	Settings storage.Section[schema.Wake]
	Shown    *hook.Hook[Showing]
	Effects  Effects
}

func (p *Presentation) Begin() {
	if p.Effects != nil {
		p.Effects.Begin()
	}
}
func (p *Presentation) End() {
	if p.Effects != nil {
		p.Effects.End()
	}
}
func (p *Presentation) Elsewhere() {
	if p.Effects != nil {
		p.Effects.Elsewhere()
	}
}
func (p *Presentation) Show(s Showing) {
	if p.Shown != nil {
		p.Shown.Emit(s)
	}
}
func (p *Presentation) Phase(slot int, phase Phase) {
	if p.Effects == nil {
		return
	}
	if phase == Idle {
		p.Effects.Clear()
		return
	}
	w := p.Settings.Read().Slot(slot)
	fallback := w.Effect
	if fallback == schema.EffectNone {
		fallback = ""
	}
	if phase == Listening {
		if fallback != "" {
			p.Effects.Play(fallback, false)
		}
		return
	}
	name := w.ThinkingEffect
	if phase == Replying {
		name = w.ReplyingEffect
	}
	switch name {
	case "":
		if fallback != "" {
			p.Effects.Play(fallback, true)
		}
	case schema.EffectNone:
		p.Effects.Clear()
	default:
		p.Effects.Play(name, false)
	}
}
