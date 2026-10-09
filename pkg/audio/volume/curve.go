package volume

import "math"

// MEDIA_VOLUME_CURVE from Android's volume_tables.xml: index against attenuation in millibels.
var curve = []struct{ at, mB int }{
	{1, -5800}, {20, -4000}, {60, -1700}, {100, 0},
}

func Gain(percent int) float32 {
	percent = Clamp(percent)
	if percent == 0 {
		return 0
	}

	mB := float64(curve[0].mB)
	for i := 1; i < len(curve); i++ {
		lo, hi := curve[i-1], curve[i]
		if percent > hi.at {
			continue
		}
		if percent <= lo.at {
			break
		}

		across := float64(percent-lo.at) / float64(hi.at-lo.at)
		mB = float64(lo.mB) + across*float64(hi.mB-lo.mB)
		break
	}
	return float32(math.Pow(10, mB/2000))
}
