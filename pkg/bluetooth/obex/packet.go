package obex

import (
	"encoding/binary"
	"fmt"
)

// Opcode, length including itself, headers; connect and its answer add four fixed bytes.

const packetHeader = 3

// 1.0.
const Version = 0x10

type Packet struct {
	Code byte

	Headers []Header

	// Only in a connect and its answer.
	Version byte
	Flags   byte
	MTU     uint16
}

func connects(code byte) bool { return code == OpConnect }

func Connect(mtu uint16, target []byte) Packet {
	p := Packet{Code: OpConnect, Version: Version, MTU: mtu}
	if len(target) > 0 {
		p.Headers = append(p.Headers, Bytes(HeaderTarget, target))
	}
	return p
}

func Get(connection uint32, headers ...Header) Packet {
	return Packet{
		Code:    OpGet | Final,
		Headers: append([]Header{Uint32(HeaderConnection, connection)}, headers...),
	}
}

// After Continue the target holds the rest until asked.
func More(connection uint32) Packet { return Get(connection) }

func Disconnect(connection uint32) Packet {
	return Packet{Code: OpDisconnect, Headers: []Header{Uint32(HeaderConnection, connection)}}
}

func (p Packet) Marshal() []byte {
	var body []byte

	if connects(p.Code) {
		body = []byte{p.Version, p.Flags, 0, 0}
		binary.BigEndian.PutUint16(body[2:], p.MTU)
	}

	for _, h := range p.Headers {
		body = append(body, h.Marshal()...)
	}

	out := make([]byte, packetHeader, packetHeader+len(body))
	out[0] = p.Code
	binary.BigEndian.PutUint16(out[1:], uint16(packetHeader+len(body)))

	return append(out, body...)
}

// Nothing in a response says whether it answers a connect.
func ParseAnswer(buf []byte, connected bool) (Packet, error) {
	if len(buf) < packetHeader {
		return Packet{}, ErrShort
	}

	n := int(binary.BigEndian.Uint16(buf[1:]))
	if n < packetHeader || len(buf) < n {
		return Packet{}, fmt.Errorf("obex: a packet says %d bytes and %d arrived", n, len(buf))
	}

	p := Packet{Code: buf[0]}
	body := buf[packetHeader:n]

	if connected {
		if len(body) < 4 {
			return p, ErrShort
		}
		p.Version, p.Flags = body[0], body[1]
		p.MTU = binary.BigEndian.Uint16(body[2:])
		body = body[4:]
	}

	headers, err := ParseHeaders(body)
	p.Headers = headers
	return p, err
}

func (p Packet) Connection() (uint32, bool) {
	h, ok := Find(p.Headers, HeaderConnection)
	if !ok {
		return 0, false
	}
	return h.Uint32()
}

// Each answer carries Body; the last carries EndOfBody.
func (p Packet) Body() (data []byte, done bool) {
	for _, h := range p.Headers {
		switch h.ID {
		case HeaderBody:
			data = append(data, h.Value...)
		case HeaderEndOfBody:
			data = append(data, h.Value...)
			done = true
		}
	}
	return data, done
}
