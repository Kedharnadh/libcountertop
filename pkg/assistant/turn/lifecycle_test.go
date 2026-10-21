package turn

import (
	"context"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/hook"
)

func TestOldReplyCannotEndOrDisarmNewReply(t *testing.T) {
	c := &Conversation{phase: phaseReplying, replyID: 2, deadline: time.NewTimer(time.Hour)}
	defer c.deadline.Stop()
	c.handle(event{kind: evPlayed, replyID: 1})
	c.handle(event{kind: evPlaying, replyID: 1})
	if c.phase != phaseReplying || c.deadline == nil {
		t.Fatal("old reply changed the current reply")
	}
	c.handle(event{kind: evPlaying, replyID: 2})
	if c.deadline != nil {
		t.Fatal("current audio did not disarm its deadline")
	}
}

func TestListeningHoldShowsTheMicStageOnlyWhileIdle(t *testing.T) {
	var shown []Showing
	h := &hook.Hook[Showing]{}
	h.Listen(func(s Showing) { shown = append(shown, s) })
	c := &Conversation{slot: 1, options: Dependencies{Presentation: &Presentation{Shown: h}}}

	c.handle(event{kind: evListen})
	c.handle(event{kind: evRelease})
	if len(shown) != 2 || shown[0].Phase != Listening || shown[0].Slot != 1 ||
		shown[1].Phase != Idle {
		t.Fatalf("idle hold showed %+v, want listening on slot 1 then idle", shown)
	}

	shown = nil
	c.phase = phaseThinking
	c.handle(event{kind: evListen})
	c.handle(event{kind: evRelease})
	if len(shown) != 0 {
		t.Fatalf("a hold during a turn showed %+v, want nothing", shown)
	}
}

type replySink struct{ samples []int16 }

func (s *replySink) PlayVoice(samples []int16) { s.samples = append(s.samples, samples...) }
func (*replySink) Queued() int                 { return 0 }

func TestReplyOwnsChunksAndClosesIdempotently(t *testing.T) {
	s := NewReply(0, 0, time.Second)
	data := []byte{1, 0, 2, 0}
	s.Send(data)
	data[0] = 99
	s.Done()
	s.Done()
	s.Send([]byte{3, 0})
	out := &replySink{}
	if err := s.Play(context.Background(), out, func() {}); err != nil {
		t.Fatal(err)
	}
	if len(out.samples) != 2 || out.samples[0] != 1 || out.samples[1] != 2 {
		t.Fatalf("samples=%v", out.samples)
	}
}
