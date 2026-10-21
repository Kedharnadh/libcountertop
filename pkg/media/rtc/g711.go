package rtc

const (
	g711Rate = 8000
	widening = PlayRate / g711Rate
)

func ulaw(u byte) int16 {
	u = ^u
	t := (int32(u&0x0f)<<3 + 0x84) << ((u & 0x70) >> 4)
	if u&0x80 != 0 {
		return int16(0x84 - t)
	}
	return int16(t - 0x84)
}

func alaw(a byte) int16 {
	a ^= 0x55
	t := int32(a&0x0f) << 4
	switch seg := (a & 0x70) >> 4; seg {
	case 0:
		t += 8
	case 1:
		t += 0x108
	default:
		t = (t + 0x108) << (seg - 1)
	}
	if a&0x80 != 0 {
		return int16(t)
	}
	return int16(-t)
}

func widen(narrow []int16, before int16) []int16 {
	out := make([]int16, 0, len(narrow)*widening*PlayChannels)
	from := int32(before)
	for _, s := range narrow {
		to := int32(s)
		for k := 1; k <= widening; k++ {
			v := int16(from + (to-from)*int32(k)/widening)
			for range PlayChannels {
				out = append(out, v)
			}
		}
		from = to
	}
	return out
}
