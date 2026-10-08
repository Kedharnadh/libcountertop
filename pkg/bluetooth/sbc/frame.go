package sbc

import (
	"errors"
	"fmt"
)

const Syncword = 0x9c

var ErrShort = errors.New("sbc: shorter than a whole frame")

const (
	Mono = iota
	DualChannel
	Stereo
	JointStereo
)

const (
	Loudness = 0
	SNR      = 1
)

const headerBytes = 4

var (
	rates    = [4]int{16000, 32000, 44100, 48000}
	blocks   = [4]int{4, 8, 12, 16}
	subbands = [2]int{4, 8}
)

type Header struct {
	Rate       int
	Blocks     int
	Mode       int
	Allocation int
	Subbands   int
	Bitpool    int

	CRC byte
}

func (h Header) Channels() int {
	if h.Mode == Mono {
		return 1
	}
	return 2
}

func (h Header) Joint() bool { return h.Mode == JointStereo }

func (h Header) Samples() int { return h.Blocks * h.Subbands }

func (h Header) Length() int {
	scale := 4 * h.Subbands * h.Channels() / 8

	var bits int
	switch h.Mode {
	case Mono, DualChannel:
		bits = h.Blocks * h.Channels() * h.Bitpool
	case Stereo:
		bits = h.Blocks * h.Bitpool
	default: // joint
		bits = h.Subbands + h.Blocks*h.Bitpool
	}

	return headerBytes + scale + (bits+7)/8
}

func ParseHeader(buf []byte) (Header, error) {
	if len(buf) < headerBytes {
		return Header{}, ErrShort
	}
	if buf[0] != Syncword {
		return Header{}, fmt.Errorf("sbc: %#02x is not the syncword", buf[0])
	}

	h := Header{
		Rate:       rates[buf[1]>>6&0x3],
		Blocks:     blocks[buf[1]>>4&0x3],
		Mode:       int(buf[1] >> 2 & 0x3),
		Allocation: int(buf[1] >> 1 & 0x1),
		Subbands:   subbands[buf[1]&0x1],
		Bitpool:    int(buf[2]),
		CRC:        buf[3],
	}

	if h.Bitpool == 0 {
		return Header{}, errors.New("sbc: a frame with no bitpool")
	}

	// The spec caps the bitpool by mode and subbands.
	if max := h.maxBitpool(); h.Bitpool > max {
		return Header{}, fmt.Errorf(
			"sbc: bitpool %d, above the %d this mode allows",
			h.Bitpool,
			max,
		)
	}

	return h, nil
}

func (h Header) maxBitpool() int {
	if h.Mode == Mono || h.Mode == DualChannel {
		return 16 * h.Subbands
	}
	return 32 * h.Subbands
}

// x^8 + x^4 + x^3 + x^2 + 1, over the header and scale factors.
var crcTable = func() [256]byte {
	var t [256]byte
	for i := range 256 {
		c := byte(i)
		for range 8 {
			if c&0x80 != 0 {
				c = c<<1 ^ 0x1d
			} else {
				c <<= 1
			}
		}
		t[i] = c
	}
	return t
}()

// In joint stereo the CRC span does not end on a byte boundary.
func crcBits(buf []byte, bits int) byte {
	crc := byte(0x0f)

	whole := bits / 8
	for _, b := range buf[:whole] {
		crc = crcTable[crc^b]
	}

	// The leftover bits, most significant first.
	for i := range bits % 8 {
		bit := buf[whole] >> (7 - i) & 1

		top := crc >> 7 & 1
		crc <<= 1
		if top^bit != 0 {
			crc ^= 0x1d
		}
	}
	return crc
}

// Covers header bytes 1 and 2, the join bits and every scale factor.
func CRC(h Header, frame []byte) (byte, error) {
	if len(frame) < h.Length() {
		return 0, ErrShort
	}

	// Two header bytes, the join bits, and four bits per scale factor.
	bits := 16
	if h.Joint() {
		bits += h.Subbands
	}
	bits += 4 * h.Subbands * h.Channels()

	covered := make([]byte, 0, (bits+7)/8)
	covered = append(covered, frame[1], frame[2])
	covered = append(covered, frame[headerBytes:h.Length()]...)

	return crcBits(covered, bits), nil
}

func Valid(h Header, frame []byte) bool {
	got, err := CRC(h, frame)
	return err == nil && got == h.CRC
}

func Find(buf []byte) (int, bool) {
	for i := range buf {
		if buf[i] != Syncword {
			continue
		}

		h, err := ParseHeader(buf[i:])
		if err != nil {
			continue
		}

		end := i + h.Length()
		if end > len(buf) || !Valid(h, buf[i:]) {
			continue
		}
		if end < len(buf) && buf[end] != Syncword {
			continue
		}
		return i, true
	}
	return 0, false
}
