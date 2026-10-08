package sdp

import (
	"encoding/binary"
	"fmt"
)

// The two sequences are data elements; the byte cap between them is not.
func (s Search) Request(transaction uint16) (PDU, error) {
	pattern := make([]Element, 0, len(s.Pattern))
	for _, u := range s.Pattern {
		if u <= 0xffff {
			pattern = append(pattern, UUID16(uint16(u)))
			continue
		}
		pattern = append(pattern, UUID32(u))
	}

	classes, err := Sequence(pattern...).Marshal()
	if err != nil {
		return PDU{}, err
	}

	wanted := make([]Element, 0, len(s.Attributes))
	for _, a := range s.Attributes {
		wanted = append(wanted, a.Element())
	}

	attributes, err := Sequence(wanted...).Marshal()
	if err != nil {
		return PDU{}, err
	}

	params := make([]byte, 0, len(classes)+2+len(attributes)+1+len(s.Continuation))
	params = append(params, classes...)
	params = binary.BigEndian.AppendUint16(params, s.MaxBytes)
	params = append(params, attributes...)

	// The continuation is a length and that many bytes.
	params = append(params, byte(len(s.Continuation)))
	params = append(params, s.Continuation...)

	return PDU{ID: PDUSearchAttributeRequest, Transaction: transaction, Params: params}, nil
}

// A non-empty continuation is echoed back to get the rest.
func ParseSearchAttributeResponse(params []byte) (records []Record, more []byte, err error) {
	if len(params) < 3 {
		return nil, nil, ErrShort
	}

	n := int(binary.BigEndian.Uint16(params))
	if len(params) < 2+n+1 {
		return nil, nil, fmt.Errorf("sdp: an answer says %d bytes of records and %d arrived",
			n, len(params)-2)
	}

	body := params[2 : 2+n]
	rest := params[2+n:]

	if size := int(rest[0]); len(rest) < 1+size {
		return nil, nil, ErrShort
	} else if size > 0 {
		more = rest[1 : 1+size]
	}

	// An empty body with a continuation is legal.
	if n == 0 {
		return nil, more, nil
	}

	outer, _, err := ParseElement(body)
	if err != nil {
		return nil, more, err
	}
	if outer.Type != TypeSequence {
		return nil, more, fmt.Errorf("sdp: the answer is %#02x rather than a list of records",
			outer.Type)
	}

	for _, e := range outer.Children {
		r, err := record(e)
		if err != nil {
			return records, more, err
		}
		records = append(records, r)
	}
	return records, more, nil
}

// A record is one flat sequence of alternating identifiers and values.
func record(e Element) (Record, error) {
	if e.Type != TypeSequence {
		return nil, fmt.Errorf("sdp: a record is %#02x rather than a sequence", e.Type)
	}
	if len(e.Children)%2 != 0 {
		return nil, fmt.Errorf("sdp: a record of %d fields, which cannot pair up", len(e.Children))
	}

	out := make(Record, 0, len(e.Children)/2)
	for i := 0; i < len(e.Children); i += 2 {
		id, ok := e.Children[i].Uint()
		if !ok {
			return nil, fmt.Errorf("sdp: an attribute identifier is %#02x rather than a number",
				e.Children[i].Type)
		}
		out = append(out, Attribute{ID: uint16(id), Value: e.Children[i+1]})
	}
	return out, nil
}

func Classes(r Record) []uint32 {
	list, ok := r.Attribute(AttrServiceClasses)
	if !ok {
		return nil
	}

	out := make([]uint32, 0, len(list.Children))
	for _, c := range list.Children {
		if v, ok := c.Uint(); ok {
			out = append(out, v)
		}
	}
	return out
}

// Cover art and browsing PSMs are in the additional protocol descriptor lists.
func MorePSM(r Record, under uint32) (uint16, bool) {
	lists, ok := r.Attribute(AttrMoreProtocols)
	if !ok {
		return 0, false
	}

	for _, list := range lists.Children {
		if !carries(list, under) {
			continue
		}
		if psm, ok := PSM(list); ok {
			return psm, true
		}
	}
	return 0, false
}

func carries(list Element, uuid uint32) bool {
	for _, layer := range list.Children {
		for _, field := range layer.Children {
			if field.Type != TypeUUID {
				continue
			}
			if v, ok := field.Uint(); ok && v == uuid {
				return true
			}
		}
	}
	return false
}
