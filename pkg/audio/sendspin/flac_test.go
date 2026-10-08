package sendspin

import (
	"bytes"
	"testing"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

const flacBlock = 1024

func flacStream(t *testing.T, blocks int) (encoded []byte, want []int16) {
	t.Helper()

	var buf bytes.Buffer
	info := &meta.StreamInfo{
		BlockSizeMin:  flacBlock,
		BlockSizeMax:  flacBlock,
		SampleRate:    Rate,
		NChannels:     Channels,
		BitsPerSample: Bits,
		NSamples:      uint64(flacBlock * blocks),
	}

	enc, err := flac.NewEncoder(&buf, info)
	if err != nil {
		t.Fatalf("flac.NewEncoder: %v", err)
	}

	for b := range blocks {
		subs := make([]*frame.Subframe, Channels)
		for c := range subs {
			samples := make([]int32, flacBlock)
			for i := range samples {
				samples[i] = int32((b*flacBlock+i)%2000 - 1000 + c*3)
				want = append(want, 0)
			}
			subs[c] = &frame.Subframe{
				SubHeader: frame.SubHeader{Pred: frame.PredVerbatim},
				Samples:   samples,
				NSamples:  flacBlock,
			}
		}

		at := b * flacBlock * Channels
		for i := range flacBlock {
			for c := range Channels {
				want[at+i*Channels+c] = int16(subs[c].Samples[i])
			}
		}

		f := &frame.Frame{
			Header: frame.Header{
				HasFixedBlockSize: true,
				BlockSize:         flacBlock,
				SampleRate:        Rate,
				Channels:          frame.ChannelsLR,
				BitsPerSample:     Bits,
				Num:               uint64(b),
			},
			Subframes: subs,
		}
		if err := enc.WriteFrame(f); err != nil {
			t.Fatalf("WriteFrame %d: %v", b, err)
		}
	}
	if err := enc.Close(); err != nil {
		t.Fatalf("closing the encoder: %v", err)
	}
	return buf.Bytes(), want
}

func drain(t *testing.T, d decoder, rest []byte, piece int) []int16 {
	t.Helper()

	var got []int16
	for len(rest) > 0 {
		n := min(piece, len(rest))
		pcm, err := d.decode(rest[:n])
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		got = append(got, pcm...)
		rest = rest[n:]
	}

	for range 200 {
		pcm, err := d.decode(nil)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(pcm) == 0 {
			break
		}
		got = append(got, pcm...)
	}
	return got
}

func TestFLACDecodesWhatWasEncoded(t *testing.T) {
	encoded, want := flacStream(t, 3)

	d, err := newFLACDecoder(encoded[:4])
	if err != nil {
		t.Fatalf("newFLACDecoder: %v", err)
	}
	defer d.close()

	got := drain(t, d, encoded[4:], 512)

	if len(got) != len(want) {
		t.Fatalf("decoded %d samples, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sample %d is %d, want %d", i, got[i], want[i])
		}
	}
}

func TestFLACSaysNothingUntilAFrameIsWhole(t *testing.T) {
	encoded, want := flacStream(t, 1)

	d, err := newFLACDecoder(encoded[:4])
	if err != nil {
		t.Fatalf("newFLACDecoder: %v", err)
	}
	defer d.close()

	got := drain(t, d, encoded[4:], 1)

	if len(got) != len(want) {
		t.Fatalf("decoded %d samples, want %d", len(got), len(want))
	}
}

func TestFLACRefusesToStartWithoutItsHeader(t *testing.T) {
	if _, err := newFLACDecoder(nil); err == nil {
		t.Error("started with no header")
	}
}
