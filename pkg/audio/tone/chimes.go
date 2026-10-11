package tone

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const Level = 0.3

var (
	ToneVolume = []Note{{Freq: 880, Ms: 80}}

	ToneMute     = []Note{{Freq: 880, Ms: 70}, {Freq: 587, Ms: 110}}
	ToneUnmute   = []Note{{Freq: 587, Ms: 70}, {Freq: 880, Ms: 110}}
	ToneMuteHold = []Note{{Freq: 587, Ms: 70}, {Freq: 440, Ms: 130}}

	ToneTrouble = []Note{{Freq: 622, Ms: 90}, {Freq: 466, Ms: 90}, {Freq: 349, Ms: 180}}

	ToneCancel = []Note{{Freq: 698, Ms: 60}, {Freq: 466, Ms: 90}}

	ToneTimer = []Note{
		{Freq: 880, Ms: 160}, {Ms: 130},
		{Freq: 880, Ms: 160}, {Ms: 130},
		{Freq: 880, Ms: 160},
	}
)

var wakeTones = map[schema.Chime][]Note{
	schema.ChimeNone:     nil,
	schema.ChimeMuted:    ToneMute,
	schema.ChimeUnmuted:  ToneUnmute,
	schema.ChimeMuteHold: ToneMuteHold,
	schema.ChimeCancel:   ToneCancel,
	schema.ChimeChirp:    {{Freq: 784, Ms: 60}, {Freq: 1175, Ms: 90}},
	schema.ChimeDing:     {{Freq: 1319, Ms: 200}},
	schema.ChimeRise:     {{Freq: 659, Ms: 55}, {Freq: 880, Ms: 55}, {Freq: 1319, Ms: 110}},
	schema.ChimeTimer:    ToneTimer,
	schema.ChimeTrouble:  ToneTrouble,
}

func Wake(t schema.Chime) []Note { return wakeTones[t] }

func Length(notes []Note) time.Duration {
	var ms int
	for _, n := range notes {
		ms += n.Ms
	}
	return time.Duration(ms) * time.Millisecond
}
