package hci

import (
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	typeCommand = 0x01
	typeACL     = 0x02
	typeSCO     = 0x03
	typeEvent   = 0x04
)

const (
	scoHeader = 4
	scoKept   = 64
)

// In-band sleep bytes travel outside H4 framing.
const (
	ibsSleep   = 0xfe
	ibsWake    = 0xfd
	ibsWakeAck = 0xfc
)

const (
	eventCommandComplete = 0x0e
	eventCommandStatus   = 0x0f
	eventVendor          = 0xff
)

const (
	hciReset             = 0x0c03
	hciReadLocalVersion  = 0x1001
	hciReadLocalFeatures = 0x1003
)

// A chip in deep sleep loses the first wake byte waking its own UART.
const wakeAgain = 100 * time.Millisecond

// A command is at most 255 bytes of parameters.
const maxParams = 255

var (
	errShort = errors.New("ble: the packet is shorter than its header says")

	errQuiet = errors.New("ble: nothing came back")
)

// H4 type, opcode, length, parameters.
func command(opcode uint16, params ...byte) ([]byte, error) {
	if len(params) > maxParams {
		return nil, fmt.Errorf("ble: %d parameters, more than the %d a command carries",
			len(params), maxParams)
	}

	out := make([]byte, 4+len(params))
	out[0] = typeCommand
	binary.LittleEndian.PutUint16(out[1:], opcode)
	out[3] = byte(len(params))
	copy(out[4:], params)
	return out, nil
}

type event struct {
	Code   byte
	Params []byte
}

type packet struct {
	kind  byte
	event event
	acl   aclPacket
	sco   scoPacket
}

type scoPacket struct {
	Handle Handle
	Status byte
	Data   []byte
}

type pending struct {
	events []event
	acl    []aclPacket
	sco    []scoPacket
	scos   int
}

func (q *pending) add(p packet) {
	switch p.kind {
	case typeACL:
		q.acl = append(q.acl, p.acl)
	case typeSCO:
		q.scos++
		if len(q.sco) == scoKept {
			q.sco = append(q.sco[:0], q.sco[1:]...)
		}
		q.sco = append(q.sco, p.sco)
	default:
		q.events = append(q.events, p.event)
	}
}

func (q *pending) take() (event, bool) {
	if len(q.events) == 0 {
		return event{}, false
	}

	e := q.events[0]
	q.events = q.events[1:]
	return e, true
}

func (q *pending) takeACL() (aclPacket, bool) {
	if len(q.acl) == 0 {
		return aclPacket{}, false
	}

	a := q.acl[0]
	q.acl = q.acl[1:]
	return a, true
}

type stream struct {
	pending

	held []byte
}

func (s *stream) drain(skip func() bool) bool {
	var any bool

	for len(s.held) > 0 {
		if skip != nil && skip() {
			continue
		}

		got, took, err := parsePacket(s.held)
		if errors.Is(err, errShort) {
			break
		}
		if err != nil {
			s.held = resync(s.held)
			continue
		}

		s.held = s.held[took:]
		s.add(got)
		any = true
	}

	return any
}

func resync(buf []byte) []byte {
	for i, b := range buf {
		if b == typeEvent || b == typeACL || b == typeSCO {
			return buf[i:]
		}
	}
	return nil
}

func parsePacket(buf []byte) (p packet, took int, err error) {
	if len(buf) == 0 {
		return packet{}, 0, errShort
	}

	if buf[0] == typeACL {
		a, n, err := parseACL(buf)
		if err != nil {
			return packet{}, 0, err
		}
		return packet{kind: typeACL, acl: a}, n, nil
	}
	if buf[0] == typeSCO {
		if len(buf) < scoHeader {
			return packet{}, 0, errShort
		}
		n := int(buf[3])
		if len(buf) < scoHeader+n {
			return packet{}, 0, errShort
		}
		word := binary.LittleEndian.Uint16(buf[1:])
		return packet{kind: typeSCO, sco: scoPacket{
			Handle: Handle(word & handleMask),
			Status: byte(word >> 12 & 0x3),
			Data:   buf[scoHeader : scoHeader+n],
		}}, scoHeader + n, nil
	}

	// Type, code, length.
	const header = 3

	if len(buf) < header {
		return packet{}, 0, errShort
	}
	if buf[0] != typeEvent {
		return packet{}, 0, fmt.Errorf("ble: packet type %#02x is not an event, acl or sco", buf[0])
	}

	n := int(buf[2])
	if len(buf) < header+n {
		return packet{}, 0, errShort
	}
	return packet{kind: typeEvent, event: event{Code: buf[1], Params: buf[header : header+n]}},
		header + n, nil
}

func complete(e event, opcode uint16) ([]byte, error) {
	if e.Code != eventCommandComplete {
		return nil, fmt.Errorf("ble: event %#02x, want a command complete", e.Code)
	}
	if len(e.Params) < 4 {
		return nil, errShort
	}

	if got := binary.LittleEndian.Uint16(e.Params[1:]); got != opcode {
		return nil, fmt.Errorf("ble: a command complete for %#04x, waiting on %#04x", got, opcode)
	}
	if status := e.Params[3]; status != 0 {
		return nil, fmt.Errorf("ble: %#04x refused, status %#02x", opcode, status)
	}
	return e.Params[4:], nil
}
