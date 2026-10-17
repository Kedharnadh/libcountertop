package sendspin

import (
	"slices"
	"testing"
)

func lefts(q *queue) []int16 {
	s := frames(q.frames())
	q.read(s)
	var out []int16
	for i := 0; i < len(s); i += Channels {
		out = append(out, s[i])
	}
	return out
}

func TestTheQueueWrapsWithoutLosingOrder(t *testing.T) {
	q := newQueue(4)
	q.put(0, ramp(3))
	q.drop(2)
	q.put(1, tone(2, 7))

	if got, want := lefts(&q), []int16{2, 7, 7}; !slices.Equal(got, want) {
		t.Errorf("queued %v, want %v", got, want)
	}
}

func TestLeadingSilenceGoesInFrontAcrossTheWrap(t *testing.T) {
	q := newQueue(4)
	q.put(0, tone(2, 5))
	if n := q.lead(1); n != 1 {
		t.Fatalf("lead inserted %d frames, want 1", n)
	}

	if got, want := lefts(&q), []int16{0, 5, 5}; !slices.Equal(got, want) {
		t.Errorf("queued %v, want %v", got, want)
	}
}

func TestRepeatCopiesTheFrontFrame(t *testing.T) {
	q := newQueue(4)
	q.put(0, ramp(2))
	q.drop(1)
	q.repeat()

	if got, want := lefts(&q), []int16{1, 1}; !slices.Equal(got, want) {
		t.Errorf("queued %v, want %v", got, want)
	}
}

func TestAGapPastTheEndIsSilenceNotOldAudio(t *testing.T) {
	q := newQueue(4)
	q.put(0, tone(4, 9))
	q.drop(4)
	q.put(2, tone(1, 3))

	if got, want := lefts(&q), []int16{0, 0, 3}; !slices.Equal(got, want) {
		t.Errorf("queued %v, want %v", got, want)
	}
}

func TestAFullQueueRefusesWhatDoesNotFit(t *testing.T) {
	q := newQueue(4)
	if n := q.put(2, tone(4, 1)); n != 2 {
		t.Errorf("stored %d frames, want 2", n)
	}
	if q.repeat() {
		t.Error("repeat fit into a full queue")
	}
	if n := q.lead(3); n != 0 {
		t.Errorf("lead inserted %d frames into a full queue", n)
	}
}
