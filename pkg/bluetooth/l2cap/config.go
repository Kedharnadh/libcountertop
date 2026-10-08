package l2cap

import (
	"encoding/binary"
	"fmt"
)

const (
	OptionMTU     = 0x01
	OptionFlush   = 0x02
	OptionQoS     = 0x03
	OptionRetrans = 0x04
)

// A hinted option may be skipped; any other must be answered.
const OptionHint = 0x80

type Option struct {
	Type  byte
	Hint  bool
	Value []byte
}

func (o Option) MTU() (uint16, bool) {
	if o.Type != OptionMTU || len(o.Value) < 2 {
		return 0, false
	}
	return binary.LittleEndian.Uint16(o.Value), true
}

func (o Option) Marshal() []byte {
	t := o.Type
	if o.Hint {
		t |= OptionHint
	}

	out := make([]byte, 2+len(o.Value))
	out[0] = t
	out[1] = byte(len(o.Value))
	copy(out[2:], o.Value)
	return out
}

func MTUOption(mtu uint16) Option {
	v := make([]byte, 2)
	binary.LittleEndian.PutUint16(v, mtu)

	return Option{Type: OptionMTU, Value: v}
}

func ParseOptions(buf []byte) ([]Option, error) {
	var out []Option

	for len(buf) > 0 {
		if len(buf) < 2 {
			return nil, fmt.Errorf(
				"l2cap: %d bytes left over, less than an option header",
				len(buf),
			)
		}

		n := int(buf[1])
		if len(buf) < 2+n {
			return nil, fmt.Errorf("l2cap: an option says %d bytes and %d are left", n, len(buf)-2)
		}

		out = append(out, Option{
			Type:  buf[0] &^ OptionHint,
			Hint:  buf[0]&OptionHint != 0,
			Value: buf[2 : 2+n],
		})
		buf = buf[2+n:]
	}
	return out, nil
}

// A response to a continued request must carry the flag back.
const ConfigFlagContinues = 0x0001

type Configure struct {
	DestinationCID uint16
	Flags          uint16
	Options        []Option
}

func (c Configure) Continues() bool { return c.Flags&ConfigFlagContinues != 0 }

// Absent means the 672-byte default.
func (c Configure) MTU() uint16 {
	for _, o := range c.Options {
		if mtu, ok := o.MTU(); ok {
			return mtu
		}
	}
	return MTUDefault
}

func (c Configure) Unknown() []byte {
	var out []byte
	for _, o := range c.Options {
		switch {
		case o.Hint:
		case o.Type == OptionMTU, o.Type == OptionFlush, o.Type == OptionQoS:

		case o.Type == OptionRetrans:

		default:
			out = append(out, o.Type)
		}
	}
	return out
}

const (
	ModeBasic   = 0x00
	ModeRetrans = 0x01
	ModeFlow    = 0x02
	ModeERTM    = 0x03
	ModeStream  = 0x04
)

// A mode, a window, a transmit count, two timers and a size.
const retransFields = 9

// Absent means basic mode.
func (c Configure) Mode() (mode byte, asked bool) {
	for _, o := range c.Options {
		if o.Type == OptionRetrans && len(o.Value) >= 1 {
			return o.Value[0], true
		}
	}
	return ModeBasic, false
}

func BasicMode() Option {
	return Option{Type: OptionRetrans, Value: make([]byte, retransFields)}
}

func ParseConfigure(c Command) (Configure, error) {
	if len(c.Data) < 4 {
		return Configure{}, ErrShort
	}

	options, err := ParseOptions(c.Data[4:])
	if err != nil {
		return Configure{}, err
	}

	return Configure{
		DestinationCID: binary.LittleEndian.Uint16(c.Data),
		Flags:          binary.LittleEndian.Uint16(c.Data[2:]),
		Options:        options,
	}, nil
}

func (c Configure) Command(id byte) Command {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data, c.DestinationCID)
	binary.LittleEndian.PutUint16(data[2:], c.Flags)

	for _, o := range c.Options {
		data = append(data, o.Marshal()...)
	}
	return Command{Code: CodeConfigRequest, ID: id, Data: data}
}

const (
	ConfigSuccess      = 0x0000
	ConfigUnacceptable = 0x0001
	ConfigRejected     = 0x0002
	ConfigUnknown      = 0x0003
	ConfigPending      = 0x0004
)

type Configured struct {
	SourceCID uint16
	Flags     uint16
	Result    uint16
	Options   []Option
}

func (r Configured) Command(id byte) Command {
	data := make([]byte, 6)
	binary.LittleEndian.PutUint16(data, r.SourceCID)
	binary.LittleEndian.PutUint16(data[2:], r.Flags)
	binary.LittleEndian.PutUint16(data[4:], r.Result)

	for _, o := range r.Options {
		data = append(data, o.Marshal()...)
	}
	return Command{Code: CodeConfigResponse, ID: id, Data: data}
}

func ParseConfigured(c Command) (Configured, error) {
	if len(c.Data) < 6 {
		return Configured{}, ErrShort
	}

	options, err := ParseOptions(c.Data[6:])
	if err != nil {
		return Configured{}, err
	}

	return Configured{
		SourceCID: binary.LittleEndian.Uint16(c.Data),
		Flags:     binary.LittleEndian.Uint16(c.Data[2:]),
		Result:    binary.LittleEndian.Uint16(c.Data[4:]),
		Options:   options,
	}, nil
}
