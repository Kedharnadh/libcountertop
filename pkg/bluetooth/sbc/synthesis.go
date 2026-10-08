package sbc

import "fmt"

// Products wrap modulo 2^32, as in the reference.

// Ten blocks of history: 8 phases of 10 for four subbands, 16 of 10 for eight.
const (
	history = 10

	spill = history - 1
)

type Filter struct {
	subbands int
	channels int

	v [][]int32

	at [][]int
}

type Samples [][]int16

func (f *Filter) Synthesize(fr *Frame) (Samples, error) {
	h := fr.Header
	if h.Subbands != 4 && h.Subbands != 8 {
		return nil, fmt.Errorf("sbc: %d subbands, want 4 or 8", h.Subbands)
	}
	if len(fr.Samples) != h.Blocks {
		return nil, fmt.Errorf("sbc: %d blocks of samples for a %d block frame",
			len(fr.Samples), h.Blocks)
	}

	f.reshape(h)

	out := make(Samples, h.Channels())
	for ch := range out {
		out[ch] = make([]int16, h.Blocks*h.Subbands)
	}

	for blk := range h.Blocks {
		if len(fr.Samples[blk]) != h.Channels() {
			return nil, fmt.Errorf("sbc: block %d has %d channels, want %d",
				blk, len(fr.Samples[blk]), h.Channels())
		}

		for ch := range h.Channels() {
			if len(fr.Samples[blk][ch]) != h.Subbands {
				return nil, fmt.Errorf("sbc: block %d channel %d has %d subbands, want %d",
					blk, ch, len(fr.Samples[blk][ch]), h.Subbands)
			}
			f.block(fr.Samples[blk][ch], ch, out[ch][blk*h.Subbands:])
		}
	}
	return out, nil
}

func (f *Filter) reshape(h Header) {
	if f.subbands == h.Subbands && f.channels == h.Channels() {
		return
	}

	f.subbands, f.channels = h.Subbands, h.Channels()

	phases := 2 * h.Subbands
	ring := phases * history

	f.v = make([][]int32, f.channels)
	f.at = make([][]int, f.channels)

	for ch := range f.v {
		f.v[ch] = make([]int32, ring+spill)
		f.at[ch] = make([]int, phases)

		for i := range f.at[ch] {
			f.at[ch][i] = history * (i + 1)
		}
	}
}

func (f *Filter) block(in []int32, ch int, out []int16) {
	matrix, m0, m1 := synMatrix4, proto4m0, proto4m1
	if f.subbands == 8 {
		matrix, m0, m1 = synMatrix8, proto8m0, proto8m1
	}

	v, at := f.v[ch], f.at[ch]
	phases := len(at)
	ring := phases * history

	for i := range phases {
		at[i]--
		if at[i] < 0 {
			at[i] = ring - 1

			copy(v[ring:], v[:spill])
		}

		var acc uint32
		for sb, s := range in {
			acc += uint32(matrix[i][sb]) * uint32(s)
		}
		v[at[i]] = int32(acc) >> 15
	}

	for i := range f.subbands {
		lo, hi := at[i], at[(i+f.subbands)&(phases-1)]

		var acc uint32
		for j := range history / 2 {
			acc += uint32(v[lo+2*j]) * uint32(m0[i*(history/2)+j])
			acc += uint32(v[hi+2*j+1]) * uint32(m1[i*(history/2)+j])
		}
		out[i] = clip16(int32(acc) >> 15)
	}
}

func clip16(v int32) int16 {
	switch {
	case v > 0x7fff:
		return 0x7fff
	case v < -0x8000:
		return -0x8000
	}
	return int16(v)
}
