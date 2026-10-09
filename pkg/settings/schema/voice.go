package schema

import (
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Chime string

const (
	ChimeNone Chime = "none"

	ChimeChirp Chime = "chirp"

	ChimeDing Chime = "ding"

	ChimeRise       Chime = "rise"
	ChimeTimer      Chime = "timer"
	ChimeTrouble    Chime = "trouble"
	ChimeMuteSwitch Chime = "mute"
)

func (c Chime) Label() string {
	switch c {
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
	}
	return say.T("chime.chirp")
}

func Chimes() []Chime { return []Chime{ChimeNone, ChimeChirp, ChimeDing, ChimeRise} }

func AlertChimes() []Chime { return append(Chimes(), ChimeTimer, ChimeTrouble) }

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
