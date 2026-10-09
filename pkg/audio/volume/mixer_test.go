package volume

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type fakeOutput struct {
	levels map[schema.Stream]int
	muted  map[schema.Stream]bool
}

func newFakeOutput() *fakeOutput {
	return &fakeOutput{levels: map[schema.Stream]int{}, muted: map[schema.Stream]bool{}}
}

func (o *fakeOutput) SetLevel(s schema.Stream, l int)   { o.levels[s] = l }
func (o *fakeOutput) Mute(s schema.Stream, on bool)     { o.muted[s] = on }
func (o *fakeOutput) Muted(s schema.Stream) bool        { return o.muted[s] }
func (o *fakeOutput) level(s schema.Stream) (int, bool) { l, ok := o.levels[s]; return l, ok }

func newTestMixer(out Output, saved map[schema.Stream]int) *Mixer {
	return NewMixer(MixerOptions{
		Output:  out,
		Streams: []schema.Stream{schema.StreamMedia, schema.StreamVoice},
		Save: func(s schema.Stream, l int) error {
			saved[s] = l
			return nil
		},
	})
}

func TestMixerRestoreAppliesWithoutSavingOrAnnouncing(t *testing.T) {
	out, saved := newFakeOutput(), map[schema.Stream]int{}
	m := newTestMixer(out, saved)
	var heard int
	m.Changed.Listen(func(Change) { heard++ })

	m.Restore(func(s schema.Stream) int {
		if s == schema.StreamMedia {
			return 40
		}
		return 70
	})

	if l, _ := out.level(schema.StreamMedia); l != 40 || m.Level(schema.StreamVoice) != 70 {
		t.Errorf("restored media %d voice %d", l, m.Level(schema.StreamVoice))
	}
	if len(saved) != 0 || heard != 0 {
		t.Errorf("restore saved %v and announced %d times", saved, heard)
	}
	if n := m.Number(schema.StreamMedia); n == nil || n.ObjectID != "volume_media" {
		t.Error("no volume_media entity")
	}
}

func TestMixerSetClampsSavesAndAnnounces(t *testing.T) {
	out, saved := newFakeOutput(), map[schema.Stream]int{}
	m := newTestMixer(out, saved)
	var got []Change
	m.Changed.Listen(func(c Change) { got = append(got, c) })

	m.Set(schema.StreamVoice, 140)
	m.Set(schema.StreamVoice, 100)

	if l, _ := out.level(schema.StreamVoice); l != 100 || saved[schema.StreamVoice] != 100 {
		t.Errorf("output %d saved %d, want 100", l, saved[schema.StreamVoice])
	}
	if len(got) != 1 || got[0] != (Change{Stream: schema.StreamVoice, Level: 100}) {
		t.Errorf("announced %v, want one change to 100", got)
	}
}

func TestMixerIgnoresStreamsItDoesNotHold(t *testing.T) {
	out, saved := newFakeOutput(), map[schema.Stream]int{}
	m := newTestMixer(out, saved)

	m.Set(schema.StreamAlerts, 50)

	if _, ok := out.level(schema.StreamAlerts); ok || len(saved) != 0 {
		t.Error("a stream outside the mixer was applied")
	}
}

func TestMixerMuteKeepsTheSavedLevel(t *testing.T) {
	out, saved := newFakeOutput(), map[schema.Stream]int{}
	m := newTestMixer(out, saved)
	m.Set(schema.StreamMedia, 60)
	m.Set(schema.StreamVoice, 30)
	clear(saved)

	m.Mute(schema.StreamMedia, true)
	if !m.Muted(schema.StreamMedia) || m.Muted(schema.StreamVoice) {
		t.Error("mute reached the wrong stream")
	}
	if m.Level(schema.StreamMedia) != 60 || len(saved) != 0 {
		t.Errorf("muting changed the level to %d or saved %v", m.Level(schema.StreamMedia), saved)
	}

	m.Mute(schema.StreamMedia, false)
	if m.Muted(schema.StreamMedia) || m.Level(schema.StreamMedia) != 60 {
		t.Errorf("unmuted at %d", m.Level(schema.StreamMedia))
	}
}
