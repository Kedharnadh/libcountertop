package l2cap

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	CIDSignalling = 0x0001

	CIDConnectionless = 0x0002

	// Below 0x0040 is reserved.
	CIDDynamic = 0x0040
)

const (
	PSMSDP         = 0x0001
	PSMAVCTP       = 0x0017 // AVRCP
	PSMAVDTP       = 0x0019
	PSMAVCTPBrowse = 0x001b // AVRCP browsing
)

// Length and channel ID.
const Header = 4

// MTUSignalling is the minimum the signalling channel is guaranteed.
const (
	MTUDefault    = 672
	MTUSignalling = 48
)

var ErrShort = errors.New("l2cap: shorter than its header says")

type Frame struct {
	CID     uint16
	Payload []byte
}

// The payload aliases buf.
func ParseFrame(buf []byte) (Frame, error) {
	if len(buf) < Header {
		return Frame{}, ErrShort
	}

	n := int(binary.LittleEndian.Uint16(buf))
	if len(buf) != Header+n {
		return Frame{}, fmt.Errorf("l2cap: %d bytes for a frame saying %d", len(buf)-Header, n)
	}

	return Frame{CID: binary.LittleEndian.Uint16(buf[2:]), Payload: buf[Header:]}, nil
}

func (f Frame) Marshal() []byte {
	out := make([]byte, Header+len(f.Payload))
	binary.LittleEndian.PutUint16(out, uint16(len(f.Payload)))
	binary.LittleEndian.PutUint16(out[2:], f.CID)
	copy(out[Header:], f.Payload)
	return out
}

const (
	CodeReject             = 0x01
	CodeConnectRequest     = 0x02
	CodeConnectResponse    = 0x03
	CodeConfigRequest      = 0x04
	CodeConfigResponse     = 0x05
	CodeDisconnectRequest  = 0x06
	CodeDisconnectResponse = 0x07
	CodeEchoRequest        = 0x08
	CodeEchoResponse       = 0x09
	CodeInfoRequest        = 0x0a
	CodeInfoResponse       = 0x0b
)

// Code, identifier, length.
const commandHeader = 4

// The identifier pairs a response with its request.
type Command struct {
	Code byte
	ID   byte
	Data []byte
}

// Several commands may share a frame.
func ParseCommands(payload []byte) ([]Command, error) {
	var out []Command

	for len(payload) > 0 {
		if len(payload) < commandHeader {
			return nil, fmt.Errorf("l2cap: %d bytes left over, less than a command header",
				len(payload))
		}

		n := int(binary.LittleEndian.Uint16(payload[2:]))
		if len(payload) < commandHeader+n {
			return nil, fmt.Errorf("l2cap: a command says %d bytes and %d are left",
				n, len(payload)-commandHeader)
		}

		out = append(out, Command{
			Code: payload[0],
			ID:   payload[1],
			Data: payload[commandHeader : commandHeader+n],
		})
		payload = payload[commandHeader+n:]
	}

	if len(out) == 0 {
		return nil, errors.New("l2cap: a signalling frame with no commands in it")
	}
	return out, nil
}

func (c Command) Marshal() []byte {
	out := make([]byte, commandHeader+len(c.Data))
	out[0] = c.Code
	out[1] = c.ID
	binary.LittleEndian.PutUint16(out[2:], uint16(len(c.Data)))
	copy(out[commandHeader:], c.Data)
	return out
}

func Signal(commands ...Command) Frame {
	var payload []byte
	for _, c := range commands {
		payload = append(payload, c.Marshal()...)
	}
	return Frame{CID: CIDSignalling, Payload: payload}
}

type Connect struct {
	PSM       uint16
	SourceCID uint16
}

func (r Connect) Command(id byte) Command {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data, r.PSM)
	binary.LittleEndian.PutUint16(data[2:], r.SourceCID)

	return Command{Code: CodeConnectRequest, ID: id, Data: data}
}

func ParseConnect(c Command) (Connect, error) {
	if len(c.Data) < 4 {
		return Connect{}, ErrShort
	}
	return Connect{
		PSM:       binary.LittleEndian.Uint16(c.Data),
		SourceCID: binary.LittleEndian.Uint16(c.Data[2:]),
	}, nil
}

const (
	ConnectSuccess     = 0x0000
	ConnectPending     = 0x0001
	ConnectBadPSM      = 0x0002
	ConnectSecurity    = 0x0003
	ConnectNoResources = 0x0004
)

type Connected struct {
	DestinationCID uint16
	SourceCID      uint16
	Result         uint16
	Status         uint16
}

func (r Connected) Command(id byte) Command {
	data := make([]byte, 8)
	binary.LittleEndian.PutUint16(data, r.DestinationCID)
	binary.LittleEndian.PutUint16(data[2:], r.SourceCID)
	binary.LittleEndian.PutUint16(data[4:], r.Result)
	binary.LittleEndian.PutUint16(data[6:], r.Status)

	return Command{Code: CodeConnectResponse, ID: id, Data: data}
}

func ParseConnected(c Command) (Connected, error) {
	if len(c.Data) < 8 {
		return Connected{}, ErrShort
	}
	return Connected{
		DestinationCID: binary.LittleEndian.Uint16(c.Data),
		SourceCID:      binary.LittleEndian.Uint16(c.Data[2:]),
		Result:         binary.LittleEndian.Uint16(c.Data[4:]),
		Status:         binary.LittleEndian.Uint16(c.Data[6:]),
	}, nil
}

type Disconnect struct {
	DestinationCID uint16
	SourceCID      uint16
}

func ParseDisconnect(c Command) (Disconnect, error) {
	if len(c.Data) < 4 {
		return Disconnect{}, ErrShort
	}
	return Disconnect{
		DestinationCID: binary.LittleEndian.Uint16(c.Data),
		SourceCID:      binary.LittleEndian.Uint16(c.Data[2:]),
	}, nil
}

func (d Disconnect) Command(code, id byte) Command {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data, d.DestinationCID)
	binary.LittleEndian.PutUint16(data[2:], d.SourceCID)

	return Command{Code: code, ID: id, Data: data}
}

const (
	RejectNotUnderstood = 0x0000
	RejectBadMTU        = 0x0001
	RejectBadCID        = 0x0002
)

func Reject(id byte, reason uint16, extra ...uint16) Command {
	data := make([]byte, 2+2*len(extra))
	binary.LittleEndian.PutUint16(data, reason)

	for i, v := range extra {
		binary.LittleEndian.PutUint16(data[2+2*i:], v)
	}
	return Command{Code: CodeReject, ID: id, Data: data}
}
