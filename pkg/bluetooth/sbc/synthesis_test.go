package sbc

import (
	"encoding/binary"
	"os"
	"testing"
)

// Vectors are ffmpeg's SBC encoder and decoder output, except joint, which a phone sent.

func reference(t *testing.T, name string) (coded []byte, want []int16) {
	t.Helper()

	coded, err := os.ReadFile("testdata/vectors/" + name + ".sbc")
	if err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile("testdata/vectors/" + name + ".pcm")
	if err != nil {
		t.Fatal(err)
	}

	want = make([]int16, len(raw)/2)
	for i := range want {
		want[i] = int16(binary.LittleEndian.Uint16(raw[2*i:]))
	}
	return coded, want
}

func TestDecodingMatchesTheReference(t *testing.T) {
	for _, name := range []string{"tone", "mono", "sweep", "short", "joint"} {
		t.Run(name, func(t *testing.T) { matches(t, name) })
	}
}

func matches(t *testing.T, name string) {
	t.Helper()
	coded, want := reference(t, name)

	var f Filter
	var got []int16
	frames := 0

	for at := 0; at < len(coded); {
		start, ok := Find(coded[at:])
		if !ok {
			break
		}
		at += start

		fr, err := Unpack(coded[at:])
		if err != nil {
			t.Fatalf("frame %d at %d: %v", frames, at, err)
		}

		out, err := f.Synthesize(fr)
		if err != nil {
			t.Fatalf("frame %d: %v", frames, err)
		}

		for i := range out[0] {
			for ch := range out {
				got = append(got, out[ch][i])
			}
		}

		at += fr.Header.Length()
		frames++
	}

	if frames == 0 {
		t.Fatal("no frames were found in the vector")
	}
	t.Logf("%d frames, %d samples", frames, len(got))

	if len(got) != len(want) {
		t.Fatalf("decoded %d samples, the reference has %d", len(got), len(want))
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d is %d, the reference says %d", i, got[i], want[i])
		}
	}
}

func TestTheFilterbankCarriesHistoryBetweenFrames(t *testing.T) {
	coded, _ := reference(t, "tone")

	carried := decoded(t, coded, false)
	restarted := decoded(t, coded, true)

	if len(carried) != len(restarted) {
		t.Fatalf("%d samples against %d", len(carried), len(restarted))
	}

	same := 0
	for i := range carried {
		if carried[i] == restarted[i] {
			same++
		}
	}

	if same == len(carried) {
		t.Fatal("restarting the filterbank every frame changed nothing, so it carries no history")
	}
	t.Logf("%d of %d samples differ when the filterbank is restarted per frame",
		len(carried)-same, len(carried))
}

func decoded(t *testing.T, coded []byte, fresh bool) []int16 {
	t.Helper()

	var f Filter
	var out []int16

	for at := 0; at < len(coded); {
		start, ok := Find(coded[at:])
		if !ok {
			break
		}
		at += start

		fr, err := Unpack(coded[at:])
		if err != nil {
			break
		}

		if fresh {
			f = Filter{}
		}

		audio, err := f.Synthesize(fr)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, audio[0]...)

		at += fr.Header.Length()
	}
	return out
}
