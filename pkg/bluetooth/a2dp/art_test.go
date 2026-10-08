package a2dp

import (
	"testing"
	"time"
)

func square() []byte { return []byte{0xff, 0xd8, 0xff, 0xd9} }

func held(s *Sink, handle string) bool {
	raw, _ := s.image(handle)
	return raw != nil
}

func connected(t *testing.T) *Sink {
	t.Helper()
	s := testSink()
	t.Cleanup(s.art.clear)

	s.art.clear()
	s.art.mu.Lock()
	s.art.connection, s.art.open = 1, true
	s.art.mu.Unlock()

	return s
}

func waiting(s *Sink) (busy string, want []string) {
	s.art.mu.Lock()
	defer s.art.mu.Unlock()
	return s.art.asked, append([]string(nil), s.art.want...)
}

func TestOneFetchAtATime(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.wants("222", false)

	busy, want := waiting(s)
	if busy != "111" {
		t.Fatalf("fetching %q, want the first", busy)
	}
	if len(want) != 2 {
		t.Fatalf("%d waiting, want both until one finishes", len(want))
	}

	s.fetched("111", square())

	busy, want = waiting(s)
	if busy != "222" {
		t.Errorf("fetching %q after the first finished, want the second", busy)
	}
	if len(want) != 1 || want[0] != "222" {
		t.Errorf("waiting %v, want only the second", want)
	}
}

func TestTheOneBeingPlayedGoesNext(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.wants("222", false)
	s.wants("333", true)

	if busy, _ := waiting(s); busy != "111" {
		t.Fatalf("fetching %q, want the one already in flight", busy)
	}

	s.fetched("111", square())

	if busy, _ := waiting(s); busy != "333" {
		t.Errorf("fetching %q next, want the one being played", busy)
	}
}

func TestNothingIsFetchedTwice(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.wants("111", false)
	s.wants("222", false)

	if _, want := waiting(s); len(want) != 2 {
		t.Fatalf("waiting %v, want one of each", want)
	}

	s.fetched("111", square())
	s.fetched("222", square())

	s.wants("111", false)
	if busy, want := waiting(s); busy != "" || len(want) != 0 {
		t.Errorf("a handle already held was asked for again: %q %v", busy, want)
	}
}

func TestARefusedImageDoesNotWedgeTheQueue(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.wants("222", false)
	s.fetched("111", nil)

	if busy, _ := waiting(s); busy != "222" {
		t.Errorf("fetching %q after a refusal, want the next", busy)
	}
	if held(s, "111") {
		t.Error("a refused handle has an image")
	}
}

func TestAFetchThatIsNeverAnsweredLetsGo(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.wants("222", false)

	if busy, _ := waiting(s); busy != "111" {
		t.Fatalf("fetching %q, want the first", busy)
	}

	s.art.mu.Lock()
	s.art.began = time.Now().Add(-waits - time.Second)
	s.art.mu.Unlock()

	s.pump()

	if busy, _ := waiting(s); busy != "222" {
		t.Errorf("fetching %q, want the next after the first went unanswered", busy)
	}
}

func TestWhatIsNotWantedIsDropped(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.fetched("111", square())

	if !held(s, "111") {
		t.Fatal("a fetched image was not kept")
	}

	s.art.mu.Lock()
	s.art.have["111"].used = time.Now().Add(-stays - time.Minute)
	s.art.mu.Unlock()

	s.expire()

	if held(s, "111") {
		t.Error("an image nothing has wanted for a long time is still held")
	}
}

func TestWantingOneAgainKeepsIt(t *testing.T) {
	s := connected(t)

	s.wants("111", false)
	s.fetched("111", square())

	s.art.mu.Lock()
	s.art.have["111"].used = time.Now().Add(-stays - time.Minute)
	s.art.mu.Unlock()

	s.wants("111", false)
	s.expire()

	if !held(s, "111") {
		t.Error("an image wanted again was dropped anyway")
	}
}
