//go:build linux

package hci

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"
)

func advert(address [6]byte, rssi int8) []byte {
	body := meta(report(0x00, 0x01, address, []byte{0x02, 0x01, 0x06}, rssi))

	out := []byte{typeEvent, eventLEMeta, byte(len(body))}
	return append(out, body...)
}

func classic(code byte, params ...byte) []byte {
	out := []byte{typeEvent, code, byte(len(params))}
	return append(out, params...)
}

func answerTo(opcode uint16, status byte) []byte {
	out := []byte{typeEvent, eventCommandComplete, 0x04, 0x01}
	out = binary.LittleEndian.AppendUint16(out, opcode)
	return append(out, status)
}

func reading(t *testing.T) (*Session, *os.File) {
	t.Helper()

	p, chip := paired(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	return Reader(ctx, p), chip
}

func TestAScanAndALinkBothGetTheirOwn(t *testing.T) {
	s, chip := reading(t)

	reports, stopScan := s.Reports()
	defer stopScan()

	events, stopLink := s.Events()
	defer stopLink()

	want := [6]byte{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	for _, b := range [][]byte{
		advert(want, -64),
		classic(0x05, 0x00, 0x05, 0x00, 0x13),
		advert(want, -70),
		aclOf(0x0005, boundaryStart, []byte{1, 2, 3, 4}),
	} {
		if _, err := chip.Write(b); err != nil {
			t.Fatal(err)
		}
	}

	for i := range 2 {
		g := within(t, reports)
		if g.kind != typeEvent || g.event.Code != eventLEMeta {
			t.Fatalf("advertisement %d came through as %#02x", i, g.event.Code)
		}
	}

	first := within(t, events)
	if first.kind != typeEvent || first.event.Code != 0x05 {
		t.Fatalf("the link was handed %#02x first, want the disconnection", first.event.Code)
	}

	second := within(t, events)
	if second.kind != typeACL {
		t.Fatalf("the link was handed an event second, want the data")
	}
	if !bytes.Equal(second.acl.Data, []byte{1, 2, 3, 4}) {
		t.Fatalf("the data came through as % x", second.acl.Data)
	}

	nothingMore(t, reports)
	nothingMore(t, events)
}

func TestAnAnswerIsFoundAmongTheAdvertisements(t *testing.T) {
	s, chip := reading(t)

	reports, stop := s.Reports()
	defer stop()

	noise := make(chan struct{})
	go func() {
		defer close(noise)
		for range 20 {
			chip.Write(advert([6]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}, -50))
			time.Sleep(time.Millisecond)
		}
	}()

	go func() {
		want, _ := command(hciReadAddress)
		got := make([]byte, len(want))

		chip.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := io.ReadFull(chip, got); err != nil {
			return
		}
		chip.SetReadDeadline(time.Time{})

		// Six bytes of address after the status, least significant first.
		out := []byte{typeEvent, eventCommandComplete, 0x0a, 0x01}
		out = binary.LittleEndian.AppendUint16(out, hciReadAddress)
		out = append(out, 0x00, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01)
		chip.Write(out)
	}()

	said, err := s.Command(hciReadAddress, nil...)
	if err != nil {
		t.Fatalf("asking for the address with a scan running: %v", err)
	}
	if want := []byte{0x06, 0x05, 0x04, 0x03, 0x02, 0x01}; !bytes.Equal(said, want) {
		t.Fatalf("the address came back as % x, want % x", said, want)
	}

	<-noise

	if len(reports) == 0 {
		t.Error("the scan saw nothing while the command was in flight")
	}
}

func TestARefusalIsAnAnswer(t *testing.T) {
	s, chip := reading(t)

	go func() {
		want, _ := command(hciWriteSimplePairing, 0x01)
		got := make([]byte, len(want))

		chip.SetReadDeadline(time.Now().Add(2 * time.Second))
		if _, err := io.ReadFull(chip, got); err != nil {
			return
		}
		chip.Write(answerTo(hciWriteSimplePairing, 0x12))
	}()

	if _, err := s.Command(hciWriteSimplePairing, 0x01); err == nil {
		t.Fatal("a refused command came back without an error")
	}
}

func TestTheSameCommandTwiceAtOnceIsRefused(t *testing.T) {
	s, _ := reading(t)

	cmd, err := command(hciReadAddress)
	if err != nil {
		t.Fatal(err)
	}

	started := make(chan struct{})
	go func() {
		close(started)
		s.Ask(cmd, 500*time.Millisecond)
	}()
	<-started

	time.Sleep(50 * time.Millisecond)

	if _, err := s.Ask(cmd, 100*time.Millisecond); err == nil {
		t.Fatal("the same command twice at once was accepted")
	}
}

func TestLettingGoStopsTheDelivery(t *testing.T) {
	s, chip := reading(t)

	reports, stop := s.Reports()
	stop()

	if _, ok := <-reports; ok {
		t.Fatal("a released taker is still open")
	}

	events, release := s.Events()
	defer release()

	if _, err := chip.Write(classic(0x05, 0x00, 0x05, 0x00, 0x13)); err != nil {
		t.Fatal(err)
	}
	if g := within(t, events); g.event.Code != 0x05 {
		t.Fatalf("the link was handed %#02x after a scan let go", g.event.Code)
	}
}

func within(t *testing.T, from <-chan packet) packet {
	t.Helper()

	select {
	case g, ok := <-from:
		if !ok {
			t.Fatal("the reader stopped")
		}
		return g
	case <-time.After(2 * time.Second):
		t.Fatal("nothing was delivered in 2s")
	}
	return packet{}
}

func nothingMore(t *testing.T, from <-chan packet) {
	t.Helper()

	select {
	case g := <-from:
		t.Fatalf("something else arrived: kind %#02x code %#02x", g.kind, g.event.Code)
	case <-time.After(100 * time.Millisecond):
	}
}
