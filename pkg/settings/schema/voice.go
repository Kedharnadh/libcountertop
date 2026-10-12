package schema

import (
	"strings"

	"github.com/ygelfand/libcountertop/pkg/say"
)

type Chime string

const customChime = "tone:"

func CustomChime(key string) Chime { return Chime(customChime + key) }

func (c Chime) Custom() (key string, ok bool) { return strings.CutPrefix(string(c), customChime) }

const (
	ChimeNone Chime = "none"

	ChimeChirp Chime = "chirp"

	ChimeDing Chime = "ding"

	ChimeRise       Chime = "rise"
	ChimeTimer      Chime = "timer"
	ChimeTrouble    Chime = "trouble"
	ChimeMuteSwitch Chime = "mute"
	ChimeMuted      Chime = "muted"
	ChimeUnmuted    Chime = "unmuted"
	ChimeMuteHold   Chime = "mute_hold"
	ChimeCancel     Chime = "cancel"
	ChimeRing       Chime = "ring"
	ChimeRingback   Chime = "ringback"
)

func (c Chime) Label() string {
	if key, ok := c.Custom(); ok {
		return key
	}
	switch c {
	case ChimeMuted:
		return "Mute"
	case ChimeUnmuted:
		return "Unmute"
	case ChimeMuteHold:
		return "Mute hold"
	case ChimeCancel:
		return "Cancel"
	case ChimeMuteSwitch:
		return "Mute tones"
	case ChimeNone:
		return say.T("chime.none")
	case ChimeDing:
		return say.T("chime.ding")
	case ChimeRise:
		return say.T("chime.rise")
	case ChimeTimer:
		return say.T("chime.timer")
	case ChimeTrouble:
		return say.T("chime.trouble")
	case ChimeRing:
		return say.T("chime.ring")
	case ChimeRingback:
		return say.T("chime.ringback")
	}
	return say.T("chime.chirp")
}

func Chimes() []Chime { return []Chime{ChimeNone, ChimeChirp, ChimeDing, ChimeRise} }

func AlertChimes() []Chime { return append(Chimes(), ChimeTimer, ChimeTrouble) }

func CallChimes() []Chime { return append(AlertChimes(), ChimeRing, ChimeRingback) }

func (c Chime) Silent() bool { return c == ChimeNone }

type Delivery string

const (
	DeliveryWhole Delivery = "whole"

	DeliveryStream Delivery = "stream"
)

func (d Delivery) Label() string {
	switch d {
	case DeliveryWhole:
		return say.T("delivery.whole")
	case DeliveryStream:
		return say.T("delivery.stream")
	}
	return string(d)
}

func Deliveries() []Delivery { return []Delivery{DeliveryWhole, DeliveryStream} }
