package a2dp

import (
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	speaker "github.com/ygelfand/libcountertop/pkg/audio/resample"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sbc"
	fixtures "github.com/ygelfand/libcountertop/pkg/bluetooth/sbc/testdata"
)

func fed(t *testing.T, s *Sink) int {
	t.Helper()

	coded, err := fixtures.Read("tone.sbc")
	if err != nil {
		t.Fatal(err)
	}

	at, ok := sbc.Find(coded)
	if !ok {
		t.Fatal("no frame in the vector")
	}
	fr, err := sbc.Unpack(coded[at:])
	if err != nil {
		t.Fatal(err)
	}

	s.said = time.Now()

	before := s.samples
	s.play(coded[at : at+fr.Header.Length()])
	return s.samples - before
}

func TestNothingIsPlayedWhileStoodDown(t *testing.T) {
	s := testSink()
	s.start("stand test")
	s.resample = speaker.NewRational(44100, Rate, 2)

	if got := fed(t, s); got == 0 {
		t.Fatal("nothing reached the card while standing")
	}

	s.Stand(true)
	if got := fed(t, s); got != 0 {
		t.Errorf("%d samples reached the card while stood down", got)
	}

	s.Stand(false)
	if got := fed(t, s); got == 0 {
		t.Error("nothing reached the card after standing back up")
	}
}

func TestTheSameStandingTwiceIsOnce(t *testing.T) {
	s := testSink()
	s.start("repeat test")

	s.Stand(true)
	s.Stand(true)
	s.Stand(true)

	s.mu.Lock()
	down := s.down
	s.mu.Unlock()

	if !down {
		t.Error("a repeated standing lost the standing")
	}

	s.Stand(false)
	s.Stand(false)

	s.mu.Lock()
	down = s.down
	s.mu.Unlock()

	if down {
		t.Error("standing back up twice left it down")
	}
}

func TestDuckingQuietensRatherThanStops(t *testing.T) {
	s := testSink()
	s.start("duck test")
	s.resample = speaker.NewRational(44100, Rate, 2)

	s.Duck(true)
	if got := fed(t, s); got == 0 {
		t.Error("ducking stopped the audio instead of quietening it")
	}

	s.mu.Lock()
	gain := s.gain
	s.mu.Unlock()

	if gain != ducking.Gain(-15) {
		t.Errorf("ducked to %v, want %v", gain, ducking.Gain(-15))
	}

	s.Duck(false)

	s.mu.Lock()
	gain = s.gain
	s.mu.Unlock()

	if gain != 1 {
		t.Errorf("unducked to %v, want 1", gain)
	}
}
