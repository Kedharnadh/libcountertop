package tone

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type section struct{ at schema.Feedback }

func (s *section) Read() schema.Feedback { return s.at }
func (s *section) Update(change func(*schema.Feedback)) error {
	change(&s.at)
	return nil
}

type heard struct{ streams []schema.Stream }

func (h *heard) Chime(s schema.Stream, _ []Note) { h.streams = append(h.streams, s) }

func TestPreviewPlaysOnTheStreamAsked(t *testing.T) {
	h := &heard{}
	f := NewFeedback(&section{at: schema.DefaultFeedback()}, h, 0)

	f.Preview(schema.StreamAlerts)
	f.Preview(schema.StreamMain)
	f.Sound(ToneCancel)

	want := []schema.Stream{schema.StreamAlerts, schema.StreamFeedback, schema.StreamFeedback}
	if len(h.streams) != len(want) {
		t.Fatalf("heard %v, want %v", h.streams, want)
	}
	for i := range want {
		if h.streams[i] != want[i] {
			t.Errorf("chime %d on %s, want %s", i, h.streams[i], want[i])
		}
	}
}

func TestNoDeviceChimeSilencesEveryFeedbackSound(t *testing.T) {
	h := &heard{}
	f := NewFeedback(&section{at: schema.Feedback{Chime: schema.ChimeNone}}, h, 0)

	f.Preview(schema.StreamMedia)
	f.Sound(ToneVolume)
	f.Play(schema.StreamVoice, ToneTrouble)

	if len(h.streams) != 0 {
		t.Errorf("heard %v with the device chime off", h.streams)
	}
}
