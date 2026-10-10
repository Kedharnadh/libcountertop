package tones

import (
	"encoding/binary"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func pcm(samples int) []byte {
	out := make([]byte, 0, samples*2)
	for i := range samples {
		out = binary.LittleEndian.AppendUint16(out, uint16(int16(i%2000-1000)))
	}
	return out
}

func send(t *testing.T, s *Store, key, name string, data []byte) error {
	t.Helper()
	pages := (len(data) + Page - 1) / Page
	for p := range pages {
		end := min((p+1)*Page, len(data))
		if err := s.Write(key, name, p, pages, data[p*Page:end]); err != nil {
			return err
		}
	}
	return nil
}

func TestAToneSentInPagesBecomesAChoiceAtTheDevicesFormat(t *testing.T) {
	s := New(Options{Dir: t.TempDir(), Rate: 48000, Channels: 2})
	changed := 0
	s.Changed.Listen(func(struct{}) { changed++ })

	if err := send(t, s, "doorbell", "Doorbell chime", pcm(Rate)); err != nil {
		t.Fatal(err)
	}
	if changed != 1 {
		t.Errorf("changed %d times, want once", changed)
	}
	choices := s.Choices()
	if len(choices) != 1 || choices[0].Label != "Doorbell chime" ||
		choices[0].Value != string(schema.CustomChime("doorbell")) {
		t.Fatalf("choices %+v", choices)
	}
	snd := s.Sound(schema.CustomChime("doorbell"), schema.ChimeDing)
	if len(snd.Clip) != 2*Rate || snd.Channels != 2 {
		t.Errorf(
			"clip %d samples on %d channels, want a second of stereo",
			len(snd.Clip),
			snd.Channels,
		)
	}
	if got := snd.Length().Seconds(); got != 1 {
		t.Errorf("length %vs, want 1s", got)
	}
}

func TestAToneSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	if err := send(
		t,
		New(Options{Dir: dir, Rate: 16000, Channels: 1}),
		"gong",
		"Soft gong",
		pcm(Rate/2),
	); err != nil {
		t.Fatal(err)
	}
	again := New(Options{Dir: dir, Rate: 16000, Channels: 1})
	if list := again.List(); len(list) != 1 || list[0].Name != "Soft gong" {
		t.Fatalf("after a restart the list is %+v", list)
	}
	if snd := again.Sound(schema.CustomChime("gong"), schema.ChimeDing); len(snd.Clip) == 0 {
		t.Error("the tone did not play after a restart")
	}
}

func TestAMissingPageIsRefused(t *testing.T) {
	s := New(Options{Dir: t.TempDir(), Rate: 48000, Channels: 1})
	data := pcm(Page)
	if err := s.Write("ring", "Ring", 0, 3, data[:Page]); err != nil {
		t.Fatal(err)
	}
	err := s.Write("ring", "Ring", 2, 3, data[Page:])
	if err == nil || !strings.Contains(err.Error(), "missing page 1") {
		t.Errorf("err %v, want the missing page named", err)
	}
	if len(s.List()) != 0 {
		t.Error("an incomplete tone was kept")
	}
}

func TestATooLongToneIsRefused(t *testing.T) {
	s := New(Options{Dir: t.TempDir(), Rate: 48000, Channels: 1})
	err := send(t, s, "song", "Song", pcm(int(Longest.Seconds())*Rate+Rate))
	if err == nil {
		t.Fatal("a tone over the limit was taken")
	}
}

func TestABadKeyIsRefused(t *testing.T) {
	s := New(Options{Dir: t.TempDir(), Rate: 48000, Channels: 1})
	for _, key := range []string{"", "../escape", "Upper", "a b"} {
		if err := s.Write(key, "x", 0, 1, pcm(10)); err == nil {
			t.Errorf("key %q was taken", key)
		}
	}
}

func TestRenameKeepsTheKeyAndRemoveFallsBack(t *testing.T) {
	s := New(Options{Dir: t.TempDir(), Rate: 48000, Channels: 1})
	if err := send(t, s, "chime", "Chime", pcm(100)); err != nil {
		t.Fatal(err)
	}
	if err := s.Rename("chime", "Kitchen chime"); err != nil {
		t.Fatal(err)
	}
	if got := s.Choices(); len(got) != 1 || got[0].Label != "Kitchen chime" ||
		got[0].Value != string(schema.CustomChime("chime")) {
		t.Errorf("after rename %+v", got)
	}
	changed := 0
	s.Changed.Listen(func(struct{}) { changed++ })
	if err := s.Remove("chime"); err != nil {
		t.Fatal(err)
	}
	if err := s.Remove("chime"); err != nil || changed != 1 {
		t.Errorf("removing it again gave %v and %d changes, want nothing new", err, changed)
	}
	snd := s.Sound(schema.CustomChime("chime"), schema.ChimeDing)
	if len(snd.Clip) != 0 || len(snd.Notes) != len(tone.Wake(schema.ChimeDing)) {
		t.Errorf("a removed tone played %+v, want the fallback", snd)
	}
}
