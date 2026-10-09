package schema

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Tone string

const (
	ToneInfo    Tone = "info"
	ToneSuccess Tone = "success"
	ToneWarning Tone = "warning"
	ToneAlert   Tone = "alert"
)

func Tones() []Tone { return []Tone{ToneInfo, ToneSuccess, ToneWarning, ToneAlert} }

func (t Tone) Label() string {
	switch t {
	case ToneSuccess:
		return say.T("tone.success")
	case ToneWarning:
		return say.T("tone.warning")
	case ToneAlert:
		return say.T("tone.alert")
	}
	return say.T("tone.info")
}

func (t Tone) Color(palette theme.Theme) theme.Color {
	switch t {
	case ToneSuccess:
		return palette.Success
	case ToneWarning:
		return palette.Warning
	case ToneAlert:
		return palette.Danger
	}
	return palette.Accent
}
