package clockview

import "github.com/ygelfand/libcountertop/pkg/display/clockkind"

// Of resolves a clock identifier, reporting false for unsupported choices or no clock.
func Of(kind clockkind.Kind) (View, bool) {
	switch kind {
	case clockkind.Plain:
		return Plain, true
	case clockkind.Stack:
		return Stack, true
	case clockkind.Cards:
		return Cards, true
	case clockkind.Analog:
		return Analog(false), true
	case clockkind.AnalogSeconds:
		return Analog(true), true
	case clockkind.Words:
		return Words, true
	case clockkind.Overlap:
		return Overlap, true
	case clockkind.Segments:
		return Segments, true
	}
	return nil, false
}
