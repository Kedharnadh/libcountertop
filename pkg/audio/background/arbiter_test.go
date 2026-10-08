package background

import (
	"sync"
	"testing"
)

type producer struct {
	mu       sync.Mutex
	down     bool
	ducked   bool
	requeues int
}

func (p *producer) Stand(down bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.down = down
}

func (p *producer) Duck(on bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ducked = on
}

func (p *producer) Requeue() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requeues++
}

func (p *producer) requeued() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.requeues
}

func (p *producer) held() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.down
}

func (p *producer) quiet() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ducked
}

func TestNewestProducerIsTheOneHeard(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}

	a.Took(group)
	a.Took(track)

	if !group.held() {
		t.Error("the group kept playing under the track that displaced it")
	}
	if track.held() {
		t.Error("the track that just started was stood down")
	}
	if a.Playing() != track {
		t.Error("the track is not the one being heard")
	}
}

func TestWhatWasInterruptedCarriesOn(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}

	a.Took(group)
	a.Took(track)
	a.Gave(track)

	if group.held() {
		t.Error("the group never came back: still standing down with nothing above it")
	}
	if a.Playing() != group {
		t.Error("the group is not the one being heard again")
	}
}

func TestFinishingUnderneathChangesNothing(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}

	a.Took(group)
	a.Took(track)
	a.Gave(group)

	if track.held() {
		t.Error("the track was stood down when something beneath it finished")
	}
	if a.Playing() != track {
		t.Error("the track stopped being the one heard")
	}
}

func TestTheHoldWinsOverAHandover(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}
	a.Took(group)

	a.Stand(true)
	if !group.held() {
		t.Fatal("the group ignored the speaker being taken")
	}

	a.Took(track)
	if !track.held() {
		t.Error("a track that started while the speaker was held began playing")
	}

	a.Stand(false)
	if track.held() {
		t.Error("the track never started: still standing down after the hold ended")
	}
}

func TestDuckingReachesTheOnesWaiting(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}
	a.Took(group)
	a.Took(track)

	a.Duck(true)
	if !group.quiet() || !track.quiet() {
		t.Error("ducking a turn missed a producer")
	}

	a.Duck(false)
	if group.quiet() || track.quiet() {
		t.Error("the turn ended and something stayed quiet")
	}
}

func TestOnlyTheProducerBeingHeardRescalesWhatIsQueued(t *testing.T) {
	a := &Arbiter{}
	group, track := &producer{}, &producer{}
	a.Took(group)
	a.Took(track)

	a.Duck(true)

	if got := track.requeued(); got != 1 {
		t.Errorf("the audible producer requeued %d times, want 1", got)
	}
	if got := group.requeued(); got != 0 {
		t.Errorf("a waiting producer requeued %d times, want 0", got)
	}
}

func TestNothingIsRescaledWhenTheTurnEnds(t *testing.T) {
	a := &Arbiter{}
	track := &producer{}
	a.Took(track)

	a.Duck(true)
	before := track.requeued()
	a.Duck(false)

	if got := track.requeued(); got != before {
		t.Errorf("unducking requeued: %d then %d", before, got)
	}
}

func TestAProducerStartingMidTurnArrivesQuiet(t *testing.T) {
	a := &Arbiter{}
	a.Duck(true)

	track := &producer{}
	a.Took(track)

	if !track.quiet() {
		t.Error("a track that started mid-turn came in at full volume")
	}
}

func TestStartingTwiceLeavesOneEntry(t *testing.T) {
	a := &Arbiter{}
	group := &producer{}

	a.Took(group)
	a.Took(group)
	a.Gave(group)

	if a.Playing() != nil {
		t.Error("a duplicate entry outlived the producer that finished")
	}
}

func TestAResumingTrackRetakingTheSpeakerIsNotHeldTwice(t *testing.T) {
	a := &Arbiter{}
	track := &producer{}
	a.Took(track)

	a.Stand(true)
	if !track.held() {
		t.Fatal("the track ignored the speaker being taken")
	}

	a.Took(track)
	if !track.held() {
		t.Error("a track must stay stood down while the speaker is held")
	}

	a.Stand(false)
	if track.held() {
		t.Error("the track never came back: still standing down after the hold ended")
	}
}

func TestARetakeAfterGivingBackDuringAHoldIsNotHeldTwice(t *testing.T) {
	a := &Arbiter{}
	track := &producer{}
	a.Took(track)

	a.Stand(true)
	if !track.held() {
		t.Fatal("the track ignored the speaker being taken")
	}

	a.Gave(track)
	a.Took(track)
	if !track.held() {
		t.Error("a track must stay stood down while the speaker is held")
	}

	a.Stand(false)
	if track.held() {
		t.Error("the track never came back: still standing down after the hold ended")
	}
}

func TestARetakeIsHeardRatherThanLeftStandingDown(t *testing.T) {
	a := &Arbiter{}
	stream, spin := &producer{}, &producer{}

	a.Took(stream)
	a.Took(spin)
	a.Took(stream)

	if stream.held() {
		t.Error(
			"the producer that took the speaker back is still standing down, so nothing is audible",
		)
	}
	if !spin.held() {
		t.Error("the producer it took over from is still playing")
	}
	if a.Playing() != stream {
		t.Error("the one being heard is not the one that took the speaker last")
	}
}

func TestTheQueueBelongsToWhateverIsBeingHeard(t *testing.T) {
	a := &Arbiter{}
	track, spin := &producer{}, &producer{}

	a.Took(track)
	if !a.Owns(track) {
		t.Error("the only producer does not own the queue it is filling")
	}

	a.Stand(true)
	if a.Owns(track) {
		t.Error("the queue is not the producer's while the speaker is held")
	}
	a.Stand(false)

	a.Took(spin)
	if a.Owns(track) {
		t.Error("the queue belongs to what took over, not to what stood down for it")
	}
	if !a.Owns(spin) {
		t.Error("the producer being heard does not own the queue")
	}

	a.Gave(spin)
	a.Gave(track)
	if !a.Owns(track) {
		t.Error("with nothing being heard there is no one else's audio to protect")
	}
}
