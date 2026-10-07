package turn

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Chimes interface{ Chime([]tone.Note) }
type Words struct {
	Settings storage.Section[schema.Wake]
	Speaker  Chimes
}

func (w Words) MaxListen(slot int) time.Duration {
	return time.Duration(w.Settings.Read().Slot(slot).MaxListen) * time.Second
}
func (w Words) MaxThink(slot int) time.Duration {
	return time.Duration(w.Settings.Read().Slot(slot).MaxThink) * time.Second
}
func (w Words) FollowUp(slot int) time.Duration {
	return time.Duration(w.Settings.Read().Slot(slot).FollowUp) * time.Second
}
func (w Words) Buffer(slot int) time.Duration {
	return time.Duration(w.Settings.Read().Slot(slot).Buffer) * time.Millisecond
}
func (w Words) ChimeLength(slot int) time.Duration {
	return tone.Length(tone.Wake(w.Settings.Read().Slot(slot).Tone))
}
func (w Words) Delivery(slot int) string { return string(w.Settings.Read().Slot(slot).Delivery) }
func (w Words) Tones(slot int) bool      { return !w.Settings.Read().Slot(slot).Tone.Silent() }
func (w Words) Chime(slot int)           { w.Speaker.Chime(tone.Wake(w.Settings.Read().Slot(slot).Tone)) }
