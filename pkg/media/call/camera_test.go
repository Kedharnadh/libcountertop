package call

import (
	"bytes"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/camera/helper"
	camerasession "github.com/ygelfand/libcountertop/pkg/camera/session"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type cameraTransport struct {
	frames chan helper.Frame
	done   chan struct{}
	once   sync.Once
	ready  sync.Once
	opened chan struct{}
}

func (s *cameraTransport) Next(time.Duration) (helper.Frame, error) {
	if s.opened != nil {
		s.ready.Do(func() { close(s.opened) })
	}
	select {
	case f := <-s.frames:
		return f, nil
	case <-s.done:
		return helper.Frame{}, net.ErrClosed
	}
}
func (*cameraTransport) AskStill() error        { return nil }
func (*cameraTransport) AskKey() error          { return nil }
func (*cameraTransport) SetParams(string) error { return nil }
func (s *cameraTransport) Close() error {
	s.once.Do(func() { close(s.done) })
	return nil
}

func TestRTCCameraWithoutResizeSubscriberSendsConfiguredKeyframes(t *testing.T) {
	transport := &cameraTransport{frames: make(chan helper.Frame, 2), done: make(chan struct{}), opened: make(chan struct{})}
	hub := camerasession.New(camerasession.Options{
		Snapshot: func() camerasession.Snapshot {
			return camerasession.Snapshot{Config: helper.Config{Width: 1280, Height: 720}, Key: "main"}
		},
		Open: func(helper.Config) (camerasession.Transport, error) { return transport, nil },
	})
	t.Cleanup(hub.Close)
	source := NewCamera(hub)
	// RTC does not subscribe to size changes; local preview does.
	frames, release := (camera{source: source}).Frames()
	t.Cleanup(release)
	select {
	case <-transport.opened:
	case <-time.After(time.Second):
		t.Fatal("camera did not join the shared session")
	}
	transport.frames <- helper.Frame{Config: true, Data: []byte{1, 2}}
	transport.frames <- helper.Frame{Key: true, Data: []byte{3, 4}, PTS: 100}
	select {
	case f, ok := <-frames:
		if !ok || !f.Key || f.PTS != 100 || !bytes.Equal(f.Data, []byte{1, 2, 3, 4}) {
			t.Fatalf("configured keyframe = %+v, open = %v", f, ok)
		}
	case <-time.After(time.Second):
		t.Fatal("camera did not send a frame without a resize callback")
	}
	release()
	if _, ok := <-frames; ok {
		t.Fatal("released camera still sends frames")
	}
}

func TestCallCameraSelectsAvailableStreams(t *testing.T) {
	main, sub := Size{Width: 1280, Height: 720}, Size{Width: 640, Height: 360}
	for _, tc := range []struct {
		name   string
		sizes  []Size
		stream schema.CallStream
		at     int
		size   Size
		ok     bool
	}{
		{name: "no camera", stream: schema.CallMain},
		{name: "main", sizes: []Size{main, sub}, stream: schema.CallMain, size: main, ok: true},
		{name: "sub", sizes: []Size{main, sub}, stream: schema.CallSub, at: 1, size: sub, ok: true},
		{name: "sub unavailable", sizes: []Size{main}, stream: schema.CallSub, size: main, ok: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hub := camerasession.New(camerasession.Options{Snapshot: func() camerasession.Snapshot {
				var cfg helper.Config
				if len(tc.sizes) > 0 {
					cfg.Width, cfg.Height = tc.sizes[0].Width, tc.sizes[0].Height
				}
				if len(tc.sizes) > 1 {
					cfg.SubWidth, cfg.SubHeight = tc.sizes[1].Width, tc.sizes[1].Height
				}
				return camerasession.Snapshot{Config: cfg}
			}})
			t.Cleanup(hub.Close)
			c := NewCamera(hub)
			at, size, ok := c.Select(tc.stream)
			if at != tc.at || size != tc.size || ok != tc.ok {
				t.Fatalf("Select = %d, %+v, %v", at, size, ok)
			}
		})
	}
}
