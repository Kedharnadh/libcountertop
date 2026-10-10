package settings

import (
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	config "github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type chimeChoice struct {
	chime config.Chime
	label string
}

func (c chimeChoice) Label() string { return c.label }

func chimeChoices(builtin []config.Chime, tones []setting.Option) []chimeChoice {
	out := make([]chimeChoice, 0, len(builtin)+len(tones))
	for _, c := range builtin {
		out = append(out, chimeChoice{chime: c, label: c.Label()})
	}
	for _, o := range tones {
		out = append(out, chimeChoice{chime: config.Chime(o.Value), label: o.Label})
	}
	return out
}

func chimeChoiceOf(choices []chimeChoice, c config.Chime) chimeChoice {
	for _, choice := range choices {
		if choice.chime == c {
			return choice
		}
	}
	return chimeChoice{chime: c, label: c.Label()}
}
