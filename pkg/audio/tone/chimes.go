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

	// ToneTrouble falls twice and ends low, which no acknowledgement does. A request that cannot be
	// served has to sound different from one that was, or a failure is indistinguishable from the
	// device having ignored the person entirely.
	ToneTrouble = []Note{{Freq: 622, Ms: 90}, {Freq: 466, Ms: 90}, {Freq: 349, Ms: 180}}

	// ToneCancel is one short falling pair: the request was dropped, which is neither a failure nor
	// an answer.
	ToneCancel = []Note{{Freq: 698, Ms: 60}, {Freq: 466, Ms: 90}}

	// ToneTimer is a timer that has finished. Three of the same note, because it repeats until
	// somebody stops it and a melody wears out faster than a beep does.
	ToneTimer = []Note{
		{Freq: 880, Ms: 160}, {Ms: 130},
		{Freq: 880, Ms: 160}, {Ms: 130},
		{Freq: 880, Ms: 160},
	}
)

// wakeTones is what a detection can sound like. They are told apart by shape rather than pitch, so
// two wake words set to different ones are distinguishable without knowing which is which.
var wakeTones = map[schema.Chime][]Note{
	schema.ChimeNone:  nil,
	schema.ChimeChirp: {{Freq: 784, Ms: 60}, {Freq: 1175, Ms: 90}},
	schema.ChimeDing:  {{Freq: 1319, Ms: 200}},
	schema.ChimeRise:  {{Freq: 659, Ms: 55}, {Freq: 880, Ms: 55}, {Freq: 1319, Ms: 110}},
}

// WakeTone is what a slot set to t sounds like.
func Wake(t schema.Chime) []Note { return wakeTones[t] }

// Length is how long notes take to sound, rests included.
func Length(notes []Note) time.Duration {
	var ms int
	for _, n := range notes {
		ms += n.Ms
	}
	return time.Duration(ms) * time.Millisecond
}
