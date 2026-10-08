package avrcp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrShort = errors.New("avrcp: shorter than its header says")

const PID = 0x110e

const (
	MessageCommand  = 0
	MessageResponse = 1
)

const (
	PacketSingle   = 0
	PacketStart    = 1
	PacketContinue = 2
	PacketEnd      = 3
)

const avctpHeader = 3

type Transport struct {
	// Four bits; wraps at sixteen.
	Label byte

	Type byte

	Invalid bool

	Payload []byte
}

func ParseTransport(buf []byte) (Transport, error) {
	if len(buf) < avctpHeader {
		return Transport{}, ErrShort
	}

	if packet := buf[0] >> 2 & 0x3; packet != PacketSingle {
		return Transport{}, fmt.Errorf("avrcp: packet type %d is a fragment, not a whole message",
			packet)
	}

	if pid := binary.BigEndian.Uint16(buf[1:]); pid != PID {
		return Transport{}, fmt.Errorf("avrcp: profile %#04x is not this one", pid)
	}

	return Transport{
		Label:   buf[0] >> 4,
		Type:    buf[0] >> 1 & 1,
		Invalid: buf[0]&1 != 0,
		Payload: buf[avctpHeader:],
	}, nil
}

func (t Transport) Marshal() []byte {
	out := make([]byte, avctpHeader+len(t.Payload))

	out[0] = t.Label<<4 | PacketSingle<<2 | t.Type&1<<1
	if t.Invalid {
		out[0] |= 1
	}
	binary.BigEndian.PutUint16(out[1:], PID)
	copy(out[avctpHeader:], t.Payload)

	return out
}

type Reassembler struct {
	label   byte
	partial []byte
	left    int
	holding bool
}

func (r *Reassembler) Push(buf []byte) ([]byte, error) {
	if len(buf) < avctpHeader {
		return nil, ErrShort
	}

	label := buf[0] >> 4
	packet := buf[0] >> 2 & 0x3

	switch packet {
	case PacketSingle:
		r.reset()
		return buf, nil

	case PacketStart:
		// A start packet carries a packet count between the label byte and the PID.
		if len(buf) < avctpHeader+1 {
			return nil, ErrShort
		}

		count := int(buf[1])
		if count < 2 {
			return nil, fmt.Errorf("avrcp: a start packet claiming %d packets", count)
		}

		r.label = label
		r.left = count - 1
		r.holding = true

		r.partial = append([]byte(nil), buf[0]&^byte(0x0c), buf[2], buf[3])
		r.partial = append(r.partial, buf[avctpHeader+1:]...)
		return nil, nil
	}

	if !r.holding || label != r.label || r.left <= 0 {
		r.reset()
		return nil, fmt.Errorf("avrcp: a %s packet for label %d with no run to join",
			packetName(packet), label)
	}

	r.partial = append(r.partial, buf[avctpHeader:]...)
	r.left--

	if packet != PacketEnd {
		return nil, nil
	}

	whole := r.partial
	short := r.left != 0
	r.reset()

	if short {
		return nil, fmt.Errorf("avrcp: a message ended %d packets early", r.left)
	}
	return whole, nil
}

func (r *Reassembler) reset() {
	r.partial = nil
	r.left = 0
	r.holding = false
}

func packetName(packet byte) string {
	switch packet {
	case PacketStart:
		return "start"
	case PacketContinue:
		return "continue"
	case PacketEnd:
		return "end"
	}
	return "single"
}
