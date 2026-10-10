package turn

import (
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Chimes interface {
	Sound(s schema.Stream, level float64, snd tone.Sound)
}

type Sounds interface {
	Sound(c, fallback schema.Chime) tone.Sound
	Choices() []setting.Option
}

type Words struct {
	Settings storage.Section[schema.Wake]
	Speaker  Chimes
	Sounds   Sounds
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
	return w.sound(slot).Length()
}
func (w Words) Delivery(slot int) string { return string(w.Settings.Read().Slot(slot).Delivery) }
func (w Words) Tones(slot int) bool      { return !w.Settings.Read().Slot(slot).Tone.Silent() }

func (w Words) ToneChoices() []setting.Option {
	if w.Sounds == nil {
		return nil
	}
	return w.Sounds.Choices()
}

func (w Words) Chime(slot int) {
	w.Speaker.Sound(schema.StreamFeedback, tone.Level, w.sound(slot))
}

func (w Words) sound(slot int) tone.Sound {
	c := w.Settings.Read().Slot(slot).Tone
	if w.Sounds != nil {
		return w.Sounds.Sound(c, schema.DefaultTone)
	}
	if _, custom := c.Custom(); custom {
		c = schema.DefaultTone
	}
	return tone.Sound{Notes: tone.Wake(c)}
}
