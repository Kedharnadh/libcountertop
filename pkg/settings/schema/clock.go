package schema

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/clockkind"
	"github.com/ygelfand/libcountertop/pkg/say"
)

// Clock is how the dashboard draws the time.
//
// Its own area rather than part of Screen, because Screen is the panel — how bright it is, which
// way the drawer comes in — and this is what is drawn on it. The two change for different reasons
// and there is more coming here than there is there.
type Clock struct {
	Face Face `json:"face"`

	// Date puts the day under the time. On a device that sits in a room it is most of what the
	// screen is for besides the clock, and it is also the first thing somebody wanting only the
	// time will turn off.
	Date bool `json:"date"`

	// Position is where on the glass the clock sits, and Size is how much of it the clock takes.
	//
	// A pair, and the reason neither is the other: position without size is a clock that can only
	// be large, and size without position is a small clock stranded in the middle of a tall panel.
	Position Position `json:"position"`
	Size     Size     `json:"size"`

	// Ink is the color the clock is drawn in, over whatever theme is on.
	Ink Ink `json:"ink"`
}

// Delay is how long the device waits, with nobody touching it, before the idle screen comes up.
type Delay string

const (
	DelayNever Delay = "never"
	Delay30s   Delay = "30s"
	Delay1m    Delay = "1m"
	Delay2m    Delay = "2m"
	Delay3m    Delay = "3m"
	Delay5m    Delay = "5m"
	Delay10m   Delay = "10m"
	Delay15m   Delay = "15m"
)

// DefaultDelay is a minute: long enough that somebody reading the screen is not interrupted, short
// enough that a device left alone settles while they are still in the room to see it.
const DefaultDelay = Delay1m

func (d Delay) Label() string {
	switch d {
	case DelayNever:
		return say.T("delay.never")
	case Delay30s:
		return say.T("delay.30s")
	case Delay1m:
		return say.T("delay.1m")
	case Delay2m:
		return say.T("delay.2m")
	case Delay3m:
		return say.T("delay.3m")
	case Delay5m:
		return say.T("delay.5m")
	case Delay10m:
		return say.T("delay.10m")
	case Delay15m:
		return say.T("delay.15m")
	}
	return string(d)
}

// After is the wait as a duration. Never is zero, which every caller reads as not sleeping at all
// rather than as sleeping immediately.
func (d Delay) After() time.Duration {
	switch d {
	case Delay30s:
		return 30 * time.Second
	case Delay1m:
		return time.Minute
	case Delay2m:
		return 2 * time.Minute
	case Delay3m:
		return 3 * time.Minute
	case Delay5m:
		return 5 * time.Minute
	case Delay10m:
		return 10 * time.Minute
	case Delay15m:
		return 15 * time.Minute
	}
	return 0
}

// Delays is every wait that can be chosen, shortest first, with Never at the end: it is the one
// somebody picks on purpose rather than the one they are looking for.
func Delays() []Delay {
	return []Delay{Delay30s, Delay1m, Delay2m, Delay5m, Delay15m, DelayNever}
}

func MediaDelays() []Delay {
	return []Delay{Delay1m, Delay3m, Delay5m, Delay10m, Delay15m, DelayNever}
}

// Position is the part of the screen the clock is given.
//
// What size is for on a device that stands on something: a clock read from a bed wants to be low,
// one on a kitchen shelf high and out of the way of whatever is under it.
type Position string

const (
	PositionTop    Position = "top"
	PositionCenter Position = "center"
	PositionBottom Position = "bottom"
)

const DefaultPosition = PositionCenter

func (p Position) Label() string {
	switch p {
	case PositionTop:
		return say.T("position.top")
	case PositionCenter:
		return say.T("position.center")
	case PositionBottom:
		return say.T("position.bottom")
	}
	return string(p)
}

// Positions is every place the clock can sit, in the order they are on the screen.
func Positions() []Position {
	return []Position{PositionTop, PositionCenter, PositionBottom}
}

// Size is how much of the room it is given the clock fills.
//
// A share rather than a height in pixels. Every face already fits itself to the box it is handed
// and the two orientations are different shapes, so a number of pixels would mean one thing
// standing up and another lying down. A share means the same thing both ways.
type Size string

const (
	SizeNano   Size = "nano"
	SizeMicro  Size = "micro"
	SizeMini   Size = "mini"
	SizeSmall  Size = "small"
	SizeMedium Size = "medium"
	SizeLarge  Size = "large"
)

// DefaultSize is large. This is a clock on a shelf read from across a room, and the face filling
// what it is given is what the device did before there was a choice.
const DefaultSize = SizeLarge

func (s Size) Label() string {
	switch s {
	case SizeNano:
		return say.T("size.nano")
	case SizeMicro:
		return say.T("size.micro")
	case SizeMini:
		return say.T("size.mini")
	case SizeSmall:
		return say.T("size.small")
	case SizeMedium:
		return say.T("size.medium")
	case SizeLarge:
		return say.T("size.large")
	}
	return string(s)
}

// Share is how much of each side of the box the clock keeps.
//
// Both sides, so the clock scales rather than stretches. Small is not half: a face fits itself to
// whichever side runs out first, so halving the box is nearer a quarter of the clock, and small
// still has to be legible from the far side of a room.
func (s Size) Share() float64 {
	switch s {
	case SizeNano:
		return 0.16
	case SizeMicro:
		return 0.26
	case SizeMini:
		return 0.40
	case SizeSmall:
		return 0.58
	case SizeMedium:
		return 0.78
	}
	return 1
}

// Sizes is every size, smallest first.
func Sizes() []Size {
	return []Size{SizeNano, SizeMicro, SizeMini, SizeSmall, SizeMedium, SizeLarge}
}

// DefaultFace is what the device has always drawn, so an upgrade changes nothing about a device
// somebody has already put on a shelf.
const DefaultFace = FacePlain

// DefaultDate is on, for the same reason.
const DefaultDate = true

func DefaultClock() Clock {
	return Clock{
		Face:     DefaultFace,
		Date:     DefaultDate,
		Position: DefaultPosition,
		Size:     DefaultSize,
		Ink:      DefaultInk,
	}
}

// Face is a way of drawing the clock. The name is written to the file and branched on; the label is
// what Home Assistant shows.
//
// A face is a layout and the numerals it is drawn in together, not two settings multiplied: offering
// both is twenty combinations nobody asked for, and the pairs that read well are a short list
// somebody chose.
type Face = clockkind.Kind

const (
	FaceNone          Face = Face(clockkind.None)
	FacePlain         Face = Face(clockkind.Plain)
	FaceStack         Face = Face(clockkind.Stack)
	FaceCards         Face = Face(clockkind.Cards)
	FaceAnalog        Face = Face(clockkind.Analog)
	FaceAnalogSeconds Face = Face(clockkind.AnalogSeconds)
	FaceSegments      Face = Face(clockkind.Segments)
	FaceWords         Face = Face(clockkind.Words)
	FaceOverlap       Face = Face(clockkind.Overlap)
)

var Faces = clockkind.Kinds

// ClockWriter changes how the time is drawn.
