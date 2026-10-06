package sendspin

import (
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"testing"
)

type fakeSink struct {
	attached Renderer
	written  uint64
}

func (s *fakeSink) Attach(r Renderer) { s.attached = r }
func (s *fakeSink) Written() uint64   { return s.written }
func TestBackendControlsRemainLiveAndVolumeIsClamped(t *testing.T) {
	depth := -20.0
	volume := 35
	muted := false
	o := NewOutput(&fakeSink{}, Controls{DuckDB: func() float64 { return depth }, Volume: func() int { return volume }, SetVolume: func(v int) { volume = v }, SetMuted: func(on bool) { muted = on }})
	o.Duck(true)
	if o.gain != ducking.Gain(depth) {
		t.Fatal(o.gain)
	}
	depth = -6
	o.Duck(true)
	if o.gain != ducking.Gain(depth) {
		t.Fatal("cached depth", o.gain)
	}
	o.setVolume(500)
	if volume != 100 || o.volume() != 100 {
		t.Fatal(volume, o.volume())
	}
	o.setMuted(true)
	if !muted {
		t.Fatal("mute did not reach the backend")
	}
	o.Duck(false)
	if o.gain != 1 {
		t.Fatal(o.gain)
	}
}
func TestNegotiatedOutputAttachesAndDetachesTheDeviceRenderer(t *testing.T) {
	sink := &fakeSink{}
	o := NewOutput(sink)
	if err := o.open(Rate, Channels, Bits); err != nil {
		t.Fatal(err)
	}
	if sink.attached != o {
		t.Fatal("renderer not attached")
	}
	o.close()
	if sink.attached != nil {
		t.Fatal("renderer remains attached")
	}
}
func TestOnlyOneServerCanHoldTheRoom(t *testing.T) {
	l := NewListener(nil, nil, func() Identity { return Identity{} }, Callbacks{})
	if !l.take() || l.take() {
		t.Fatal("concurrent server admitted")
	}
	l.give()
	if !l.take() {
		t.Fatal("room not released")
	}
}
