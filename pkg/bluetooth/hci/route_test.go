//go:build linux

package hci

import (
	"errors"
	"os"
	"testing"
	"time"
)

func wired(t *testing.T) (*Port, *os.File) {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })

	return &Port{f: r}, w
}

const soon = 2 * time.Second

func TestAPacketSplitAcrossReads(t *testing.T) {
	p, w := wired(t)

	_, _ = w.Write([]byte{typeEvent, 0x0e, 0x02})
	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = w.Write([]byte{0xbb, 0xcc})
	}()

	got, err := p.Read(soon)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Code != 0x0e || string(got.Params) != "\xbb\xcc" {
		t.Errorf("came back %#02x % x", got.Code, got.Params)
	}
}

func TestQueuedEventsKeepTheirOrder(t *testing.T) {
	p, w := wired(t)

	_, _ = w.Write([]byte{
		typeEvent, 0x10, 0x00,
		typeEvent, 0x11, 0x00,
		typeEvent, 0x12, 0x00,
	})

	for _, want := range []byte{0x10, 0x11, 0x12} {
		got, err := p.Read(soon)
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if got.Code != want {
			t.Errorf("came back %#02x, want %#02x", got.Code, want)
		}
	}
}

func TestAQuietLine(t *testing.T) {
	p, _ := wired(t)

	if _, err := p.Read(50 * time.Millisecond); !errors.Is(err, errQuiet) {
		t.Errorf("gave %v, want quiet", err)
	}
}

func TestReadingPastRubbishOnTheLine(t *testing.T) {
	p, w := wired(t)

	_, _ = w.Write([]byte{0xaa, 0xbb, typeEvent, 0x01, 0x01, 0xcc})

	got, err := p.Read(soon)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Code != 0x01 || string(got.Params) != "\xcc" {
		t.Errorf("came back %#02x % x", got.Code, got.Params)
	}
}
