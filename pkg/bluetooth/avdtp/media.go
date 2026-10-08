package avdtp

import (
	"encoding/binary"
	"fmt"
)

// An RTP header, then one A2DP byte: a frame count or a fragment position.

const rtpHeader = 12

const rtpVersion = 2

type Media struct {
	Sequence uint16

	// RTP timestamp, in samples at the configured rate.
	Timestamp uint32

	SSRC uint32

	Fragmented bool
	Start      bool
	Last       bool

	// Frames in Payload, or for a fragment the pieces left including this one.
	Count int

	Payload []byte
}

func ParseMedia(buf []byte) (Media, error) {
	if len(buf) < rtpHeader+1 {
		return Media{}, ErrShort
	}

	if v := buf[0] >> 6; v != rtpVersion {
		return Media{}, fmt.Errorf("avdtp: rtp version %d, want %d", v, rtpVersion)
	}

	at := rtpHeader + int(buf[0]&0x0f)*4
	if buf[0]&0x10 != 0 {
		if len(buf) < at+4 {
			return Media{}, ErrShort
		}
		at += 4 + int(binary.BigEndian.Uint16(buf[at+2:]))*4
	}

	if len(buf) < at+1 {
		return Media{}, ErrShort
	}

	h := buf[at]
	return Media{
		Sequence:   binary.BigEndian.Uint16(buf[2:]),
		Timestamp:  binary.BigEndian.Uint32(buf[4:]),
		SSRC:       binary.BigEndian.Uint32(buf[8:]),
		Fragmented: h&0x80 != 0,
		Start:      h&0x40 != 0,
		Last:       h&0x20 != 0,
		Count:      int(h & 0x0f),
		Payload:    buf[at+1:],
	}, nil
}

func (m Media) Marshal() []byte {
	out := make([]byte, rtpHeader+1+len(m.Payload))

	out[0] = rtpVersion << 6
	out[1] = payloadType
	binary.BigEndian.PutUint16(out[2:], m.Sequence)
	binary.BigEndian.PutUint32(out[4:], m.Timestamp)
	binary.BigEndian.PutUint32(out[8:], m.SSRC)

	var h byte
	if m.Fragmented {
		h |= 0x80
	}
	if m.Start {
		h |= 0x40
	}
	if m.Last {
		h |= 0x20
	}
	out[rtpHeader] = h | byte(m.Count)&0x0f

	copy(out[rtpHeader+1:], m.Payload)
	return out
}

const payloadType = 96

type Reassembler struct {
	partial []byte
	left    int
}

func (r *Reassembler) Push(m Media) []byte {
	if !m.Fragmented {
		r.reset()
		return m.Payload
	}

	if m.Start {
		// The spec allows a one-piece fragment with start and last both set.
		if m.Last {
			r.reset()
			return m.Payload
		}

		r.partial = append([]byte(nil), m.Payload...)
		r.left = m.Count - 1
		return nil
	}

	if r.partial == nil || r.left <= 0 {
		r.reset()
		return nil
	}

	r.partial = append(r.partial, m.Payload...)
	r.left--

	if !m.Last {
		return nil
	}

	frame := r.partial
	short := r.left != 0
	r.reset()

	if short {
		return nil
	}
	return frame
}

func (r *Reassembler) reset() {
	r.partial = nil
	r.left = 0
}
