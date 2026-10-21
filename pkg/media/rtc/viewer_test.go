package rtc

import (
	"context"
	"sync"
	"testing"
	"time"
)

type ear struct {
	mu    sync.Mutex
	heard int
}

func (e *ear) Play(pcm []int16) {
	e.mu.Lock()
	e.heard += len(pcm)
	e.mu.Unlock()
}

func (e *ear) samples() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.heard
}

func TestAViewerReceivesACamera(t *testing.T) {
	screen, speaker := &wall{}, &ear{}
	v, err := NewViewer(screen.Show, speaker.Play, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer v.Close()
	camera, err := New(Media{Audio: &tone{freq: 440}, Camera: &film{}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer camera.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	offer, err := v.Offer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := camera.Accept(ctx, offer)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Answered(answer); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) && len(screen.frames()) < 10 {
		time.Sleep(100 * time.Millisecond)
	}
	got := screen.frames()
	if len(got) < 10 {
		t.Fatalf("viewer showed %d frames", len(got))
	}
	if !got[0].Key {
		t.Fatal("first frame shown was not a keyframe")
	}
	if frames, keys := v.Received(); frames < 10 || keys == 0 {
		t.Fatalf("viewer counted %d frames, %d keys", frames, keys)
	}
	for time.Now().Before(deadline) && speaker.samples() < PlayRate/10*PlayChannels {
		time.Sleep(100 * time.Millisecond)
	}
	if n := speaker.samples(); n < PlayRate/10*PlayChannels {
		t.Fatalf("viewer played %d samples of the camera's audio, want at least 100 ms", n)
	}
}

func TestG711DecodesKnownCodes(t *testing.T) {
	cases := []struct {
		name   string
		decode func(byte) int16
		in     byte
		want   int16
	}{
		{"ulaw silence", ulaw, 0xff, 0},
		{"ulaw loudest negative", ulaw, 0x00, -32124},
		{"ulaw loudest positive", ulaw, 0x80, 32124},
		{"alaw smallest positive", alaw, 0xd5, 8},
		{"alaw smallest negative", alaw, 0x55, -8},
		{"alaw loudest positive", alaw, 0xaa, 32256},
	}
	for _, c := range cases {
		if got := c.decode(c.in); got != c.want {
			t.Errorf("%s: %#x decoded to %d, want %d", c.name, c.in, got, c.want)
		}
	}
}

func TestWidenFillsSixStereoSamplesPerInput(t *testing.T) {
	out := widen([]int16{600, 0}, 0)
	if len(out) != 2*widening*PlayChannels {
		t.Fatalf("widened to %d samples, want %d", len(out), 2*widening*PlayChannels)
	}
	if out[0] != 100 || out[1] != 100 || out[2*widening-1] != 600 || out[len(out)-1] != 0 {
		t.Fatalf("widened ramp %v", out)
	}
}
