package source

import (
	"time"
)

// Now is what is playing, for the entity and for the screen.
type Now struct {
	// Playing and Paused are mutually exclusive playback states.
	Playing bool
	Paused  bool

	// Hold keeps the card up through a gap, a cast session between tracks.
	Hold bool

	// Empty where the source does not know, which is anything played from a url.
	Title  string
	Artist string
	Album  string

	// Art is the album art as it arrived, nil where there is none, and ArtID changes with it.
	// Bytes rather than a picture: this package is on every board and the decoder is not.
	Art   []byte
	ArtID uint64

	// Mark is the app's own icon, where the source is an app, as bytes like Art.
	Mark   []byte
	MarkID uint64

	// Elapsed is how far into the track playback has reached, and Length how long it runs for. A
	// zero Length is a source that does not say.
	Elapsed time.Duration
	Length  time.Duration

	// LiveWithin is how far behind live a live stream may be and still read as live, zero for one
	// that is not live.
	LiveWithin time.Duration

	// Queue is what plays next, where the source knows.
	Queue []Track

	// Can is the transport to offer.
	Can Controls
}

// Track is one thing in a queue.
type Track struct {
	Title  string
	Artist string
	Length time.Duration

	Art   []byte
	ArtID uint64

	// Play jumps to it, nil where the source cannot.
	Play func()
}

// Seeker is a source that can move within what it is playing.
type Seeker interface {
	Seek(to time.Duration)
	CanSeek() bool
}

// Opener is a source with a screen of its own, which it shows instead of the player's card.
type Opener interface {
	Open() bool
}

// Controls is which transport a source answers to.
type Controls uint8

const (
	CanPause Controls = 1 << iota
	CanStop
	CanNext
	CanPrevious
)

// Has reports whether every control in want is offered.
func (c Controls) Has(want Controls) bool { return c&want == want }

// Kind is where the audio came from.
type Kind uint8

const (
	FromQueue Kind = iota
	FromBluetooth
	FromGroup
	FromCast
)

// Details is a source that says what it is playing and answers the transport for it.
type Details interface {
	Now() Now
	Next()
	Previous()

	// Kind is which of them this is, for the mark on the card.
	Kind() Kind

	// Label is what to call it: the group's own name. Empty where it has none.
	Label() string
}

// Controller is the common playback transport.
type Controller interface {
	Play()
	Pause()
	Stop()
	Next()
	Previous()
}

// Source supplies transport controls and a playback snapshot.
type Source interface {
	Controller
	Details
}
