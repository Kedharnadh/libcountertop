package hci

import (
	"encoding/binary"
	"fmt"
)

const (
	edlOpcode = 0xfc00

	edlVersionReq = 0x19
	edlTLVReq     = 0x1e
)

// The baud rate change is a separate vendor command.
const (
	baudOpcode = 0xfc48
	baud3M     = 0x0e
)

// A vendor event names the request it answers.
const (
	edlRequestResult = 0x00

	// The version request is 0x19 and its reply 0x02.
	edlVersionResult = 0x02

	edlDownloadDone = 0x04
)

const (
	tlvPatch = 1
	tlvNVM   = 2
)

const segment = 243

type Blob struct {
	Type byte
	Raw  []byte

	Patch *Patch
}

type Patch struct {
	TotalSize  uint32
	DataLength uint32
	Format     byte
	Signature  byte

	Download byte

	ProductID uint16
	ROMBuild  uint16
	Version   uint16
	Entry     uint32
}

const (
	sayEverything = 0x00
	saySilent     = 0x03
)

func (p Patch) String() string {
	return fmt.Sprintf("product %#04x rom %d patch %d entry %#08x, %d bytes, download %#02x",
		p.ProductID, p.ROMBuild, p.Version, p.Entry, p.DataLength, p.Download)
}

// The header is a type in the low byte and a length in the three above it.
func ReadBlob(raw []byte) (Blob, error) {
	const header = 4

	if len(raw) < header {
		return Blob{}, fmt.Errorf("ble: %d bytes is too short to be a TLV", len(raw))
	}

	at := binary.LittleEndian.Uint32(raw)
	b := Blob{Type: byte(at & 0xff), Raw: raw}

	if want := int(at>>8) + header; len(raw) < want {
		return Blob{}, fmt.Errorf("ble: the TLV says %d bytes and the file is %d", want, len(raw))
	}

	switch b.Type {
	case tlvNVM:
		return b, nil
	case tlvPatch:
	default:
		return Blob{}, fmt.Errorf("ble: TLV type %d is neither a patch nor an NVM", b.Type)
	}

	// The patch header follows the TLV header.
	const patchHeader = 28
	if len(raw) < header+patchHeader {
		return Blob{}, fmt.Errorf("ble: a patch with no header in %d bytes", len(raw))
	}

	in := raw[header:]
	b.Patch = &Patch{
		TotalSize:  binary.LittleEndian.Uint32(in[0:]),
		DataLength: binary.LittleEndian.Uint32(in[4:]),
		Format:     in[8],
		Signature:  in[9],
		Download:   in[10],
		ProductID:  binary.LittleEndian.Uint16(in[12:]),
		ROMBuild:   binary.LittleEndian.Uint16(in[14:]),
		Version:    binary.LittleEndian.Uint16(in[16:]),
		Entry:      binary.LittleEndian.Uint32(in[20:]),
	}
	return b, nil
}

// Download 0x03 is silent; an NVM is always acknowledged.
func (b Blob) Acked() bool {
	return b.Patch == nil || b.Patch.Download == sayEverything
}

func (b Blob) Segments() [][]byte {
	out := make([][]byte, 0, len(b.Raw)/segment+1)

	for at := 0; at < len(b.Raw); at += segment {
		out = append(out, b.Raw[at:min(at+segment, len(b.Raw))])
	}
	return out
}

func download(piece []byte) ([]byte, error) {
	if len(piece) > segment {
		return nil, fmt.Errorf("ble: a %d byte segment, more than the %d one command holds",
			len(piece), segment)
	}
	return command(edlOpcode, append([]byte{edlTLVReq, byte(len(piece))}, piece...)...)
}

func version() ([]byte, error) { return command(edlOpcode, edlVersionReq) }

// The vendor downloader logs these and continues.
var refusals = map[byte]string{
	1: "the patch length is wrong",
	2: "the patch version is wrong",
	3: "the patch failed its CRC",
	4: "the patch data is not valid",
	5: "the TLV type is wrong",
}

func refusal(status byte) string {
	if why, ok := refusals[status]; ok {
		return why
	}
	return fmt.Sprintf("status %#02x", status)
}

// Acknowledged by a vendor event and a zero-opcode command complete, in either order.
func taken(p *Port) error {
	var said, done bool

	for range 2 {
		e, err := p.Read(answers)
		if err != nil {
			return err
		}

		switch e.Code {
		case eventVendor:
			status, err := answered(e, edlDownloadDone)
			if err != nil {
				return err
			}
			if len(status) > 0 && status[0] != 0 {
				return fmt.Errorf("ble: the chip refused the segment: %s", refusal(status[0]))
			}
			said = true
		case eventCommandComplete:
			if len(e.Params) >= 4 && e.Params[3] != 0 {
				return fmt.Errorf("ble: the segment was refused, status %#02x", e.Params[3])
			}
			done = true
		}
	}

	if !said || !done {
		return fmt.Errorf("ble: the segment was answered twice but not with both halves")
	}
	return nil
}

func answered(e event, want byte) ([]byte, error) {
	if e.Code != eventVendor {
		return nil, fmt.Errorf("ble: event %#02x, want a vendor event", e.Code)
	}
	if len(e.Params) < 2 {
		return nil, errShort
	}

	if got := e.Params[0]; got != edlRequestResult {
		return nil, fmt.Errorf("ble: the chip reports %#02x rather than a result", got)
	}
	if got := e.Params[1]; got != want {
		return nil, fmt.Errorf("ble: an answer to %#02x while waiting on %#02x", got, want)
	}
	return e.Params[2:], nil
}
