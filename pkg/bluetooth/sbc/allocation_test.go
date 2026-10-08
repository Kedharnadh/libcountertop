package sbc

import "testing"

func stereoHeader(t *testing.T, alloc, bitpool int) Header {
	t.Helper()

	h, err := ParseHeader(header(44100, 16, Stereo, alloc, 8, bitpool))
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	return h
}

func total(bits [][]int) int {
	var n int
	for _, ch := range bits {
		for _, b := range ch {
			n += b
		}
	}
	return n
}

func TestTheAllocationNeverExceedsTheBitpool(t *testing.T) {
	for _, alloc := range []int{Loudness, SNR} {
		for _, bitpool := range []int{2, 10, 26, 53, 128, 250} {
			h := stereoHeader(t, alloc, bitpool)

			scale := make([]int, h.Subbands*h.Channels())
			for i := range scale {
				scale[i] = i % 12
			}

			got := total(Allocate(h, scale))
			if got > bitpool {
				t.Errorf("allocation %d bitpool %d: spent %d bits, over budget",
					alloc, bitpool, got)
			}
		}
	}
}

// At most sixteen bits per subband.
func TestNoSubbandTakesMoreThanSixteenBits(t *testing.T) {
	h := stereoHeader(t, SNR, 250)

	scale := make([]int, h.Subbands*h.Channels())
	for i := range scale {
		scale[i] = 15
	}

	for ch, bits := range Allocate(h, scale) {
		for sb, b := range bits {
			if b > 16 {
				t.Errorf("channel %d subband %d got %d bits", ch, sb, b)
			}
			if b < 0 {
				t.Errorf("channel %d subband %d got %d bits", ch, sb, b)
			}
		}
	}
}

func TestSilentSubbandsGetNothingWhenOthersNeedIt(t *testing.T) {
	h := stereoHeader(t, Loudness, 26)

	scale := make([]int, h.Subbands*h.Channels())
	scale[0] = 12
	scale[h.Subbands] = 12

	bits := Allocate(h, scale)

	if bits[0][0] == 0 {
		t.Error("the only subband carrying signal got no bits")
	}

	var quiet int
	for _, b := range bits[0][1:] {
		quiet += b
	}
	if quiet >= bits[0][0] {
		t.Errorf("the silent subbands got %d bits against the loud one's %d", quiet, bits[0][0])
	}
}

func TestDualChannelGetsAPoolEachAndStereoShares(t *testing.T) {
	scale := make([]int, 16)
	for i := range scale {
		scale[i] = 8
	}

	dual, err := ParseHeader(header(44100, 16, DualChannel, SNR, 8, 40))
	if err != nil {
		t.Fatalf("ParseHeader: %v", err)
	}
	stereo := stereoHeader(t, SNR, 40)

	d := total(Allocate(dual, scale))
	s := total(Allocate(stereo, scale))

	// Stereo shares one pool of 40. Dual channel gets one each, so up to 80.
	if s > 40 {
		t.Errorf("stereo spent %d against a shared pool of 40", s)
	}
	if d > 80 {
		t.Errorf("dual channel spent %d against two pools of 40", d)
	}
	if d <= s {
		t.Errorf("dual channel spent %d and stereo %d; two pools should buy more than one", d, s)
	}
}

func TestTheAllocationIsDeterministic(t *testing.T) {
	h := stereoHeader(t, Loudness, 53)

	scale := []int{5, 3, 9, 1, 0, 7, 2, 11, 4, 8, 0, 6, 10, 2, 1, 3}

	first := Allocate(h, scale)
	for range 5 {
		again := Allocate(h, scale)
		for ch := range first {
			for sb := range first[ch] {
				if first[ch][sb] != again[ch][sb] {
					t.Fatalf("channel %d subband %d gave %d then %d",
						ch, sb, first[ch][sb], again[ch][sb])
				}
			}
		}
	}
}

func TestAPoolLargerThanTheSignalNeeds(t *testing.T) {
	h := stereoHeader(t, SNR, 250)

	scale := make([]int, h.Subbands*h.Channels())

	if got := total(Allocate(h, scale)); got > 250 {
		t.Errorf("spent %d bits on silence", got)
	}
}

func TestLoudnessAndSNRAllocateDifferently(t *testing.T) {
	scale := []int{8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8, 8}

	loud := Allocate(stereoHeader(t, Loudness, 40), scale)
	snr := Allocate(stereoHeader(t, SNR, 40), scale)

	same := true
	for ch := range loud {
		for sb := range loud[ch] {
			if loud[ch][sb] != snr[ch][sb] {
				same = false
			}
		}
	}
	if same {
		t.Error("loudness and snr allocation gave the same answer, so the tables are not applied")
	}
}

func TestTheLoudnessTableFollowsTheSamplingRate(t *testing.T) {
	if offset(16000, 8, 0) == offset(44100, 8, 0) {
		t.Error("16000 and 44100 share a loudness offset for subband 0, so the rate is ignored")
	}
	if offset(44100, 4, 0) == offset(44100, 8, 0) {
		t.Error("four and eight subbands share an offset, so the table is not chosen by width")
	}

	if got := offset(96000, 8, 0); got != offset4[0][0] && got != offset8[0][0] {
		t.Errorf("an unknown rate gave offset %d", got)
	}
}
