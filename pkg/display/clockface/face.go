package clockface

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/display/ui"
)

type Face interface {
	Draw(s ui.Surface, in ui.Rect, r Reading, palette theme.Theme)
}

type Ticking interface {
	Ticks() bool
}

func Ticks(name Kind) bool {
	t, ok := Of(name).(Ticking)
	return ok && t.Ticks()
}

var registered = map[Kind]Face{}

func register(name Kind, f Face) { registered[name] = f }

func Of(name Kind) Face {
	if f, ok := registered[name]; ok {
		return f
	}
	return registered[Default]
}
