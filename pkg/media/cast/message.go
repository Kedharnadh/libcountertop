package cast

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

var ErrShort = errors.New("cast: shorter than its header says")

// The field numbers of CastMessage, from cast_channel.proto.
const (
	fieldProtocol      = 1
	fieldSource        = 2
	fieldDestination   = 3
	fieldNamespace     = 4
	fieldPayloadType   = 5
	fieldPayloadUTF8   = 6
	fieldPayloadBinary = 7
)

// The only protocol version there has been.
const ProtocolV2 = 0

const (
	PayloadString = 0
	PayloadBinary = 1
)

type Message struct {
	// Endpoints within the connection: "sender-0", "receiver-0", or an application's transport id.
	Source      string
	Destination string

	Namespace string

	Payload string
	Binary  []byte
}

const Broadcast = "*"

const (
	SenderID   = "sender-0"
	ReceiverID = "receiver-0"
)

func (m Message) Marshal() []byte {
	var out []byte

	out = protowire.AppendTag(out, fieldProtocol, protowire.VarintType)
	out = protowire.AppendVarint(out, ProtocolV2)

	out = protowire.AppendTag(out, fieldSource, protowire.BytesType)
	out = protowire.AppendString(out, m.Source)

	out = protowire.AppendTag(out, fieldDestination, protowire.BytesType)
	out = protowire.AppendString(out, m.Destination)

	out = protowire.AppendTag(out, fieldNamespace, protowire.BytesType)
	out = protowire.AppendString(out, m.Namespace)

	kind := PayloadString
	if m.Binary != nil {
		kind = PayloadBinary
	}
	out = protowire.AppendTag(out, fieldPayloadType, protowire.VarintType)
	out = protowire.AppendVarint(out, uint64(kind))

	if m.Binary != nil {
		out = protowire.AppendTag(out, fieldPayloadBinary, protowire.BytesType)
		out = protowire.AppendBytes(out, m.Binary)
	} else {
		out = protowire.AppendTag(out, fieldPayloadUTF8, protowire.BytesType)
		out = protowire.AppendString(out, m.Payload)
	}

	return out
}

func Unmarshal(buf []byte) (Message, error) {
	var m Message

	for len(buf) > 0 {
		number, kind, n := protowire.ConsumeTag(buf)
		if n < 0 {
			return Message{}, fmt.Errorf("cast: %w", protowire.ParseError(n))
		}
		buf = buf[n:]

		switch {
		case number == fieldSource && kind == protowire.BytesType:
			m.Source, n = protowire.ConsumeString(buf)
		case number == fieldDestination && kind == protowire.BytesType:
			m.Destination, n = protowire.ConsumeString(buf)
		case number == fieldNamespace && kind == protowire.BytesType:
			m.Namespace, n = protowire.ConsumeString(buf)
		case number == fieldPayloadUTF8 && kind == protowire.BytesType:
			m.Payload, n = protowire.ConsumeString(buf)

		case number == fieldPayloadBinary && kind == protowire.BytesType:
			var b []byte
			b, n = protowire.ConsumeBytes(buf)
			if n >= 0 {
				m.Binary = append([]byte(nil), b...)
			}

		default:
			n = protowire.ConsumeFieldValue(number, kind, buf)
		}

		if n < 0 {
			return Message{}, fmt.Errorf("cast: field %d: %w", number, protowire.ParseError(n))
		}
		buf = buf[n:]
	}

	return m, nil
}
