package tessera

import (
	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/hook"
	"github.com/ygelfand/libcountertop/pkg/runtime/component"
)

type Set struct {
	Changed hook.Hook[struct{}]
	Told    hook.Hook[component.Event]

	screens []*Screen
}

func NewSet(screens ...*Screen) *Set {
	set := &Set{screens: screens}
	for _, s := range screens {
		s.Changed.Listen(func(struct{}) { set.Changed.Emit(struct{}{}) })
		s.Told.Listen(set.Told.Emit)
	}
	return set
}

func (set *Set) Name() string { return "tessera" }

func (set *Set) Turns() bool { return len(set.screens) > 1 }

func (set *Set) Regrid() {
	for _, s := range set.screens {
		s.Regrid()
	}
}

func (set *Set) Entities() []esphome.Entity {
	var out []esphome.Entity
	for _, s := range set.screens {
		out = append(out, s.Entities()...)
	}
	return out
}

func (set *Set) Actions() []*esphome.Action {
	var out []*esphome.Action
	for _, s := range set.screens {
		out = append(out, s.Actions()...)
	}
	return out
}

func (set *Set) For(w, h int) *Screen {
	want := Portrait
	if w >= h {
		want = Landscape
	}
	var other *Screen
	for _, s := range set.screens {
		if !s.Enabled() {
			continue
		}
		if s.Facing() == want || s.Facing() == Dashboard {
			return s
		}
		other = s
	}
	return other
}
