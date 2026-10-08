package session

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
)

type fakeTransport struct {
	frames chan helper.Frame
	closed chan struct{}
	once   sync.Once
	asks   chan struct{}
}

func fakeStream() *fakeTransport {
	return &fakeTransport{
		frames: make(chan helper.Frame, 128),
		closed: make(chan struct{}),
		asks:   make(chan struct{}, 16),
	}
}
func (f *fakeTransport) Next(time.Duration) (helper.Frame, error) {
	select {
	case frame := <-f.frames:
		return frame, nil
	case <-f.closed:
		return helper.Frame{}, net.ErrClosed
	}
}
func (f *fakeTransport) AskStill() error        { f.asks <- struct{}{}; return nil }
func (f *fakeTransport) AskKey() error          { return nil }
func (f *fakeTransport) SetParams(string) error { return nil }

func (f *fakeTransport) Close() error { f.once.Do(func() { close(f.closed) }); return nil }
func newFakeHub(t *testing.T, linger time.Duration) (*Hub, **fakeTransport, *atomic.Int32) {
	t.Helper()
	var current *fakeTransport
	var opens atomic.Int32
	h := New(Options{Snapshot: func() Snapshot {
		return Snapshot{Config: helper.Config{Width: 640, Height: 480, SubWidth: 320}, Key: "one"}
	}, Open: func(helper.Config) (Transport, error) { current = fakeStream(); opens.Add(1); return current, nil }, Linger: linger, Poll: time.Hour})
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
	(*current).frames <- helper.Frame{Still: true, Data: []byte{1, 2, 3, 4}, Width: 2, Height: 1}
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
			Snapshot: func() Snapshot { return Snapshot{Config: helper.Config{Width: 640}, Key: fmt.Sprint(shape.Load())} },
			Muted:    muted.Load,
			Open:     func(helper.Config) (Transport, error) { f := fakeStream(); streams = append(streams, f); return f, nil },
			Poll:     time.Hour,
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
	(*current).frames <- helper.Frame{Data: []byte{7}, Config: true}
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
		(*current).frames <- helper.Frame{Data: []byte{9}}
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
