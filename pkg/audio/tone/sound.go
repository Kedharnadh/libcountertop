package tone

import "time"

type Sound struct {
	Notes []Note

	Clip     []int16
	Rate     int
	Channels int
}

func (s Sound) Empty() bool { return len(s.Notes) == 0 && len(s.Clip) == 0 }

func (s Sound) Length() time.Duration {
	if len(s.Clip) == 0 || s.Rate <= 0 || s.Channels <= 0 {
		return Length(s.Notes)
	}
	return time.Duration(len(s.Clip)/s.Channels) * time.Second / time.Duration(s.Rate)
}
