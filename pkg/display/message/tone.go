package message

import (
	"github.com/ygelfand/libcountertop/pkg/display/theme"
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Message struct {
	Title  string
	Body   string
	Tone   Tone
	Icon   string
	Swatch string
	Keys   []Key
	Ended  func(End)
}

type Key struct {
	Text   string
	Swatch string
}

type End struct {
	Key int
	Why string
}

const (
	EndKey      = "key"
	EndTimeout  = "timeout"
	EndReplaced = "replaced"
	EndCleared  = "cleared"
	EndTapped   = "tapped"
)

func Swatch(name string, palette theme.Theme) (theme.Color, bool) {
	switch name {
	case "red":
		return palette.Danger, true
	case "orange":
		return palette.Warning, true
	case "yellow":
		return palette.Roles.Warm, true
	case "green":
		return palette.Success, true
	case "mint":
		return palette.Roles.Air, true
	case "blue":
		return palette.Roles.Cool, true
	case "purple":
		return palette.Roles.Choice, true
	case "pink":
		return palette.Roles.Heat, true
	case "gray", "grey":
		return palette.Muted, true
	}
	return theme.Color{}, false
}

func (m Message) Tint(palette theme.Theme) theme.Color {
	if c, ok := Swatch(m.Swatch, palette); ok {
		return c
	}
	return m.Tone.Color(palette)
}

type Tone string

const (
	ToneInfo    Tone = "info"
	ToneSuccess Tone = "success"
	ToneWarning Tone = "warning"
	ToneAlert   Tone = "alert"
)

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

func Tones() []Tone { return []Tone{ToneInfo, ToneSuccess, ToneWarning, ToneAlert} }

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
