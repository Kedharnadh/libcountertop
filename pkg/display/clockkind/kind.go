package clockkind

import "github.com/ygelfand/libcountertop/pkg/say"

type Kind string

func (k Kind) Label() string {
	if key := k.LabelKey(); key != "" {
		return say.T(key)
	}
	return string(k)
}

const (
	None          Kind = "none"
	Plain         Kind = "plain"
	Stack         Kind = "stack"
	Cards         Kind = "cards"
	Analog        Kind = "analog"
	AnalogSeconds Kind = "analog-seconds"
	Segments      Kind = "segments"
	Words         Kind = "words"
	Overlap       Kind = "overlap"
	Default            = Plain
)

func Kinds() []Kind {
	return []Kind{Plain, Stack, Cards, Analog, AnalogSeconds, Segments, Words, Overlap}
}

func (k Kind) LabelKey() string {
	if k == None {
		return "face.none"
	}
	for _, known := range Kinds() {
		if k == known {
			return "face." + string(k)
		}
	}
	return ""
}
