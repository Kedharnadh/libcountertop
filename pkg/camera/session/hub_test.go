package session

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/camerafeed"
)

type fakeTransport struct {
	frames chan camerafeed.Frame
	closed chan struct{}
	once   sync.Once
	asks   chan struct{}
	wants  chan [2]bool
}

func fakeStream() *fakeTransport {
	return &fakeTransport{
		frames: make(chan camerafeed.Frame, 128),
		closed: make(chan struct{}),
		asks:   make(chan struct{}, 16),
		wants:  make(chan [2]bool, 16),
	}
}
func (f *fakeTransport) Next(time.Duration) (camerafeed.Frame, error) {
	select {
	case frame := <-f.frames:
		return frame, nil
	case <-f.closed:
		return camerafeed.Frame{}, net.ErrClosed
	}
}
func (f *fakeTransport) AskStill() error { f.asks <- struct{}{}; return nil }
func (f *fakeTransport) AskKey() error   { return nil }
func (f *fakeTransport) Want(main, sub bool) error {
	f.wants <- [2]bool{main, sub}
	return nil
}
func (f *fakeTransport) SetParams(string) error { return nil }

func (f *fakeTransport) Close() error { f.once.Do(func() { close(f.closed) }); return nil }
func newFakeHub(t *testing.T, linger time.Duration) (*Hub, **fakeTransport, *atomic.Int32) {
	t.Helper()
	var current *fakeTransport
	var opens atomic.Int32
	h := New(Options{Snapshot: func() Snapshot {
		return Snapshot{
			Config: camerafeed.Config{Width: 640, Height: 480, SubWidth: 320},
			Key:    "one",
		}
	}, Open: func(camerafeed.Config) (Transport, error) { current = fakeStream(); opens.Add(1); return current, nil }, Linger: linger, Poll: time.Hour})
	t.Cleanup(h.Close)
	return h, &current, &opens
}
func TestRefreshReusesTheCameraAndDuplicateLeaveCannotCloseAnotherSubscriber(t *testing.T) {
	h, current, opens := newFakeHub(t, 15*time.Millisecond)
	s, first, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	h.Leave(s, first)
	h.Leave(s, first)
	select {
	case <-(*current).closed:
		t.Fatal("duplicate Leave closed another user's session")
	case <-time.After(30 * time.Millisecond):
	}
	h.Leave(s, second)
	reused, ch, err := h.Join(0)
	if err != nil || reused != s || opens.Load() != 1 {
		t.Fatal("refresh reopened the camera", err, opens.Load())
	}
	h.Leave(reused, ch)
	select {
	case <-(*current).closed:
	case <-time.After(time.Second):
		t.Fatal("idle camera never closed")
	}
}
func TestStillTimeoutIsRemovedAndTheNextRequestCanComplete(t *testing.T) {
	h, current, _ := newFakeHub(t, time.Second)
	if _, err := h.Still(5 * time.Millisecond); err == nil {
		t.Fatal("expected timeout")
	}
	<-(*current).asks
	h.mu.Lock()
	waiting := len(h.cur.stills)
	h.mu.Unlock()
	if waiting != 0 {
		t.Fatal("timed-out request retained", waiting)
	}
	done := make(chan error, 1)
	go func() {
		p, err := h.Still(time.Second)
		if err == nil && (p.Width != 2 || len(p.RGBA) != 4) {
			err = fmt.Errorf("bad picture: %+v", p)
		}
		done <- err
	}()
	select {
	case <-(*current).asks:
	case <-time.After(time.Second):
		t.Fatal("new still was suppressed")
	}
	(*current).frames <- camerafeed.Frame{Still: true, Data: []byte{1, 2, 3, 4}, Width: 2, Height: 1}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
func TestConfigurationReplacementAndMuteRetireTheCorrectSession(t *testing.T) {
	var shape atomic.Int32
	var muted atomic.Bool
	var streams []*fakeTransport
	h := New(
		Options{
			Snapshot: func() Snapshot { return Snapshot{Config: camerafeed.Config{Width: 640}, Key: fmt.Sprint(shape.Load())} },
			Blocked:  func() Block { return blockedIf(muted.Load()) },
			Open: func(camerafeed.Config) (Transport, error) {
				f := fakeStream()
				streams = append(streams, f)
				return f, nil
			},
			Poll: time.Hour,
		},
	)
	defer h.Close()
	first, _, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	shape.Store(1)
	second, _, err := h.Join(0)
	if err != nil || first == second {
		t.Fatal("geometry did not replace the session", err)
	}
	if !errors.Is(first.Err(), ErrRestarted) {
		t.Fatal(first.Err())
	}
	muted.Store(true)
	if _, _, err := h.Join(0); !errors.Is(err, ErrMuted) {
		t.Fatal(err)
	}
	if !errors.Is(second.Err(), ErrMuted) {
		t.Fatal(second.Err())
	}
	for _, stream := range streams {
		select {
		case <-stream.closed:
		default:
			t.Fatal("retired camera left open")
		}
	}
}
func TestSubscribersReceiveCachedHeadersAndOneSlowSubscriberDoesNotBlockAnother(t *testing.T) {
	h, current, _ := newFakeHub(t, time.Second)
	s, first, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	(*current).frames <- camerafeed.Frame{Data: []byte{7}, Config: true}
	select {
	case <-first:
	case <-time.After(time.Second):
		t.Fatal("no header")
	}
	_, second, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case frame := <-second:
		if !frame.Config || frame.Data[0] != 7 {
			t.Fatal(frame)
		}
	case <-time.After(time.Second):
		t.Fatal("cached header not delivered")
	}
	for range 100 {
		(*current).frames <- camerafeed.Frame{Data: []byte{9}}
	}
	select {
	case frame := <-second:
		if frame.Data[0] != 9 {
			t.Fatal(frame)
		}
	case <-time.After(time.Second):
		t.Fatal("slow subscriber blocked fanout")
	}
	h.Leave(s, first)
	h.Leave(s, second)
	if _, _, err := h.Join(3); err == nil {
		t.Fatal("invalid stream accepted")
	}
}

func TestKeepOpensWithoutUsersAndTellsTheHelperWhatIsWatched(t *testing.T) {
	opened := make(chan *fakeTransport, 4)
	h := New(Options{
		Snapshot: func() Snapshot {
			return Snapshot{
				Config: camerafeed.Config{Width: 640, Height: 480, SubWidth: 320},
				Key:    "one",
			}
		},
		Open: func(camerafeed.Config) (Transport, error) {
			f := fakeStream()
			opened <- f
			return f, nil
		},
		Linger: time.Millisecond,
		Poll:   5 * time.Millisecond,
		Keep:   true,
	})
	t.Cleanup(h.Close)
	h.Start()

	var f *fakeTransport
	select {
	case f = <-opened:
	case <-time.After(time.Second):
		t.Fatal("the camera was not opened without a user")
	}
	want := func(main, sub bool) {
		t.Helper()
		select {
		case got := <-f.wants:
			if got != [2]bool{main, sub} {
				t.Fatalf("told the helper %v, want main %v sub %v", got, main, sub)
			}
		case <-time.After(time.Second):
			t.Fatalf("the helper was not told main %v sub %v", main, sub)
		}
	}
	want(false, false)

	s, main, err := h.Join(0)
	if err != nil {
		t.Fatal(err)
	}
	want(true, false)
	_, sub, err := h.Join(1)
	if err != nil {
		t.Fatal(err)
	}
	want(true, true)
	h.Leave(s, main)
	want(false, true)
	h.Leave(s, sub)
	want(false, false)

	select {
	case <-f.closed:
		t.Fatal("the camera closed after its last user left")
	case <-opened:
		t.Fatal("the camera was opened twice")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestKeepReopensAfterTheHelperGoesAway(t *testing.T) {
	opened := make(chan *fakeTransport, 4)
	h := New(Options{
		Snapshot: func() Snapshot { return Snapshot{Config: camerafeed.Config{Width: 640, Height: 480}, Key: "one"} },
		Open: func(camerafeed.Config) (Transport, error) {
			f := fakeStream()
			opened <- f
			return f, nil
		},
		Poll: 5 * time.Millisecond,
		Keep: true,
	})
	t.Cleanup(h.Close)
	h.Start()
	first := <-opened
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("the camera was not reopened")
	}
}

func blockedIf(muted bool) Block {
	if muted {
		return Muted
	}
	return Open
}

func TestBlockedStreamsGetTheirCardAndUnblockingEndsIt(t *testing.T) {
	var block atomic.Int32
	block.Store(int32(Muted))
	opened := make(chan camerafeed.Config, 4)
	drawn := make(chan Block, 8)
	h := New(Options{
		Snapshot: func() Snapshot {
			return Snapshot{
				Config: camerafeed.Config{Width: 4, Height: 2, SubWidth: 2, SubHeight: 2},
				Key:    "one",
			}
		},
		Open: func(c camerafeed.Config) (Transport, error) {
			opened <- c
			return fakeStream(), nil
		},
		Blocked: func() Block { return Block(block.Load()) },
		Card: func(w, h int, b Block) []byte {
			drawn <- b
			return make([]byte, w*h)
		},
		Poll: 5 * time.Millisecond,
	})
	t.Cleanup(h.Close)

	if _, _, err := h.Join(0); !errors.Is(err, ErrMuted) {
		t.Fatalf("Join while muted = %v, want ErrMuted", err)
	}
	s, _, err := h.JoinOrCard(0)
	if err != nil {
		t.Fatal(err)
	}
	c := <-opened
	if len(c.Card[0]) != 8 || len(c.Card[1]) != 4 {
		t.Fatalf(
			"opened with card planes of %d and %d bytes, want 8 and 4",
			len(c.Card[0]),
			len(c.Card[1]),
		)
	}
	if b := <-drawn; b != Muted {
		t.Fatalf("drew the card for %v, want Muted", b)
	}
	<-drawn

	block.Store(int32(Shuttered))
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("the mute card kept running after the shutter closed instead")
	}
	if _, _, err := h.Join(0); !errors.Is(err, ErrShutter) {
		t.Fatalf("Join with the shutter closed = %v, want ErrShutter", err)
	}
	s, _, err = h.JoinOrCard(0)
	if err != nil {
		t.Fatal(err)
	}
	<-opened
	if b := <-drawn; b != Shuttered {
		t.Fatalf("drew the card for %v, want Shuttered", b)
	}

	block.Store(int32(Open))
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("the card kept running after the camera was unblocked")
	}
	if err := s.Err(); !errors.Is(err, ErrRestarted) {
		t.Errorf("the card ended with %v, want ErrRestarted", err)
	}
	if _, _, err := h.JoinOrCard(0); err != nil {
		t.Fatal(err)
	}
	if c := <-opened; c.Card[0] != nil {
		t.Error("after unmuting, JoinOrCard opened another card instead of the camera")
	}
}
