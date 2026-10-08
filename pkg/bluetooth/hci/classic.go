package hci

import (
	"fmt"
	"log/slog"
)

// OGF 3: controller and baseband.
const (
	hciWriteLocalName     = 0x0c13
	hciWriteScanEnable    = 0x0c1a
	hciWriteClassOfDevice = 0x0c24
	hciWriteSimplePairing = 0x0c56
	hciWriteEventMask     = 0x0c01
)

const (
	scanOff       = 0x00
	scanInquiry   = 0x01
	scanPage      = 0x02
	scanDiscovery = scanInquiry | scanPage
)

// Audio/video, loudspeaker, audio and rendering services; little endian.
var SpeakerClass = [3]byte{0x14, 0x04, 0x24}

func discoverable(s *Session, name string, on bool) error {
	if !on {
		_, err := s.Command(hciWriteScanEnable, scanOff)
		return err
	}

	if _, err := s.Command(hciWriteEventMask, eventMask[:]...); err != nil {
		slog.Warn("the controller would not take an event mask, so pairing may not complete",
			"err", err)
	}

	if _, err := s.Command(hciWriteSimplePairing, 0x01); err != nil {
		return fmt.Errorf("ble: simple pairing: %w", err)
	}

	if _, err := s.Command(hciWriteClassOfDevice, SpeakerClass[:]...); err != nil {
		return fmt.Errorf("ble: the class of device: %w", err)
	}
	if _, err := s.Command(hciWriteLocalName, localName(name)...); err != nil {
		return fmt.Errorf("ble: the name: %w", err)
	}

	_, err := s.Command(hciWriteScanEnable, scanDiscovery)
	return err
}

// The top two bits are reserved.
var eventMask = [8]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x3f}

const nameBytes = 248

// 248 bytes of UTF-8, zero padded.
func localName(name string) []byte {
	out := make([]byte, nameBytes)

	if len(name) <= nameBytes {
		copy(out, name)
		return out
	}

	cut := 0
	for i := range name {
		if i > nameBytes {
			break
		}
		cut = i
	}
	copy(out, name[:cut])
	return out
}

func (r *Radio) Buffers() (size, total int, err error) {
	s, err := r.Line()
	if err != nil {
		return 0, 0, err
	}

	said, err := s.Command(hciReadBufferSize)
	if err != nil {
		return 0, 0, err
	}
	return parseBufferSize(said)
}

const hciReadAddress = 0x1009

func (r *Radio) Address() ([6]byte, error) {
	var none [6]byte

	s, err := r.Line()
	if err != nil {
		return none, err
	}

	said, err := s.Command(hciReadAddress)
	if err != nil {
		return none, err
	}
	if len(said) < 6 {
		return none, fmt.Errorf("ble: an address of %d bytes", len(said))
	}

	// HCI addresses are little endian.
	copy(none[:], said[:6])
	for i, j := 0, 5; i < j; i, j = i+1, j-1 {
		none[i], none[j] = none[j], none[i]
	}
	return none, nil
}
