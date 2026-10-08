package touch

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"sync"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/geometry"
)

const (
	panelW = 1200
	panelH = 1920
)

type fakePanel struct {
	mu   sync.Mutex
	w, h int
	rot  geometry.Orientation
}

func (p *fakePanel) Native() (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.w, p.h
}

func (p *fakePanel) Orientation() geometry.Orientation {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rot
}

func (p *fakePanel) turn(rot geometry.Orientation) {
	p.mu.Lock()
	p.rot = rot
	p.mu.Unlock()
}

func landscape() *fakePanel {
	return &fakePanel{w: panelW, h: panelH, rot: geometry.Rotate90}
}

func TestRecognizerFollowsTheRotation(t *testing.T) {
	p := landscape()
	s := New("", p)

	s.recognize(down(1, 10, 10))

	s.mu.Lock()
	first := s.rec
	s.mu.Unlock()

	if first == nil {
		t.Fatal("no recognizer was built")
	}

	p.turn(geometry.Rotate180)
	s.recognize(down(2, 10, 10))

	s.mu.Lock()
	second := s.rec
	s.mu.Unlock()

	if second == first {
		t.Error("the recognizer survived a rotation")
	}
}

func TestRecognizerIsSizedInViewedCoordinates(t *testing.T) {
	p := landscape()
	s := New("", p)
	s.recognize(down(1, 10, 10))

	s.mu.Lock()
	rec := s.rec
	s.mu.Unlock()

	w, h := p.rot.Size(panelW, panelH)
	if rec.W != w || rec.H != h {
		t.Errorf("the recognizer is %dx%d, want %dx%d at %v", rec.W, rec.H, w, h, p.rot)
	}
}

func TestPlace(t *testing.T) {
	s := New("", landscape())

	tests := []struct {
		name         string
		px, py       int
		wantX, wantY int
	}{
		{"panel origin is the bottom left of the picture", 0, 0, panelH - 1, 0},
		{"the far corner of the panel is the top right", panelW - 1, panelH - 1, 0, panelW - 1},
		{"the middle stays in the middle", 600, 960, 959, 600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := s.place(tt.px, tt.py)
			if x != tt.wantX || y != tt.wantY {
				t.Errorf("place(%d, %d) = %d,%d, want %d,%d",
					tt.px, tt.py, x, y, tt.wantX, tt.wantY)
			}
		})
	}
}

func TestPlaceStaysOnScreen(t *testing.T) {
	s := New("", landscape())

	corners := [][2]int{{0, 0}, {panelW - 1, 0}, {0, panelH - 1}, {panelW - 1, panelH - 1}}

	for _, c := range corners {
		x, y := s.place(c[0], c[1])
		if x < 0 || x >= panelH {
			t.Errorf("place(%d, %d) x = %d, outside 0..%d", c[0], c[1], x, panelH-1)
		}
		if y < 0 || y >= panelW {
			t.Errorf("place(%d, %d) y = %d, outside 0..%d", c[0], c[1], y, panelW-1)
		}
	}
}

func TestWithNoPanelTouchesStayWhereTheHardwarePutThem(t *testing.T) {
	s := New("", &fakePanel{rot: geometry.Rotate90})

	if x, y := s.place(3, 400); x != 3 || y != 400 {
		t.Errorf("place(3, 400) = %d,%d with no panel, want 3,400", x, y)
	}
}

func TestFeedReachesBothHooks(t *testing.T) {
	s := New("", landscape())

	var contacts []Contact
	var gestures []Gesture
	s.Contacts.Listen(func(c Contact) { contacts = append(contacts, c) })
	s.Gestures.Listen(func(g Gesture) { gestures = append(gestures, g) })

	s.Feed(down(1, 900, 600))
	s.Feed(up(1, 903, 604))

	if len(contacts) != 2 {
		t.Errorf("got %d contacts, want 2", len(contacts))
	}
	if len(gestures) != 1 || gestures[0].Kind != Tap {
		t.Errorf("got %+v, want one tap", gestures)
	}
}

func TestRunWithNothingOpenWaitsForTheContext(t *testing.T) {
	s := New("", landscape())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Run(ctx); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
}

func TestRunDeliversUntilTheContextEnds(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	s := New("", landscape())
	s.f = r

	got := make(chan Contact, 4)
	s.Contacts.Listen(func(c Contact) { got <- c })

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.Run(ctx) }()

	var b bytes.Buffer
	for _, e := range []rawEvent{
		ev(evAbs, absMTTrackingID, 7),
		ev(evAbs, absMTPositionX, 600),
		ev(evAbs, absMTPositionY, 960),
		syn(),
	} {
		if err := binary.Write(&b, binary.LittleEndian, e); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.Write(b.Bytes()); err != nil {
		t.Fatal(err)
	}

	if c := <-got; c.Phase != Down || c.X != 959 || c.Y != 600 {
		t.Errorf("got %v, want a down at 959,600", c)
	}

	cancel()
	if err := <-done; err != nil {
		t.Errorf("Run = %v after the context ended, want nil", err)
	}
}
