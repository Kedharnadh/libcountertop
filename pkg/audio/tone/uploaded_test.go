package tone_test

import (
	"reflect"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/audio/tones"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type feedbackSection struct{ at schema.Feedback }

func (s *feedbackSection) Read() schema.Feedback                  { return s.at }
func (s *feedbackSection) Update(fn func(*schema.Feedback)) error { fn(&s.at); return nil }

type playback struct {
	sounds  []tone.Sound
	streams []schema.Stream
}

func (p *playback) Chime(s schema.Stream, notes []tone.Note) {
	p.Sound(s, tone.Level, tone.Sound{Notes: notes})
}
func (p *playback) Sound(s schema.Stream, _ float64, snd tone.Sound) {
	p.sounds = append(p.sounds, snd)
	p.streams = append(p.streams, s)
}

func TestUploadedFeedbackSelectionsAndFallbacks(t *testing.T) {
	store := tones.New(tones.Options{Dir: t.TempDir(), Rate: tones.Rate, Channels: 1})
	at := &feedbackSection{at: schema.DefaultFeedback()}
	out := &playback{}
	f := tone.NewFeedback(at, out, 0, tone.FeedbackOptions{Sounds: store, Mute: true, Hold: true})
	store.Changed.Listen(func(struct{}) { f.Refresh() })
	selector := f.Entity("chime").(*esphome.Select)
	if err := store.Write("bell", "Uploaded bell", 0, 1, []byte{1, 0, 2, 0}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, option := range selector.Options {
		if option == "Uploaded bell" {
			found = true
		}
	}
	if !found {
		t.Fatalf("uploaded sound missing from refreshed options: %v", selector.Options)
	}
	occasions := []struct {
		name     string
		play     func()
		fallback schema.Chime
	}{
		{"chime", func() { f.Preview(schema.StreamMain) }, schema.DefaultChime},
		{"failure", f.Failure, schema.ChimeTrouble},
		{"cancel", f.Cancelled, schema.ChimeCancel},
		{"mute", f.Muted, schema.ChimeMuted},
		{"unmute", f.Unmuted, schema.ChimeUnmuted},
		{"hold", f.Hold, schema.ChimeMuteHold},
	}
	for _, occasion := range occasions {
		if err := f.Change(occasion.name, "Uploaded bell"); err != nil {
			t.Fatal(err)
		}
		occasion.play()
		got := out.sounds[len(out.sounds)-1]
		if !reflect.DeepEqual(got.Clip, []int16{1, 2}) {
			t.Errorf("%s played %+v, want uploaded clip", occasion.name, got)
		}
	}
	if err := store.Remove("bell"); err != nil {
		t.Fatal(err)
	}
	for _, option := range selector.Options {
		if option == "Uploaded bell" {
			t.Error("removed sound remains selectable")
		}
	}
	for _, occasion := range occasions {
		occasion.play()
		got := out.sounds[len(out.sounds)-1]
		if !reflect.DeepEqual(got.Notes, tone.Wake(occasion.fallback)) {
			t.Errorf("%s fallback = %+v", occasion.name, got)
		}
	}
	// The global silence switch still suppresses regular feedback; hold stays independent.
	f.SetChime(schema.ChimeNone)
	count := len(out.sounds)
	f.Failure()
	f.Cancelled()
	f.Muted()
	f.Unmuted()
	f.Preview(schema.StreamAlerts)
	if len(out.sounds) != count {
		t.Error("device chime None did not silence feedback")
	}
	f.Hold()
	if len(out.sounds) != count+1 {
		t.Error("mute hold was incorrectly silenced by device chime None")
	}
	if err := f.Change("hold", "none"); err != nil {
		t.Fatal(err)
	}
	count = len(out.sounds)
	f.Hold()
	if len(out.sounds) != count {
		t.Error("hold None played a sound")
	}
}

func TestOldFeedbackSettingsKeepDefaultOccasionSounds(t *testing.T) {
	out := &playback{}
	f := tone.NewFeedback(
		&feedbackSection{at: schema.Feedback{Chime: schema.ChimeDing}},
		out,
		0,
		tone.FeedbackOptions{},
	)
	f.Failure()
	f.Cancelled()
	f.Muted()
	f.Unmuted()
	f.Hold()
	want := [][]tone.Note{
		tone.ToneTrouble,
		tone.ToneCancel,
		tone.ToneMute,
		tone.ToneUnmute,
		tone.ToneMuteHold,
	}
	for i, notes := range want {
		if !reflect.DeepEqual(out.sounds[i].Notes, notes) {
			t.Errorf("occasion %d = %+v, want %v", i, out.sounds[i], notes)
		}
	}
}

func TestUploadedMuteSoundKeepsPairedFallback(t *testing.T) {
	store := tones.New(tones.Options{Dir: t.TempDir(), Rate: tones.Rate, Channels: 1})
	if err := store.Write("bell", "Bell", 0, 1, []byte{1, 0, 2, 0}); err != nil {
		t.Fatal(err)
	}
	table := tone.MuteTable(func(c *schema.Chime) *schema.Chime { return c }, store.Choices)
	var selected schema.Chime
	if err := table.MustRow("mute_sound").Write(&selected, "tone:bell"); err != nil {
		t.Fatal(err)
	}
	for _, muted := range []bool{true, false} {
		if got := tone.MuteSound(
			store,
			selected,
			muted,
		); !reflect.DeepEqual(
			got.Clip,
			[]int16{1, 2},
		) {
			t.Errorf("muted=%v played %+v", muted, got)
		}
	}
	if err := store.Remove("bell"); err != nil {
		t.Fatal(err)
	}
	for _, muted := range []bool{true, false} {
		want := tone.ToneMute
		if !muted {
			want = tone.ToneUnmute
		}
		for _, choice := range []schema.Chime{selected, schema.ChimeMuteSwitch} {
			if got := tone.MuteSound(store, choice, muted); !reflect.DeepEqual(got.Notes, want) {
				t.Errorf("choice=%s muted=%v played %+v, want %v", choice, muted, got, want)
			}
		}
		if got := tone.MuteSound(store, schema.ChimeNone, muted); !got.Empty() {
			t.Errorf("None played %+v", got)
		}
	}
}
