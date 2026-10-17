package volume

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type steps struct {
	step    int
	watched []func(int)
}

func (s *steps) Step() int { return s.step }

func (s *steps) Set(step int) {
	s.step = step
	for _, w := range s.watched {
		w(step)
	}
}

func (s *steps) WatchVolume(changed func(int)) func() {
	s.watched = append(s.watched, changed)
	return func() {}
}

func steppedVolume() (*Volume, *steps) {
	speaker := &steps{step: 25}
	v := NewVolume(Options{
		Mixer: MixerOptions{
			Output:  newFakeOutput(),
			Streams: []schema.Stream{schema.StreamMain, schema.StreamMedia},
		},
		Main:      Steps(speaker, 30),
		MainSteps: 30,
	})
	return v, speaker
}

func TestASuppliedMainHasItsOwnEntity(t *testing.T) {
	v, _ := steppedVolume()
	v.Restore(func(schema.Stream) int { return 60 })

	n := v.Number(schema.StreamMain)
	if n == nil || n.ObjectID != "volume_main" {
		t.Fatal("no volume_main entity")
	}
	if n.Get() != 83 {
		t.Errorf("main entity restored at %v, want step 25 of 30 as 83", n.Get())
	}
	if got := v.Streams(); len(got) != 2 || got[0] != schema.StreamMain {
		t.Errorf("streams %v, want main first", got)
	}
	if len(v.Entities()) != 2 {
		t.Errorf("%d entities, want main and media", len(v.Entities()))
	}
}

func TestAButtonStepMovesTheMainEntity(t *testing.T) {
	v, speaker := steppedVolume()
	var heard []Change
	v.Changed.Listen(func(c Change) { heard = append(heard, c) })

	v.Adjust(schema.StreamMain, -1)
	if speaker.step != 24 {
		t.Fatalf("one step down left the speaker at %d, want 24", speaker.step)
	}
	if got := v.Number(schema.StreamMain).Get(); got != 80 {
		t.Errorf("main entity reads %v, want 80", got)
	}
	if len(heard) != 1 || heard[0].Stream != schema.StreamMain || heard[0].Level != 80 {
		t.Errorf("heard %v, want one main change to 80", heard)
	}

	for range 30 {
		v.Adjust(schema.StreamMain, 1)
	}
	if speaker.step != 30 {
		t.Errorf("stepping up stopped at %d, want 30", speaker.step)
	}
}

func TestHomeAssistantSetsTheSuppliedMain(t *testing.T) {
	v, speaker := steppedVolume()
	v.Number(schema.StreamMain).OnCommand(50)
	if speaker.step != 15 {
		t.Errorf("main set to 50%% put the speaker at step %d, want 15", speaker.step)
	}
}

func TestWithoutASuppliedMainItIsAMixerStream(t *testing.T) {
	out := newFakeOutput()
	v := NewVolume(Options{Mixer: MixerOptions{
		Output:  out,
		Streams: []schema.Stream{schema.StreamMain, schema.StreamMedia},
	}})
	v.Set(schema.StreamMain, 40)
	v.Adjust(schema.StreamMain, 1)
	if l, _ := out.level(schema.StreamMain); l != 40+Step {
		t.Errorf("main reached the output at %d, want %d", l, 40+Step)
	}
	if n := v.Number(schema.StreamMain); n == nil || n.ObjectID != "volume_main" {
		t.Error("no volume_main entity")
	}
}
