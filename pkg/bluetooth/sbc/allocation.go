package sbc

// From the A2DP specification.
var offset4 = [4][4]int{
	{-1, 0, 0, 0},
	{-2, 0, 0, 1},
	{-2, 0, 0, 1},
	{-2, 0, 0, 1},
}

var offset8 = [4][8]int{
	{-2, 0, 0, 0, 0, 0, 0, 1},
	{-3, 0, 0, 0, 0, 0, 1, 2},
	{-4, 0, 0, 0, 0, 0, 1, 2},
	{-4, 0, 0, 0, 0, 0, 1, 2},
}

func rateIndex(rate int) int {
	for i, v := range rates {
		if v == rate {
			return i
		}
	}
	return 0
}

func offset(rate, subbands, sb int) int {
	i := rateIndex(rate)
	if subbands == 4 {
		return offset4[i][sb]
	}
	return offset8[i][sb]
}

// SNR uses the scale factor; loudness subtracts the offset and halves a positive result.
func bitneed(h Header, scale []int, ch int) []int {
	out := make([]int, h.Subbands)

	for sb := range h.Subbands {
		sf := scale[ch*h.Subbands+sb]

		if h.Allocation == SNR {
			out[sb] = sf
			continue
		}

		if sf == 0 {
			out[sb] = -5
			continue
		}

		loud := sf - offset(h.Rate, h.Subbands, sb)
		if loud > 0 {
			loud /= 2
		}
		out[sb] = loud
	}
	return out
}

// The A2DP specification pseudocode; the two leftover passes must run in this order.
func share(need []int, bitpool int) []int {
	bits := make([]int, len(need))

	most := need[0]
	for _, v := range need[1:] {
		most = max(most, v)
	}

	bitcount, slicecount := 0, 0
	slice := most + 1

	for {
		slice--
		bitcount += slicecount
		slicecount = 0

		for _, v := range need {
			switch {
			case v > slice+1 && v < slice+16:
				slicecount++
			case v == slice+1:
				slicecount += 2
			}
		}

		if bitcount+slicecount >= bitpool {
			break
		}
		if slice < -16 {
			break
		}
	}

	if bitcount+slicecount == bitpool {
		bitcount += slicecount
		slice--
	}

	for i, v := range need {
		if v < slice+2 {
			bits[i] = 0
			continue
		}
		bits[i] = min(v-slice, 16)
	}

	for i := 0; i < len(need) && bitcount < bitpool; i++ {
		switch {
		case bits[i] >= 2 && bits[i] < 16:
			bits[i]++
			bitcount++
		case need[i] == slice+1 && bitpool > bitcount+1:
			bits[i] = 2
			bitcount += 2
		}
	}

	for i := 0; i < len(need) && bitcount < bitpool; i++ {
		if bits[i] < 16 {
			bits[i]++
			bitcount++
		}
	}
	return bits
}

// Mono and dual channel have a pool per channel; stereo and joint share one.
func Allocate(h Header, scale []int) [][]int {
	out := make([][]int, h.Channels())

	if h.Mode == Mono || h.Mode == DualChannel {
		for ch := range h.Channels() {
			out[ch] = share(bitneed(h, scale, ch), h.Bitpool)
		}
		return out
	}

	// The specification alternates channels, then steps the subband.
	left, right := bitneed(h, scale, 0), bitneed(h, scale, 1)

	both := make([]int, 0, len(left)+len(right))
	for sb := range h.Subbands {
		both = append(both, left[sb], right[sb])
	}

	bits := share(both, h.Bitpool)

	out[0] = make([]int, h.Subbands)
	out[1] = make([]int, h.Subbands)
	for sb := range h.Subbands {
		out[0][sb] = bits[sb*2]
		out[1][sb] = bits[sb*2+1]
	}
	return out
}
