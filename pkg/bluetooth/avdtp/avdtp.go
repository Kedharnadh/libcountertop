package avdtp

import (
	"errors"
	"fmt"
)

var ErrShort = errors.New("avdtp: shorter than its header says")

const (
	SignalDiscover        = 0x01
	SignalGetCapabilities = 0x02
	SignalSetConfig       = 0x03
	SignalGetConfig       = 0x04
	SignalReconfigure     = 0x05
	SignalOpen            = 0x06
	SignalStart           = 0x07
	SignalClose           = 0x08
	SignalSuspend         = 0x09
	SignalAbort           = 0x0a
	SignalSecurity        = 0x0b
	SignalGetAllCaps      = 0x0c
	SignalDelayReport     = 0x0d
)

const (
	Command        = 0
	GeneralReject  = 1
	ResponseAccept = 2
	ResponseReject = 3
)

const (
	PacketSingle   = 0
	PacketStart    = 1
	PacketContinue = 2
	PacketEnd      = 3
)

type Message struct {
	// Four bits; wraps at sixteen.
	Label byte

	Type   byte
	Signal byte
	Data   []byte
}

func ParseMessage(buf []byte) (Message, error) {
	if len(buf) < 2 {
		return Message{}, ErrShort
	}

	packet := buf[0] >> 2 & 0x3
	if packet != PacketSingle {
		return Message{}, fmt.Errorf("avdtp: a fragmented message, packet type %d", packet)
	}

	return Message{
		Label:  buf[0] >> 4,
		Type:   buf[0] & 0x3,
		Signal: buf[1] & 0x3f,
		Data:   buf[2:],
	}, nil
}

func (m Message) Marshal() []byte {
	out := make([]byte, 2+len(m.Data))
	out[0] = m.Label<<4 | PacketSingle<<2 | m.Type&0x3
	out[1] = m.Signal & 0x3f
	copy(out[2:], m.Data)
	return out
}

func (m Message) Accept(data ...byte) Message {
	return Message{Label: m.Label, Type: ResponseAccept, Signal: m.Signal, Data: data}
}

// Configuration signals put the failing category before the error code.
func (m Message) Reject(code byte) Message {
	return Message{Label: m.Label, Type: ResponseReject, Signal: m.Signal, Data: []byte{code}}
}

func (m Message) RejectConfig(category, code byte) Message {
	return Message{Label: m.Label, Type: ResponseReject, Signal: m.Signal,
		Data: []byte{category, code}}
}

const (
	ErrorBadHeaderFormat   = 0x01
	ErrorBadLength         = 0x11
	ErrorBadSEID           = 0x12
	ErrorSEPInUse          = 0x13
	ErrorSEPNotInUse       = 0x14
	ErrorBadCategory       = 0x17
	ErrorBadPayloadFormat  = 0x18
	ErrorNotSupported      = 0x19
	ErrorBadState          = 0x31
	ErrorUnsupportedConfig = 0x29
)

const (
	MediaAudio      = 0x00
	MediaVideo      = 0x01
	MediaMultimedia = 0x02
)

const (
	Source = 0
	Sink   = 1
)

type SEP struct {
	// Six bits; zero is reserved.
	SEID byte

	InUse bool
	Media byte
	TSEP  byte
}

const sepBytes = 2

func (s SEP) Marshal() []byte {
	first := s.SEID << 2
	if s.InUse {
		first |= 1 << 1
	}
	return []byte{first, s.Media<<4 | s.TSEP<<3}
}

func ParseSEPs(data []byte) ([]SEP, error) {
	if len(data)%sepBytes != 0 {
		return nil, fmt.Errorf(
			"avdtp: %d bytes of endpoints, not a whole number of them",
			len(data),
		)
	}

	out := make([]SEP, 0, len(data)/sepBytes)
	for i := 0; i < len(data); i += sepBytes {
		out = append(out, SEP{
			SEID:  data[i] >> 2,
			InUse: data[i]&(1<<1) != 0,
			Media: data[i+1] >> 4,
			TSEP:  data[i+1] >> 3 & 1,
		})
	}
	return out, nil
}

// The SEID is the top six bits of the first byte.
func SEID(data []byte) (byte, bool) {
	if len(data) < 1 {
		return 0, false
	}
	return data[0] >> 2, true
}

func AddressTo(seid byte) byte { return seid << 2 }
