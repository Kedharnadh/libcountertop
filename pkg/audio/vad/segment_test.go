package vad

import (
	"math/rand/v2"
	"testing"
	"time"
)

func silence(seconds float64) []int16 { return make([]int16, int(seconds*SampleRate)) }

func noise(seconds float64, level float64) []int16 {
	r := rand.New(rand.NewPCG(3, 4))
	out := make([]int16, int(seconds*SampleRate))
	for i := range out {
		out[i] = int16(r.NormFloat64() * level)
	}
	return out
}

func feed(t *testing.T, parts ...[]int16) []Clip {
	t.Helper()
	s, err := NewSegmenter()
	if err != nil {
		t.Fatal(err)
	}
	var clips []Clip
	for _, p := range parts {
		for len(p) > 0 {
			n := min(160, len(p))
			clips = append(clips, s.Feed(p[:n])...)
			p = p[n:]
		}
	}
	return clips
}

func voiced(t *testing.T) []int16 { return speech(t)[18*Chunk : 122*Chunk] }

func TestSpeechBetweenSilencesIsOneClip(t *testing.T) {
	words := voiced(t)[:2*SampleRate]
	clips := feed(t, silence(1), words, silence(1))
	if len(clips) != 1 {
		t.Fatalf("%d clips, want 1", len(clips))
	}
	c := clips[0]
	if c.Start < time.Second-chunkTime(leadIn) || c.Start > time.Second+chunkTime(3) ||
		c.End < 3*time.Second+chunkTime(tail-2) || c.End > 3*time.Second+chunkTime(tail+3) {
		t.Errorf("clip %v-%v, want the speech at 1s-3s with its lead-in and tail", c.Start, c.End)
	}
	if want := int((c.End - c.Start) * SampleRate / time.Second); len(c.Samples) != want {
		t.Errorf("%d samples for %v, want %d", len(c.Samples), c.End-c.Start, want)
	}
}

func TestNoiseAndSilenceMakeNoClips(t *testing.T) {
	if clips := feed(t, silence(3), noise(5, 300), silence(1)); len(clips) != 0 {
		t.Errorf("%d clips from noise and silence", len(clips))
	}
}

func TestSpeechLongerThanTheLimitIsDropped(t *testing.T) {
	words := voiced(t)
	long := append(append([]int16{}, words...), words...)
	if clips := feed(t, silence(1), long, silence(1)); len(clips) != 0 {
		t.Errorf(
			"%d clips from %.1fs of speech, want none over %v",
			len(clips),
			float64(len(long))/SampleRate,
			longest,
		)
	}
}

func TestResetDropsAHalfFinishedClip(t *testing.T) {
	s, err := NewSegmenter()
	if err != nil {
		t.Fatal(err)
	}
	s.Feed(silence(1))
	s.Feed(speech(t)[:SampleRate])
	s.Reset()
	if clips := s.Feed(silence(1)); len(clips) != 0 {
		t.Errorf("%d clips after Reset mid-speech", len(clips))
	}
}
