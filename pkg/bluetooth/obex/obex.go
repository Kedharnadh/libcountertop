package obex

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrShort = errors.New("obex: shorter than its header says")

// The high bit marks the final packet.
const (
	OpConnect    = 0x80
	OpDisconnect = 0x81
	OpPut        = 0x02
	OpGet        = 0x03
	OpAbort      = 0xff

	Final = 0x80
)

// Modelled on HTTP status codes.
const (
	Continue    = 0x90
	OK          = 0xa0
	BadRequest  = 0xc0
	NotFound    = 0xc4
	Unsupported = 0xc3
)

// The top two bits give the value encoding.
const (
	HeaderCount       = 0xc0
	HeaderName        = 0x01
	HeaderType        = 0x42
	HeaderLength      = 0xc3
	HeaderBody        = 0x48
	HeaderEndOfBody   = 0x49
	HeaderWho         = 0x4a
	HeaderConnection  = 0xcb
	HeaderTarget      = 0x46
	HeaderDescription = 0x05
)

const (
	encodingUnicode = 0x00
	encodingBytes   = 0x40
	encodingByte    = 0x80
	encodingUint32  = 0xc0
)

// Unicode headers are UTF-16BE with a two-byte null terminator.
type Header struct {
	ID    byte
	Value []byte
}

func Text(id byte, s string) Header {
	out := make([]byte, 0, len(s)*2+2)
	for _, r := range s {
		if r > 0xffff {
			r = 0xfffd
		}
		out = binary.BigEndian.AppendUint16(out, uint16(r))
	}
	return Header{ID: id, Value: append(out, 0, 0)}
}

func Bytes(id byte, v []byte) Header { return Header{ID: id, Value: v} }

func Uint32(id byte, v uint32) Header {
	return Header{ID: id, Value: binary.BigEndian.AppendUint32(nil, v)}
}

func (h Header) Uint32() (uint32, bool) {
	if h.ID&0xc0 != encodingUint32 || len(h.Value) != 4 {
		return 0, false
	}
	return binary.BigEndian.Uint32(h.Value), true
}

func (h Header) Marshal() []byte {
	switch h.ID & 0xc0 {
	case encodingByte:
		v := byte(0)
		if len(h.Value) > 0 {
			v = h.Value[0]
		}
		return []byte{h.ID, v}

	case encodingUint32:
		out := make([]byte, 5)
		out[0] = h.ID
		copy(out[1:], h.Value)
		return out
	}

	// The length counts the identifier and itself as well as the value.
	out := make([]byte, 3, 3+len(h.Value))
	out[0] = h.ID
	binary.BigEndian.PutUint16(out[1:], uint16(3+len(h.Value)))
	return append(out, h.Value...)
}

func ParseHeaders(buf []byte) ([]Header, error) {
	var out []Header

	for len(buf) > 0 {
		id := buf[0]

		switch id & 0xc0 {
		case encodingByte:
			if len(buf) < 2 {
				return out, ErrShort
			}
			out = append(out, Header{ID: id, Value: []byte{buf[1]}})
			buf = buf[2:]

		case encodingUint32:
			if len(buf) < 5 {
				return out, ErrShort
			}
			out = append(out, Header{ID: id, Value: buf[1:5]})
			buf = buf[5:]

		case encodingUnicode, encodingBytes:
			if len(buf) < 3 {
				return out, ErrShort
			}

			n := int(binary.BigEndian.Uint16(buf[1:]))
			if n < 3 || len(buf) < n {
				return out, fmt.Errorf("obex: a header says %d bytes and %d are left", n, len(buf))
			}
			out = append(out, Header{ID: id, Value: buf[3:n]})
			buf = buf[n:]

		default:
			return out, fmt.Errorf("obex: a header encoded %#02x, which is not one of the four",
				id&0xc0)
		}
	}
	return out, nil
}

func Find(headers []Header, id byte) (Header, bool) {
	for _, h := range headers {
		if h.ID == id {
			return h, true
		}
	}
	return Header{}, false
}
