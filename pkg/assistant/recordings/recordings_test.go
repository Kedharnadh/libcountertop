package recordings

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const rate = 16000

func somewhere(t *testing.T, keep ...int) *Store {
	t.Helper()

	return New(Options{
		Dir:  t.TempDir(),
		Rate: rate,
		Keeps: func(slot int) int {
			if slot < 0 || slot >= len(keep) {
				return 0
			}
			return keep[slot]
		},
	})
}

func record(t *testing.T, s *Store, id string, slot int, pcm []byte) {
	t.Helper()

	s.Opens(id, slot)
	s.Frame(pcm)
	s.Closes()
}

func kept(t *testing.T, s *Store) []string {
	t.Helper()

	entries, err := os.ReadDir(s.options.Dir)
	if err != nil {
		t.Fatal(err)
	}

	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

func TestOnlyTheNewestRecordingsAreKept(t *testing.T) {
	s := somewhere(t, 2)

	for _, id := range []string{"first", "second", "third"} {
		record(t, s, id, 0, []byte{1, 2, 3, 4})
	}

	got := kept(t, s)
	for _, name := range []string{"second.wav", "second.json", "third.wav", "third.json"} {
		if !slices.Contains(got, name) {
			t.Errorf("%s was pruned, and it is one of the newest two", name)
		}
	}
	for _, name := range []string{"first.wav", "first.json"} {
		if slices.Contains(got, name) {
			t.Errorf("%s survived, and it is the oldest of three", name)
		}
	}
}

func TestTheLastTurnSurvivesEvenWithinASecond(t *testing.T) {
	s := somewhere(t, 1)

	record(t, s, "zz-earlier", 0, []byte{1, 2})
	record(t, s, "aa-later", 0, []byte{3, 4})

	got := kept(t, s)
	if !slices.Contains(got, "aa-later.wav") {
		t.Error("the turn that came last was pruned")
	}
	if slices.Contains(got, "zz-earlier.wav") {
		t.Error("the turn that came first survived over a newer one")
	}
}

func TestEachAssistantIsPrunedAgainstItsOwnLimit(t *testing.T) {
	s := somewhere(t, 0, 1)

	record(t, s, "kept", 1, []byte{1, 2})
	record(t, s, "never", 0, []byte{1, 2})

	got := kept(t, s)
	if !slices.Contains(got, "kept.wav") {
		t.Error("the recording of an assistant set to keep one was pruned")
	}
	if slices.Contains(got, "never.wav") {
		t.Error("an assistant set to keep none recorded anyway")
	}
}

func TestAnOrphanedRecordingIsSweptUp(t *testing.T) {
	s := somewhere(t, 2)

	path := filepath.Join(s.options.Dir, "halfway.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o644); err != nil {
		t.Fatal(err)
	}

	s.Prune()

	if slices.Contains(kept(t, s), "halfway.wav") {
		t.Error("a recording with no sidecar was left behind")
	}
}

func TestAnEmptyTurnIsNotSaved(t *testing.T) {
	s := somewhere(t, 2)

	s.Opens("silent", 0)
	s.Closes()

	if got := kept(t, s); len(got) != 0 {
		t.Errorf("an empty turn left %v behind", got)
	}
}

func TestATurnStopsGrowingAtTheCap(t *testing.T) {
	s := somewhere(t, 1)

	s.Opens("long", 0)
	for range 4 {
		s.Frame(make([]byte, s.Longest()/2))
	}
	s.Closes()

	if got := s.Seconds("long"); got > LongestSeconds+1 {
		t.Errorf("a turn was kept for %.0fs, want the cap of %ds", got, LongestSeconds)
	}
}

func TestTheKeptCountIsSavedAndPrunes(t *testing.T) {
	keep := []int{2}
	var saved []int
	s := New(Options{
		Dir:   t.TempDir(),
		Rate:  rate,
		Slots: 1,
		Keeps: func(slot int) int { return keep[slot] },
		Save: func(slot, count int) error {
			saved = append(saved, count)
			keep[slot] = count
			return nil
		},
	})

	record(t, s, "first", 0, []byte{1, 2})
	record(t, s, "second", 0, []byte{1, 2})

	s.keep[0].OnCommand(1)

	if !slices.Equal(saved, []int{1}) {
		t.Errorf("saved %v, want [1]", saved)
	}
	if got := kept(t, s); slices.Contains(got, "first.wav") {
		t.Errorf("lowering the count left %v", got)
	}
}

func TestTheHeaderDescribesTheAudio(t *testing.T) {
	pcm := make([]byte, 320)
	got := wav(pcm, rate)

	if string(got[:4]) != "RIFF" || string(got[8:12]) != "WAVE" {
		t.Fatalf("not a wav: %q", got[:12])
	}
	if n := binary.LittleEndian.Uint16(got[22:24]); n != 1 {
		t.Errorf("%d channels, want mono", n)
	}
	if r := binary.LittleEndian.Uint32(got[24:28]); r != rate {
		t.Errorf("%d Hz, want %d", r, rate)
	}
	if bits := binary.LittleEndian.Uint16(got[34:36]); bits != 16 {
		t.Errorf("%d bits a sample, want 16", bits)
	}
	if size := binary.LittleEndian.Uint32(got[40:44]); int(size) != len(pcm) {
		t.Errorf("the header says %d bytes of audio and there are %d", size, len(pcm))
	}
	if len(got) != header+len(pcm) {
		t.Errorf("%d bytes for a %d byte recording", len(got), len(pcm))
	}
}
