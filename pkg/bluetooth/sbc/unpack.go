package sbc

import "fmt"

// SBC packs MSB first.
type bitReader struct {
	buf []byte
	at  int // in bits
}

func (r *bitReader) read(n int) (uint32, error) {
	if r.at+n > len(r.buf)*8 {
		return 0, ErrShort
	}

	var v uint32
	for range n {
		v <<= 1
		if r.buf[r.at>>3]>>(7-r.at&7)&1 != 0 {
			v |= 1
		}
		r.at++
	}
	return v, nil
}

func (r *bitReader) align() {
	if r.at&7 != 0 {
		r.at += 8 - r.at&7
	}
}

type Frame struct {
	Header Header

	Join uint32

	// Channel-major.
	Scale []int

	// Indexed block, channel, subband.
	Samples [][][]int32
}

// The reference decoder's headroom.
const extraBits = 2

func Unpack(frame []byte) (*Frame, error) {
	h, err := ParseHeader(frame)
	if err != nil {
		return nil, err
	}
	if len(frame) < h.Length() {
		return nil, ErrShort
	}
	if !Valid(h, frame) {
		return nil, fmt.Errorf("sbc: the frame's check byte does not match its contents")
	}

	f := &Frame{Header: h}
	r := &bitReader{buf: frame[:h.Length()], at: headerBytes * 8}

	// nrof_subbands-1 join flags in nrof_subbands bits; the last is reserved.
	if h.Joint() {
		for sb := range h.Subbands - 1 {
			v, err := r.read(1)
			if err != nil {
				return nil, err
			}
			if v != 0 {
				f.Join |= 1 << sb
			}
		}
		if _, err := r.read(1); err != nil {
			return nil, err
		}
	}

	f.Scale = make([]int, h.Subbands*h.Channels())
	for i := range f.Scale {
		v, err := r.read(4)
		if err != nil {
			return nil, err
		}
		f.Scale[i] = int(v)
	}

	bits := Allocate(h, f.Scale)

	if err := f.readSamples(r, bits); err != nil {
		return nil, err
	}
	f.unjoin()

	return f, nil
}

// The reference's exact integer dequantisation; the test vectors check the low bits.
func (f *Frame) readSamples(r *bitReader, bits [][]int) error {
	h := f.Header

	levels := make([][]int64, h.Channels())
	for ch := range levels {
		levels[ch] = make([]int64, h.Subbands)
		for sb := range h.Subbands {
			levels[ch][sb] = int64(1)<<uint(bits[ch][sb]) - 1
		}
	}

	f.Samples = make([][][]int32, h.Blocks)
	for blk := range h.Blocks {
		f.Samples[blk] = make([][]int32, h.Channels())

		for ch := range h.Channels() {
			f.Samples[blk][ch] = make([]int32, h.Subbands)

			for sb := range h.Subbands {
				if levels[ch][sb] == 0 {
					continue
				}

				raw, err := r.read(bits[ch][sb])
				if err != nil {
					return err
				}

				shift := uint(f.Scale[ch*h.Subbands+sb] + 1 + extraBits)
				v := (int64(raw)<<1 | 1) << shift / levels[ch][sb]

				f.Samples[blk][ch][sb] = int32(v - 1<<shift)
			}
		}
	}

	r.align()
	return nil
}

func (f *Frame) unjoin() {
	if !f.Header.Joint() {
		return
	}

	for blk := range f.Header.Blocks {
		for sb := range f.Header.Subbands {
			if f.Join&(1<<sb) == 0 {
				continue
			}

			left := f.Samples[blk][0][sb]
			right := f.Samples[blk][1][sb]

			f.Samples[blk][0][sb] = left + right
			f.Samples[blk][1][sb] = left - right
		}
	}
}
