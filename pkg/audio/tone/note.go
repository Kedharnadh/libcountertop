// Package tone describes the notes used by hardware chime players.
package tone

// Note is one tone in a chime. A zero frequency is a rest.
type Note struct {
	Freq float64
	Ms   int
}
