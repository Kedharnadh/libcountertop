package avdtp

import "fmt"

const (
	CatMediaTransport = 0x01
	CatReporting      = 0x02
	CatRecovery       = 0x03
	CatContentProtect = 0x04
	CatHeaderCompress = 0x05
	CatMultiplexing   = 0x06
	CatMediaCodec     = 0x07
	CatDelayReporting = 0x08
)

type Capability struct {
	Category byte
	Data     []byte
}

func (c Capability) Marshal() []byte {
	out := make([]byte, 2+len(c.Data))
	out[0] = c.Category
	out[1] = byte(len(c.Data))
	copy(out[2:], c.Data)
	return out
}

func ParseCapabilities(buf []byte) ([]Capability, error) {
	var out []Capability

	for len(buf) > 0 {
		if len(buf) < 2 {
			return nil, fmt.Errorf("avdtp: %d bytes left over, less than a capability header",
				len(buf))
		}

		n := int(buf[1])
		if len(buf) < 2+n {
			return nil, fmt.Errorf("avdtp: a capability says %d bytes and %d are left",
				n, len(buf)-2)
		}

		out = append(out, Capability{Category: buf[0], Data: buf[2 : 2+n]})
		buf = buf[2+n:]
	}
	return out, nil
}

func MarshalCapabilities(caps []Capability) []byte {
	var out []byte
	for _, c := range caps {
		out = append(out, c.Marshal()...)
	}
	return out
}

func Find(caps []Capability, category byte) (Capability, bool) {
	for _, c := range caps {
		if c.Category == category {
			return c, true
		}
	}
	return Capability{}, false
}

// SBC is mandatory for every A2DP device.
const CodecSBC = 0x00

const (
	Rate16000 = 1 << 7
	Rate32000 = 1 << 6
	Rate44100 = 1 << 5
	Rate48000 = 1 << 4
)

const (
	ChannelMono        = 1 << 3
	ChannelDualChannel = 1 << 2
	ChannelStereo      = 1 << 1
	ChannelJointStereo = 1 << 0
)

const (
	Block4  = 1 << 7
	Block8  = 1 << 6
	Block12 = 1 << 5
	Block16 = 1 << 4
)

const (
	Subbands4 = 1 << 3
	Subbands8 = 1 << 2
)

const (
	AllocationSNR      = 1 << 1
	AllocationLoudness = 1 << 0
)

type SBC struct {
	Rates      byte
	Channels   byte
	Blocks     byte
	Subbands   byte
	Allocation byte

	MinBitpool byte
	MaxBitpool byte
}

// Two packed bytes, then min and max bitpool.
const sbcBytes = 4

func (s SBC) Capability() Capability {
	return Capability{
		Category: CatMediaCodec,
		Data: []byte{
			MediaAudio << 4,
			CodecSBC,
			s.Rates | s.Channels,
			s.Blocks | s.Subbands | s.Allocation,
			s.MinBitpool,
			s.MaxBitpool,
		},
	}
}

func ParseSBC(c Capability) (SBC, error) {
	if c.Category != CatMediaCodec {
		return SBC{}, fmt.Errorf("avdtp: category %#x is not a media codec", c.Category)
	}
	// Media type, codec type, then the codec's own bytes.
	if len(c.Data) < 2+sbcBytes {
		return SBC{}, ErrShort
	}
	if c.Data[1] != CodecSBC {
		return SBC{}, fmt.Errorf("avdtp: codec %#x is not sbc", c.Data[1])
	}

	return SBC{
		Rates:      c.Data[2] & 0xf0,
		Channels:   c.Data[2] & 0x0f,
		Blocks:     c.Data[3] & 0xf0,
		Subbands:   c.Data[3] & 0x0c,
		Allocation: c.Data[3] & 0x03,
		MinBitpool: c.Data[4],
		MaxBitpool: c.Data[5],
	}, nil
}

// The bitpool bounds are the A2DP specification defaults.
func SinkSBC() SBC {
	return SBC{
		Rates:      Rate44100 | Rate48000,
		Channels:   ChannelMono | ChannelDualChannel | ChannelStereo | ChannelJointStereo,
		Blocks:     Block4 | Block8 | Block12 | Block16,
		Subbands:   Subbands4 | Subbands8,
		Allocation: AllocationSNR | AllocationLoudness,
		MinBitpool: 2,
		MaxBitpool: 53,
	}
}

func SinkCapabilities() []Capability {
	return []Capability{
		{Category: CatMediaTransport},
		SinkSBC().Capability(),
	}
}

func one(v byte) bool { return v != 0 && v&(v-1) == 0 }

func (s SBC) Chosen() bool {
	return one(s.Rates) && one(s.Channels) && one(s.Blocks) && one(s.Subbands) && one(s.Allocation)
}

func (s SBC) Within(offer SBC) bool {
	return s.Rates&offer.Rates == s.Rates &&
		s.Channels&offer.Channels == s.Channels &&
		s.Blocks&offer.Blocks == s.Blocks &&
		s.Subbands&offer.Subbands == s.Subbands &&
		s.Allocation&offer.Allocation == s.Allocation &&
		s.MaxBitpool <= offer.MaxBitpool &&
		s.MinBitpool >= offer.MinBitpool
}

func (s SBC) Rate() (int, bool) {
	switch s.Rates {
	case Rate16000:
		return 16000, true
	case Rate32000:
		return 32000, true
	case Rate44100:
		return 44100, true
	case Rate48000:
		return 48000, true
	}
	return 0, false
}
