//go:build linux

package hci

import (
	"io"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func paired(t *testing.T) (*Port, *os.File) {
	t.Helper()

	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("socketpair: %v", err)
	}

	// os.NewFile only polls a non-blocking descriptor.
	for _, fd := range fds {
		if err := unix.SetNonblock(fd, true); err != nil {
			t.Fatalf("nonblock: %v", err)
		}
	}

	mine := os.NewFile(uintptr(fds[0]), "port")
	theirs := os.NewFile(uintptr(fds[1]), "chip")
	t.Cleanup(func() { mine.Close(); theirs.Close() })

	return &Port{f: mine}, theirs
}

func silent(body int, mode byte) []byte {
	raw := blob(tlvPatch, body)
	raw[4+10] = mode
	return raw
}

func TestASilentDownloadWaitsForNothing(t *testing.T) {
	p, chip := paired(t)

	got := make(chan int, 1)
	go func() {
		n, _ := io.Copy(io.Discard, chip)
		got <- int(n)
	}()

	raw := silent(30764, saySilent)
	b, err := ReadBlob(raw)
	if err != nil {
		t.Fatal(err)
	}
	if b.Acked() {
		t.Fatal("download 0x03 came back as acknowledged")
	}

	done := make(chan error, 1)
	go func() { done <- Send(p, b) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("a silent download that nothing answered: %v", err)
		}
	case <-time.After(answers / 2):
		t.Fatal("a silent download waited for an answer that is never coming")
	}

	// Four bytes of H4 command and two of TLV request per segment.
	p.f.Close()

	const framing = 6
	sent, want := <-got, len(raw)+framing*len(b.Segments())
	if sent != want {
		t.Errorf("%d bytes reached the chip, want %d: %d segments carrying %d",
			sent, want, len(b.Segments()), len(raw))
	}
}

func TestAnAcknowledgedDownloadStillWaits(t *testing.T) {
	p, chip := paired(t)
	go io.Copy(io.Discard, chip)

	b, err := ReadBlob(silent(3000, sayEverything))
	if err != nil {
		t.Fatal(err)
	}
	if !b.Acked() {
		t.Fatal("download 0x00 came back as silent")
	}

	start := time.Now()
	err = Send(p, b)
	if err == nil {
		t.Fatal("a download nothing answered came back clean")
	}
	if waited := time.Since(start); waited < answers/2 {
		t.Errorf("gave up after %v without waiting for an answer: %v", waited, err)
	}
}
