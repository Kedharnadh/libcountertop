package sdp

import (
	"encoding/binary"
	"fmt"
	"slices"
)

const (
	PDUError                   = 0x01
	PDUServiceSearchRequest    = 0x02
	PDUServiceSearchResponse   = 0x03
	PDUAttributeRequest        = 0x04
	PDUAttributeResponse       = 0x05
	PDUSearchAttributeRequest  = 0x06
	PDUSearchAttributeResponse = 0x07
)

const (
	ErrorBadVersion      = 0x0001
	ErrorBadRecordHandle = 0x0002
	ErrorBadSyntax       = 0x0003
	ErrorBadPDUSize      = 0x0004
	ErrorBadContinuation = 0x0005
	ErrorNoResources     = 0x0006
)

// ID, transaction, parameter length.
const pduHeader = 5

type PDU struct {
	ID byte

	// Pairs a response with its request.
	Transaction uint16

	Params []byte
}

func ParsePDU(buf []byte) (PDU, error) {
	if len(buf) < pduHeader {
		return PDU{}, ErrShort
	}

	n := int(binary.BigEndian.Uint16(buf[3:]))
	if len(buf) < pduHeader+n {
		return PDU{}, ErrShort
	}

	return PDU{
		ID:          buf[0],
		Transaction: binary.BigEndian.Uint16(buf[1:]),
		Params:      buf[pduHeader : pduHeader+n],
	}, nil
}

func (p PDU) Marshal() []byte {
	out := make([]byte, pduHeader+len(p.Params))
	out[0] = p.ID
	binary.BigEndian.PutUint16(out[1:], p.Transaction)
	binary.BigEndian.PutUint16(out[3:], uint16(len(p.Params)))
	copy(out[pduHeader:], p.Params)
	return out
}

func Error(transaction uint16, reason uint16) PDU {
	params := make([]byte, 2)
	binary.BigEndian.PutUint16(params, reason)

	return PDU{ID: PDUError, Transaction: transaction, Params: params}
}

type Search struct {
	// A record matches when it carries all of them, not any.
	Pattern []uint32

	// MaxCount for a search, MaxBytes for an attribute read.
	MaxCount uint16
	MaxBytes uint16

	Attributes []AttrRange

	Continuation []byte
}

type AttrRange struct{ First, Last uint16 }

func (r AttrRange) Contains(id uint16) bool { return id >= r.First && id <= r.Last }

func (r AttrRange) Element() Element {
	if r.First == r.Last {
		return Uint16(r.First)
	}
	return Uint32(uint32(r.First)<<16 | uint32(r.Last))
}

// UUID sequence, two-byte cap, attribute ID sequence, continuation.
func ParseSearchAttribute(params []byte) (Search, error) {
	var s Search

	pattern, used, err := ParseElement(params)
	if err != nil {
		return s, fmt.Errorf("sdp: the search pattern: %w", err)
	}
	if pattern.Type != TypeSequence {
		return s, fmt.Errorf("sdp: the search pattern is type %d, want a sequence", pattern.Type)
	}
	for _, u := range pattern.Children {
		v, ok := u.Uint()
		if !ok {
			return s, fmt.Errorf("sdp: a search pattern entry is not a uuid")
		}
		s.Pattern = append(s.Pattern, v)
	}
	params = params[used:]

	if len(params) < 2 {
		return s, ErrShort
	}
	s.MaxBytes = binary.BigEndian.Uint16(params)
	params = params[2:]

	list, used, err := ParseElement(params)
	if err != nil {
		return s, fmt.Errorf("sdp: the attribute list: %w", err)
	}
	if list.Type != TypeSequence {
		return s, fmt.Errorf("sdp: the attribute list is type %d, want a sequence", list.Type)
	}
	for _, a := range list.Children {
		v, ok := a.Uint()
		if !ok {
			return s, fmt.Errorf("sdp: an attribute list entry is not a number")
		}

		// Two bytes is one identifier and four is a range, packed as first then last.
		if len(a.Value) == 4 {
			s.Attributes = append(s.Attributes, AttrRange{
				First: uint16(v >> 16),
				Last:  uint16(v),
			})
			continue
		}
		s.Attributes = append(s.Attributes, AttrRange{First: uint16(v), Last: uint16(v)})
	}
	params = params[used:]

	// The continuation carries its own length byte.
	if len(params) < 1 {
		return s, ErrShort
	}
	n := int(params[0])
	if len(params) < 1+n {
		return s, ErrShort
	}
	s.Continuation = params[1 : 1+n]

	return s, nil
}

func (s Search) Wants(classes []uint32) bool {
	for _, want := range s.Pattern {
		if !slices.Contains(classes, want) {
			return false
		}
	}
	return true
}

func (s Search) Selected(r Record) Record {
	var out Record
	for _, a := range r {
		for _, want := range s.Attributes {
			if want.Contains(a.ID) {
				out = append(out, a)
				break
			}
		}
	}
	return out
}

// The outer sequence is required even for a single record.
func SearchAttributeResponse(transaction uint16, records []Record) (PDU, error) {
	lists := make([]Element, 0, len(records))
	for _, r := range records {
		lists = append(lists, r.Element())
	}

	body, err := Sequence(lists...).Marshal()
	if err != nil {
		return PDU{}, err
	}

	params := make([]byte, 2, 2+len(body)+1)
	binary.BigEndian.PutUint16(params, uint16(len(body)))
	params = append(params, body...)

	params = append(params, 0)

	return PDU{ID: PDUSearchAttributeResponse, Transaction: transaction, Params: params}, nil
}
