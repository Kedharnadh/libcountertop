//go:build linux

package hci

import (
	"errors"
	"testing"
	"time"
)

func data(h Handle, body ...byte) []byte {
	out := []byte{typeACL, byte(h), 0x20, byte(len(body)), 0x00}
	return append(out, body...)
}

func TestAudioArrivingWhileWaitingForAnEventIsKept(t *testing.T) {
	p, w := wired(t)

	w.Write(data(1, 0x04, 0x00, 0x40, 0x00, 0xaa, 0xbb, 0xcc, 0xdd))
	w.Write([]byte{typeEvent, 0x0e, 0x01, 0x11})

	got, err := p.Read(soon)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Code != 0x0e {
		t.Errorf("event %#02x, want 0x0e", got.Code)
	}

	a, err := p.readACL(soon)
	if err != nil {
		t.Fatalf("the audio read past while waiting for an event was lost: %v", err)
	}
	if a.Handle != 1 {
		t.Errorf("handle %d, want 1", a.Handle)
	}
}

func TestAnEventArrivingWhileWaitingForAudioIsKept(t *testing.T) {
	p, w := wired(t)

	w.Write([]byte{typeEvent, 0x13, 0x01, 0x22})
	w.Write(data(2, 0x00, 0x00, 0x40, 0x00))

	if _, err := p.readACL(soon); err != nil {
		t.Fatalf("readACL: %v", err)
	}
	got, err := p.Read(soon)
	if err != nil {
		t.Fatalf("the event read past while waiting for audio was lost: %v", err)
	}
	if got.Code != 0x13 {
		t.Errorf("event %#02x, want 0x13", got.Code)
	}
}

func TestAudioKeepsItsOrder(t *testing.T) {
	p, w := wired(t)

	for h := range 3 {
		w.Write(data(Handle(h+1), 0x00, 0x00, 0x40, 0x00))
	}

	for want := range 3 {
		got, err := p.readACL(soon)
		if err != nil {
			t.Fatalf("readACL: %v", err)
		}
		if int(got.Handle) != want+1 {
			t.Errorf("handle %d, want %d", got.Handle, want+1)
		}
	}
}

func TestAQuietLineForAudio(t *testing.T) {
	p, w := wired(t)

	w.Write([]byte{typeEvent, 0x0e, 0x01, 0x33})

	if _, err := p.readACL(50 * time.Millisecond); !errors.Is(err, errQuiet) {
		t.Errorf("gave %v, want quiet", err)
	}

	if _, err := p.Read(soon); err != nil {
		t.Errorf("the event was not kept: %v", err)
	}
}

func TestAnACLPacketSplitAcrossReads(t *testing.T) {
	p, w := wired(t)

	w.Write([]byte{typeACL, 0x01, 0x20, 0x08, 0x00, 0x04, 0x00})
	go func() {
		time.Sleep(20 * time.Millisecond)
		w.Write([]byte{0x40, 0x00, 0xaa, 0xbb, 0xcc, 0xdd})
	}()

	got, err := p.readACL(soon)
	if err != nil {
		t.Fatalf("readACL: %v", err)
	}
	if got.Handle != 1 {
		t.Errorf("handle %d, want 1", got.Handle)
	}
}
