package clockface

import "github.com/ygelfand/libcountertop/pkg/display/clockkind"

type Kind = clockkind.Kind

const (
	Plain         = clockkind.Plain
	Stack         = clockkind.Stack
	Cards         = clockkind.Cards
	Analog        = clockkind.Analog
	AnalogSeconds = clockkind.AnalogSeconds
	Segments      = clockkind.Segments
	Words         = clockkind.Words
	Overlap       = clockkind.Overlap
	Default       = clockkind.Default
)

var Kinds = clockkind.Kinds
