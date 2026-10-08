package avrcp

import "fmt"

const (
	Control        = 0x0
	Status         = 0x1
	Notify         = 0x3
	GeneralInquiry = 0x4
)

const (
	NotImplemented = 0x8
	Accepted       = 0x9
	Rejected       = 0xa
	InTransition   = 0xb
	Stable         = 0xc
	Changed        = 0xd
	Interim        = 0xf
)

const (
	SubunitPanel = 0x09
	SubunitUnit  = 0x1f
)

const (
	OpVendorDependent = 0x00
	OpUnitInfo        = 0x30
	OpSubunitInfo     = 0x31
	OpPassThrough     = 0x7c
)

// Bluetooth SIG company ID: marks a vendor-dependent frame as AVRCP.
const SIG = 0x001958

const avcHeader = 3

type AVC struct {
	Code byte

	Subunit   byte
	SubunitID byte
	Opcode    byte
	Operands  []byte
}

func ParseAVC(buf []byte) (AVC, error) {
	if len(buf) < avcHeader {
		return AVC{}, ErrShort
	}

	return AVC{
		Code:      buf[0] & 0x0f,
		Subunit:   buf[1] >> 3,
		SubunitID: buf[1] & 0x07,
		Opcode:    buf[2],
		Operands:  buf[avcHeader:],
	}, nil
}

func (a AVC) Marshal() []byte {
	out := make([]byte, avcHeader+len(a.Operands))

	out[0] = a.Code & 0x0f
	out[1] = a.Subunit<<3 | a.SubunitID&0x07
	out[2] = a.Opcode
	copy(out[avcHeader:], a.Operands)

	return out
}

const (
	OpSelect      = 0x00
	OpVolumeUp    = 0x41
	OpVolumeDown  = 0x42
	OpMute        = 0x43
	OpPlay        = 0x44
	OpStop        = 0x45
	OpPause       = 0x46
	OpRewind      = 0x48
	OpFastForward = 0x49
	OpNext        = 0x4b
	OpPrevious    = 0x4c
)

// A press without a release holds the button down at the target.
const (
	Pressed  = false
	Released = true
)

func PassThrough(op byte, released bool) AVC {
	state := op & 0x7f
	if released {
		state |= 0x80
	}

	// The operation data length operand is mandatory.
	return AVC{
		Code:     Control,
		Subunit:  SubunitPanel,
		Opcode:   OpPassThrough,
		Operands: []byte{state, 0x00},
	}
}

func Button(a AVC) (op byte, released bool, err error) {
	if a.Opcode != OpPassThrough {
		return 0, false, fmt.Errorf("avrcp: opcode %#02x is not a passthrough", a.Opcode)
	}
	if len(a.Operands) < 1 {
		return 0, false, ErrShort
	}
	return a.Operands[0] & 0x7f, a.Operands[0]&0x80 != 0, nil
}

func ButtonName(op byte) string {
	switch op {
	case OpSelect:
		return "select"
	case OpVolumeUp:
		return "volume up"
	case OpVolumeDown:
		return "volume down"
	case OpMute:
		return "mute"
	case OpPlay:
		return "play"
	case OpStop:
		return "stop"
	case OpPause:
		return "pause"
	case OpRewind:
		return "rewind"
	case OpFastForward:
		return "fast forward"
	case OpNext:
		return "next"
	case OpPrevious:
		return "previous"
	}
	return fmt.Sprintf("button %#02x", op)
}
