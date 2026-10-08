package a2dp

import (
	"testing"

	speaker "github.com/ygelfand/libcountertop/pkg/audio/resample"
	"github.com/ygelfand/libcountertop/pkg/bluetooth/sbc"
	fixtures "github.com/ygelfand/libcountertop/pkg/bluetooth/sbc/testdata"
)

func TestThePlayPathFeedsTheCardAtItsOwnRate(t *testing.T) {
	for _, name := range []string{"tone", "mono", "sweep"} {
		t.Run(name, func(t *testing.T) {
			coded, err := fixtures.Read(name + ".sbc")
			if err != nil {
				t.Fatal(err)
			}

			start, ok := sbc.Find(coded)
			if !ok {
				t.Fatal("no frame in the vector")
			}
			first, err := sbc.Unpack(coded[start:])
			if err != nil {
				t.Fatal(err)
			}
			chose := first.Header.Rate

			s := testSink()
			s.start("rate test")
			s.resample = speaker.NewRational(chose, Rate, 2)

			var in, frames int
			for at := start; at < len(coded); {
				skip, ok := sbc.Find(coded[at:])
				if !ok {
					break
				}
				at += skip

				fr, err := sbc.Unpack(coded[at:])
				if err != nil {
					t.Fatalf("frame at %d: %v", at, err)
				}
				if fr.Header.Rate != chose {
					t.Fatalf("the rate changed mid stream, %d to %d", chose, fr.Header.Rate)
				}

				s.play(coded[at : at+fr.Header.Length()])
				at += fr.Header.Length()

				if frames++; frames > 1 {
					in += fr.Header.Blocks * fr.Header.Subbands
				} else {
					s.frames, s.bad, s.samples = 0, 0, 0
				}
			}

			if frames < 2 {
				t.Fatalf("%d frames in the vector, which measures nothing", frames)
			}
			if s.bad != 0 {
				t.Fatalf("%d of %d frames would not decode: %v", s.bad, s.frames, s.why)
			}

			// 44100:48000 is 147:160, so whole-sample counts wander by one.
			want := in * Rate / chose
			if s.samples < want-2 || s.samples > want+2 {
				t.Errorf("%d samples to the card from %d at %d, want %d",
					s.samples, in, chose, want)
			}
		})
	}
}

func TestAStreamAtTheCardsRateIsUntouched(t *testing.T) {
	r := speaker.NewRational(Rate, Rate, 2)

	if up, down := r.Ratio(); up != 1 || down != 1 {
		t.Fatalf("the card's own rate reduced to %d:%d", up, down)
	}
}
