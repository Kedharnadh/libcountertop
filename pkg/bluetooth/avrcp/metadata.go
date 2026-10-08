package avrcp

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	PDUGetCapabilities      = 0x10
	PDUGetElementAttributes = 0x20
	PDUGetPlayStatus        = 0x30
	PDURegisterNotification = 0x31
	PDUContinue             = 0x40
	PDUAbortContinue        = 0x41
	PDUSetAbsoluteVolume    = 0x50
)

const (
	AttrTitle    = 0x01
	AttrArtist   = 0x02
	AttrAlbum    = 0x03
	AttrTrack    = 0x04
	AttrTotal    = 0x05
	AttrGenre    = 0x06
	AttrDuration = 0x07

	// AVRCP 1.6.
	AttrCoverArt = 0x08
)

// A registration is answered INTERIM, then CHANGED once, then must be made again.
const (
	EventPlaybackStatus = 0x01
	EventTrackChanged   = 0x02
	EventTrackEnd       = 0x03
	EventTrackStart     = 0x04
	EventPosition       = 0x05
	EventBattery        = 0x06
	EventSystemStatus   = 0x07
	EventSettings       = 0x08
	EventNowPlaying     = 0x09
	EventAvailPlayers   = 0x0a
	EventAddressPlayer  = 0x0b
	EventUIDs           = 0x0c
	EventVolume         = 0x0d
)

const (
	StatusStopped = 0x00
	StatusPlaying = 0x01
	StatusPaused  = 0x02
	StatusForward = 0x03
	StatusReverse = 0x04
	StatusError   = 0xff
)

func StatusName(s byte) string {
	switch s {
	case StatusStopped:
		return "stopped"
	case StatusPlaying:
		return "playing"
	case StatusPaused:
		return "paused"
	case StatusForward:
		return "seeking forward"
	case StatusReverse:
		return "seeking back"
	case StatusError:
		return "error"
	}
	return fmt.Sprintf("status %#02x", s)
}

const UTF8 = 0x006a

const pduHeader = 4

const companyBytes = 3

type PDU struct {
	ID byte

	// AVRCP's own fragmentation; the rest is requested with PDUContinue.
	Packet byte

	Params []byte
}

func ParsePDU(operands []byte) (PDU, error) {
	if len(operands) < companyBytes+pduHeader {
		return PDU{}, ErrShort
	}

	company := uint32(operands[0])<<16 | uint32(operands[1])<<8 | uint32(operands[2])
	if company != SIG {
		return PDU{}, fmt.Errorf("avrcp: company %#06x is not the sig's", company)
	}

	body := operands[companyBytes:]
	n := int(binary.BigEndian.Uint16(body[2:]))
	if len(body) < pduHeader+n {
		return PDU{}, ErrShort
	}

	return PDU{ID: body[0], Packet: body[1], Params: body[pduHeader : pduHeader+n]}, nil
}

func (p PDU) Frame(code byte) AVC {
	operands := make([]byte, companyBytes+pduHeader+len(p.Params))

	operands[0] = SIG >> 16
	operands[1] = SIG >> 8 & 0xff
	operands[2] = SIG & 0xff
	operands[3] = p.ID
	operands[4] = p.Packet
	binary.BigEndian.PutUint16(operands[5:], uint16(len(p.Params)))
	copy(operands[companyBytes+pduHeader:], p.Params)

	return AVC{Code: code, Subunit: SubunitPanel, Opcode: OpVendorDependent, Operands: operands}
}

type Track struct {
	Title  string
	Artist string
	Album  string
	Genre  string

	Number string
	Total  string

	Duration time.Duration

	Art string
}

func (t Track) Empty() bool {
	return t.Title == "" && t.Artist == "" && t.Album == ""
}

var Basics = []uint32{
	AttrTitle, AttrArtist, AttrAlbum, AttrTrack, AttrTotal, AttrGenre, AttrDuration,
}

// Identifier zero is the current track; no attributes means all of them.
func GetElementAttributes(attrs ...uint32) PDU {
	params := make([]byte, 8, 9+len(attrs)*4)
	params = append(params, byte(len(attrs)))
	for _, a := range attrs {
		params = binary.BigEndian.AppendUint32(params, a)
	}
	return PDU{ID: PDUGetElementAttributes, Params: params}
}

func ParseElementAttributes(params []byte) (Track, error) {
	var t Track

	if len(params) < 1 {
		return t, ErrShort
	}

	count := int(params[0])
	rest := params[1:]

	for range count {
		// Identifier, character set, length.
		const header = 8
		if len(rest) < header {
			break
		}

		id := binary.BigEndian.Uint32(rest)
		n := int(binary.BigEndian.Uint16(rest[6:]))
		if len(rest) < header+n {
			break
		}

		value := string(rest[header : header+n])
		rest = rest[header+n:]

		switch id {
		case AttrTitle:
			t.Title = value
		case AttrArtist:
			t.Artist = value
		case AttrAlbum:
			t.Album = value
		case AttrGenre:
			t.Genre = value
		case AttrTrack:
			t.Number = value
		case AttrTotal:
			t.Total = value
		case AttrDuration:
			t.Duration = parseMillis(value)
		case AttrCoverArt:
			t.Art = value
		}
	}

	return t, nil
}

func parseMillis(s string) time.Duration {
	var ms int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		ms = ms*10 + int64(c-'0')

		if ms > 30*24*60*60*1000 {
			return 0
		}
	}
	return time.Duration(ms) * time.Millisecond
}

type Progress struct {
	Length   time.Duration
	Position time.Duration
	Status   byte
}

const (
	PDURequestContinuing = 0x40
	PDUAbortContinuing   = 0x41
)

func AbortContinuing(pdu byte) PDU {
	return PDU{ID: PDUAbortContinuing, Params: []byte{pdu}}
}

func GetPlayStatus() PDU { return PDU{ID: PDUGetPlayStatus} }

func ParsePlayStatus(params []byte) (Progress, error) {
	if len(params) < 9 {
		return Progress{}, ErrShort
	}

	return Progress{
		Length:   millis(binary.BigEndian.Uint32(params)),
		Position: millis(binary.BigEndian.Uint32(params[4:])),
		Status:   params[8],
	}, nil
}

// 0xffffffff is "not known".
func millis(v uint32) time.Duration {
	if v == 0xffffffff {
		return 0
	}
	return time.Duration(v) * time.Millisecond
}

// The interval applies only to playback position.
func RegisterNotification(event byte, interval uint32) PDU {
	params := append([]byte{event}, make([]byte, 4)...)
	binary.BigEndian.PutUint32(params[1:], interval)
	return PDU{ID: PDURegisterNotification, Params: params}
}

const (
	RefusedBadCommand   = 0x00
	RefusedBadParameter = 0x01
	RefusedBadContent   = 0x02
	RefusedInternal     = 0x03

	// Sent against every outstanding registration when the addressed player changes.
	RefusedPlayerMoved = 0x16
)

type Notification struct {
	Event byte

	Interim bool

	Status   byte
	Position time.Duration
	Volume   byte

	// All ones is no track selected.
	Track uint64

	Player uint16
}

func ParseNotification(params []byte, interim bool) (Notification, error) {
	if len(params) < 1 {
		return Notification{}, ErrShort
	}

	n := Notification{Event: params[0], Interim: interim}
	body := params[1:]

	switch n.Event {
	case EventPlaybackStatus:
		if len(body) < 1 {
			return n, ErrShort
		}
		n.Status = body[0]

	case EventTrackChanged:
		if len(body) < 8 {
			return n, ErrShort
		}
		n.Track = binary.BigEndian.Uint64(body)

	case EventPosition:
		if len(body) < 4 {
			return n, ErrShort
		}
		n.Position = millis(binary.BigEndian.Uint32(body))

	case EventVolume:
		if len(body) < 1 {
			return n, ErrShort
		}
		n.Volume = body[0] & 0x7f

	case EventAddressPlayer:
		// A UID counter follows.
		if len(body) < 2 {
			return n, ErrShort
		}
		n.Player = binary.BigEndian.Uint16(body)
	}

	return n, nil
}

const VolumeMax = 0x7f

func Volume(percent int) byte {
	if percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return VolumeMax
	}
	return byte((percent*VolumeMax + 50) / 100)
}

func Percent(v byte) int { return (int(v&0x7f)*100 + VolumeMax/2) / VolumeMax }

func SetAbsoluteVolume(v byte) PDU {
	return PDU{ID: PDUSetAbsoluteVolume, Params: []byte{v & 0x7f}}
}

// Capability 0x03 is the supported events list.
const CapabilityEvents = 0x03

func GetCapabilities(what byte) PDU {
	return PDU{ID: PDUGetCapabilities, Params: []byte{what}}
}

func ParseCapabilities(params []byte) ([]byte, error) {
	if len(params) < 2 {
		return nil, ErrShort
	}
	if params[0] != CapabilityEvents {
		return nil, fmt.Errorf("avrcp: capability %#02x is not the event list", params[0])
	}

	count := int(params[1])
	if len(params) < 2+count {
		return nil, ErrShort
	}
	return params[2 : 2+count], nil
}
