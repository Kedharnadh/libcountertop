package sdp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrShort = errors.New("sdp: shorter than its header says")

// The type is the top five bits of the first byte.
const (
	TypeNil         = 0
	TypeUint        = 1
	TypeInt         = 2
	TypeUUID        = 3
	TypeText        = 4
	TypeBool        = 5
	TypeSequence    = 6
	TypeAlternative = 7
	TypeURL         = 8
)

// Size index: a width for fixed types, the width of the length for variable ones.
const (
	size1   = 0
	size2   = 1
	size4   = 2
	size8   = 3
	size16  = 4
	size8b  = 5 // one byte of length follows
	size16b = 6 // two bytes
	size32b = 7 // four bytes
)

type Element struct {
	Type byte

	Value []byte

	Children []Element
}

func Uint(v uint32) Element {
	switch {
	case v <= 0xff:
		return Element{Type: TypeUint, Value: []byte{byte(v)}}
	case v <= 0xffff:
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(v))
		return Element{Type: TypeUint, Value: b}
	}

	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return Element{Type: TypeUint, Value: b}
}

// In an attribute ID list two bytes is one ID and four is a range.
func Uint16(v uint16) Element {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return Element{Type: TypeUint, Value: b}
}

func Uint32(v uint32) Element {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return Element{Type: TypeUint, Value: b}
}

func UUID16(v uint16) Element {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return Element{Type: TypeUUID, Value: b}
}

func UUID32(v uint32) Element {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return Element{Type: TypeUUID, Value: b}
}

func Text(s string) Element { return Element{Type: TypeText, Value: []byte(s)} }

func Bool(b bool) Element {
	v := byte(0)
	if b {
		v = 1
	}
	return Element{Type: TypeBool, Value: []byte{v}}
}

func Sequence(children ...Element) Element {
	return Element{Type: TypeSequence, Children: children}
}

func Alternative(children ...Element) Element {
	return Element{Type: TypeAlternative, Children: children}
}

func Nil() Element { return Element{Type: TypeNil} }

func nests(t byte) bool { return t == TypeSequence || t == TypeAlternative }

func (e Element) Marshal() ([]byte, error) {
	var body []byte

	if nests(e.Type) {
		for _, c := range e.Children {
			b, err := c.Marshal()
			if err != nil {
				return nil, err
			}
			body = append(body, b...)
		}
	} else {
		body = e.Value
	}

	head, err := header(e.Type, len(body))
	if err != nil {
		return nil, err
	}
	return append(head, body...), nil
}

func header(t byte, n int) ([]byte, error) {
	if t > TypeURL {
		return nil, fmt.Errorf("sdp: %d is not a data element type", t)
	}

	tag := func(index byte) byte { return t<<3 | index }

	// Nil uses size index 0 and carries nothing.
	if t == TypeNil {
		return []byte{tag(size1)}, nil
	}

	// Fixed width types name their width in the index and carry no length.
	if !nests(t) && t != TypeText && t != TypeURL {
		switch n {
		case 1:
			return []byte{tag(size1)}, nil
		case 2:
			return []byte{tag(size2)}, nil
		case 4:
			return []byte{tag(size4)}, nil
		case 8:
			return []byte{tag(size8)}, nil
		case 16:
			return []byte{tag(size16)}, nil
		}
		return nil, fmt.Errorf("sdp: a type %d value cannot be %d bytes", t, n)
	}

	switch {
	case n <= 0xff:
		return []byte{tag(size8b), byte(n)}, nil
	case n <= 0xffff:
		out := []byte{tag(size16b), 0, 0}
		binary.BigEndian.PutUint16(out[1:], uint16(n))
		return out, nil
	}

	out := []byte{tag(size32b), 0, 0, 0, 0}
	binary.BigEndian.PutUint32(out[1:], uint32(n))
	return out, nil
}

func ParseElement(buf []byte) (Element, int, error) {
	if len(buf) < 1 {
		return Element{}, 0, ErrShort
	}

	t := buf[0] >> 3
	index := buf[0] & 0x7

	if t > TypeURL {
		return Element{}, 0, fmt.Errorf("sdp: %d is not a data element type", t)
	}

	n, head, err := length(t, index, buf)
	if err != nil {
		return Element{}, 0, err
	}
	if len(buf) < head+n {
		return Element{}, 0, ErrShort
	}

	body := buf[head : head+n]

	if !nests(t) {
		return Element{Type: t, Value: body}, head + n, nil
	}

	e := Element{Type: t}
	for len(body) > 0 {
		child, used, err := ParseElement(body)
		if err != nil {
			return Element{}, 0, fmt.Errorf("sdp: inside a sequence: %w", err)
		}
		e.Children = append(e.Children, child)
		body = body[used:]
	}
	return e, head + n, nil
}

func length(t, index byte, buf []byte) (n, head int, err error) {
	if t == TypeNil {
		return 0, 1, nil
	}

	switch index {
	case size1:
		return 1, 1, nil
	case size2:
		return 2, 1, nil
	case size4:
		return 4, 1, nil
	case size8:
		return 8, 1, nil
	case size16:
		return 16, 1, nil

	case size8b:
		if len(buf) < 2 {
			return 0, 0, ErrShort
		}
		return int(buf[1]), 2, nil

	case size16b:
		if len(buf) < 3 {
			return 0, 0, ErrShort
		}
		return int(binary.BigEndian.Uint16(buf[1:])), 3, nil

	case size32b:
		if len(buf) < 5 {
			return 0, 0, ErrShort
		}
		return int(binary.BigEndian.Uint32(buf[1:])), 5, nil
	}
	return 0, 0, fmt.Errorf("sdp: size index %d", index)
}

func (e Element) Uint() (uint32, bool) {
	if e.Type != TypeUint && e.Type != TypeUUID {
		return 0, false
	}

	switch len(e.Value) {
	case 1:
		return uint32(e.Value[0]), true
	case 2:
		return uint32(binary.BigEndian.Uint16(e.Value)), true
	case 4:
		return binary.BigEndian.Uint32(e.Value), true
	}
	return 0, false
}
