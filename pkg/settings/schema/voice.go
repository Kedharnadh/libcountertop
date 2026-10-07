package schema

import (
	"github.com/ygelfand/libcountertop/pkg/say"
)

// Chime is what the device sounds like when it acknowledges something.
//
// A choice rather than a switch, because the device sits in a room with other devices in it and a
// beep that is distinguishable from the microwave is worth more than a beep. None is one of the
// answers: a display on a bedside table should be able to say nothing at all.
type Chime string

const (
	// ChimeNone is silence. It covers every sound the device makes about itself, not only the
	// acknowledgement, because somebody who turned the chimes off wants the device quiet rather
	// than quieter.
	ChimeNone Chime = "none"

	// ChimeChirp is two quick rising notes.
	ChimeChirp Chime = "chirp"

	// ChimeDing is one clear note, for a room where the chirp reads as a notification.
	ChimeDing Chime = "ding"

	// ChimeRise is three rising notes, which is the most this should ever be: an acknowledgement
	// longer than a moment is in the way of the thing it is acknowledging.
	ChimeRise       Chime = "rise"
	ChimeMuteSwitch Chime = "mute"
)

// Label is how the setting is shown.
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
	}
	return say.T("chime.chirp")
}

// Chimes is every value the setting takes, in the order they are offered. None first, because it is
// the one somebody goes looking for.
func Chimes() []Chime { return []Chime{ChimeNone, ChimeChirp, ChimeDing, ChimeRise} }

// Silent reports whether the device should make no sound of its own.
func (c Chime) Silent() bool { return c == ChimeNone }

// Delivery is how a spoken reply reaches the device. It is per slot because a local pipeline and a
// cloud one differ in how long the audio takes to start, so the trade between starting sooner and
// not gapping is not the same for both.
type Delivery string

const (
	// DeliveryWhole fetches the reply from the url Home Assistant serves it at. It cannot gap and it
	// says when the audio has ended, which the stream does not.
	DeliveryWhole Delivery = "whole"

	// DeliveryStream takes the reply over the API as it is generated. It starts sooner and it can
	// gap: the chunks arrive at about the rate they play, so any hiccup splices silence into a word.
	DeliveryStream Delivery = "stream"
)

// Label is how the setting is shown.
func (d Delivery) Label() string {
	switch d {
	case DeliveryWhole:
		return say.T("delivery.whole")
	case DeliveryStream:
		return say.T("delivery.stream")
	}
	return string(d)
}

// Deliveries is how a reply can arrive, in the order it is offered.
func Deliveries() []Delivery { return []Delivery{DeliveryWhole, DeliveryStream} }
