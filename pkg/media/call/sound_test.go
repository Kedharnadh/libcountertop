package call

import (
	"context"
	"slices"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type callSound struct {
	ducked []bool
	chimes int
	stop   context.CancelFunc
	queued int
	played []int16
}

func (*callSound) Frames() (<-chan []int16, func())          { return nil, func() {} }
func (s *callSound) Play(pcm []int16)                        { s.played = slices.Clone(pcm) }
func (s *callSound) Queued() int                             { return s.queued }
func (*callSound) Claim(string, func(context.Context) error) {}
func (s *callSound) Duck(on bool)                            { s.ducked = append(s.ducked, on) }
func (s *callSound) Chime(schema.Stream, float64, ...tone.Note) {
	s.chimes++
	s.stop()
}

func TestCancelledRingingRestoresBackgroundAudio(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	sound := &callSound{stop: stop}
	if ring(ctx, sound) {
		t.Fatal("cancelled ringing reported an unanswered timeout")
	}
	if sound.chimes != 1 || !slices.Equal(sound.ducked, []bool{true, false}) {
		t.Fatalf("chimes = %d, ducking = %v", sound.chimes, sound.ducked)
	}
}

func TestStoppedCallsDoNotQueueAnotherTone(t *testing.T) {
	ctx, stop := context.WithCancel(context.Background())
	stop()
	sound := &callSound{stop: stop}
	ring(ctx, sound)
	ringback(ctx, sound)
	if sound.chimes != 0 {
		t.Fatalf("queued %d chimes after cancellation", sound.chimes)
	}
}

func TestConversationDropsAudioWhenHardwareQueueIsBehind(t *testing.T) {
	sound := &callSound{queued: maxQueued + 1}
	output := conversation{sound}
	pcm := []int16{100, -100}
	output.Play(pcm)
	if len(sound.played) != 0 {
		t.Fatal("queued more speech behind delayed audio")
	}
	sound.queued = maxQueued
	output.Play(pcm)
	if !slices.Equal(sound.played, pcm) {
		t.Fatal("did not resume after the queue drained")
	}
}
