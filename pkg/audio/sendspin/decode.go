package sendspin

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/pion/opus"
)

type decoder interface {
	decode(chunk []byte) ([]int16, error)
	close() error
}

func decoderFor(codec string, sampleRate, channels, bitDepth int, header []byte) (decoder, error) {
	switch strings.ToLower(codec) {
	case "flac":
		return newFLACDecoder(header)
	case "opus":
		return newOpusDecoder(sampleRate, channels)
	case "pcm":
		return newPCMDecoder(bitDepth)
	}
	return nil, fmt.Errorf("sendspin: no decoder for %q", codec)
}

// Opus frames are at most 120 ms: 5760 samples per channel at 48 kHz.
type opusDecoder struct {
	dec      opus.Decoder
	channels int
	out      []int16
}

func newOpusDecoder(sampleRate, channels int) (decoder, error) {
	dec, err := opus.NewDecoderWithOutput(sampleRate, channels)
	if err != nil {
		return nil, fmt.Errorf("sendspin: opus decoder: %w", err)
	}
	return &opusDecoder{dec: dec, channels: channels, out: make([]int16, 5760*channels)}, nil
}

func (d *opusDecoder) decode(chunk []byte) ([]int16, error) {
	n, err := d.dec.DecodeToInt16(chunk, d.out)
	if err != nil {
		return nil, fmt.Errorf("sendspin: opus decode: %w", err)
	}
	return d.out[:n*d.channels], nil
}

func (*opusDecoder) close() error { return nil }

// The wire is little-endian signed; 24-bit is packed three bytes to a sample.
type pcmDecoder struct {
	bytes int
	shift uint
	out   []int16
}

func newPCMDecoder(bitDepth int) (decoder, error) {
	switch bitDepth {
	case 16:
		return &pcmDecoder{bytes: 2}, nil
	case 24:
		return &pcmDecoder{bytes: 3, shift: 8}, nil
	case 32:
		return &pcmDecoder{bytes: 4, shift: 16}, nil
	}
	return nil, fmt.Errorf("sendspin: no pcm decoder for %d-bit", bitDepth)
}

func (d *pcmDecoder) decode(chunk []byte) ([]int16, error) {
	if len(chunk)%d.bytes != 0 {
		return nil, fmt.Errorf(
			"sendspin: %d bytes is not whole %d-byte samples",
			len(chunk),
			d.bytes,
		)
	}

	n := len(chunk) / d.bytes
	if cap(d.out) < n {
		d.out = make([]int16, n)
	}
	d.out = d.out[:n]

	for i := range n {
		at := i * d.bytes
		switch d.bytes {
		case 2:
			d.out[i] = int16(binary.LittleEndian.Uint16(chunk[at:]))
		case 3:
			v := int32(chunk[at]) | int32(chunk[at+1])<<8 | int32(chunk[at+2])<<16
			if v&0x800000 != 0 {
				v |= ^0xFFFFFF
			}
			d.out[i] = int16(v >> d.shift)
		case 4:
			d.out[i] = int16(int32(binary.LittleEndian.Uint32(chunk[at:])) >> d.shift)
		}
	}
	return d.out, nil
}

func (*pcmDecoder) close() error { return nil }

// The spec has the server honor this preference order.
func formats() []audioFormat {
	return []audioFormat{
		{Codec: "flac", Channels: Channels, SampleRate: Rate, BitDepth: Bits},
		{Codec: "opus", Channels: Channels, SampleRate: Rate, BitDepth: Bits},
		{Codec: "pcm", Channels: Channels, SampleRate: Rate, BitDepth: Bits},
	}
}

type audioFormat struct {
	Codec      string
	Channels   int
	SampleRate int
	BitDepth   int
}
