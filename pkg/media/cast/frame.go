package cast

import (
	"encoding/binary"
	"fmt"
)

// Each message is a four byte big-endian length, then that many bytes of protobuf.

const lengthBytes = 4

const MaxMessage = 1 << 20

func Frame(m Message) []byte {
	body := m.Marshal()

	out := make([]byte, lengthBytes+len(body))
	binary.BigEndian.PutUint32(out, uint32(len(body)))
	copy(out[lengthBytes:], body)

	return out
}

func Next(buf []byte) (Message, int, error) {
	if len(buf) < lengthBytes {
		return Message{}, 0, ErrShort
	}

	n := int(binary.BigEndian.Uint32(buf))
	if n > MaxMessage {
		return Message{}, 0, fmt.Errorf(
			"cast: a message claiming %d bytes, more than the %d allowed",
			n,
			MaxMessage,
		)
	}
	if len(buf) < lengthBytes+n {
		return Message{}, 0, ErrShort
	}

	m, err := Unmarshal(buf[lengthBytes : lengthBytes+n])
	if err != nil {
		return Message{}, 0, err
	}
	return m, lengthBytes + n, nil
}
