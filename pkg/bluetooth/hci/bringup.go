package hci

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const answers = 2 * time.Second

// From the kernel's qca_set_baudrate.
const switched = 300 * time.Millisecond

type Version struct {
	ProductID uint32
	Patch     uint16
	ROM       uint16
	SOC       uint32

	Raw []byte
}

func (v Version) String() string {
	return fmt.Sprintf("product %#08x patch %#04x rom %#04x soc %#08x  [% x]",
		v.ProductID, v.Patch, v.ROM, v.SOC, v.Raw)
}

func BringUp(options BringUpOptions) (*Port, Version, error) {
	var none Version

	if options.Node != "" {
		p, err := OpenNode(options.Node)
		if err != nil {
			return nil, none, err
		}
		if err := Reset(p); err != nil {
			_ = p.Close()
			return nil, none, err
		}
		return p, none, nil
	}

	changed, err := options.Power(true)
	if err != nil {
		return nil, none, err
	}
	slog.Info("bluetooth radio", "powered", true, "changed", changed)

	p, err := Open(options.UART)
	if err != nil {
		return nil, none, err
	}

	v, err := start(p, options)
	if err != nil {
		_ = p.Close()
		return nil, none, err
	}
	return p, v, nil
}

func start(p *Port, options BringUpOptions) (Version, error) {
	var none Version

	if err := p.Flush(); err != nil {
		return none, err
	}

	v, err := Ask(p)
	if err != nil {
		return none, fmt.Errorf("reading the version: %w", err)
	}
	slog.Info("bluetooth chip", "version", v)

	if err := Fast(p); err != nil {
		return none, err
	}

	if _, err := Ask(p); err != nil {
		return none, fmt.Errorf("the chip stopped answering after the rate change: %w", err)
	}
	slog.Debug("bluetooth line", "rate", "3M")

	for _, f := range []struct {
		what string
		file string
	}{
		{"patch", options.PatchFile},
		{"nvm", options.NVMFile},
	} {
		raw, err := os.ReadFile(filepath.Join(options.FirmwareDirectory, f.file))
		if err != nil {
			return none, fmt.Errorf("reading the %s: %w", f.what, err)
		}

		b, err := ReadBlob(raw)
		if err != nil {
			return none, fmt.Errorf("the %s: %w", f.what, err)
		}
		if b.Patch != nil {
			slog.Debug("bluetooth patch", "says", b.Patch.String())
		}

		if err := Send(p, b); err != nil {
			return none, fmt.Errorf("sending the %s: %w", f.what, err)
		}
		slog.Info("bluetooth firmware", "sent", f.what, "segments", len(b.Segments()))
	}

	if err := Reset(p); err != nil {
		return none, err
	}

	if options.ListenAfterReset {
		return v, nil
	}

	local, err := ReadLocal(p)
	if err != nil {
		return none, fmt.Errorf("the controller did not come back after the reset: %w", err)
	}
	slog.Info("bluetooth controller", "says", local)

	if can, err := ReadFeatures(p); err == nil {
		slog.Debug("bluetooth controller", "can", can)
	}

	return v, nil
}

type Local struct {
	HCIVersion    byte
	HCIRevision   uint16
	LMPVersion    byte
	Manufacturer  uint16
	LMPSubversion uint16
}

func (l Local) String() string {
	return fmt.Sprintf("hci %d rev %d, lmp %d subversion %#04x, manufacturer %d",
		l.HCIVersion, l.HCIRevision, l.LMPVersion, l.LMPSubversion, l.Manufacturer)
}

func ReadLocal(p *Port) (Local, error) {
	cmd, err := command(hciReadLocalVersion)
	if err != nil {
		return Local{}, err
	}
	if _, err := p.Write(cmd); err != nil {
		return Local{}, fmt.Errorf("ble: writing: %w", err)
	}

	said, err := waitReturn(p, hciReadLocalVersion, answers)
	if err != nil {
		return Local{}, err
	}
	if len(said) < 8 {
		return Local{}, fmt.Errorf("ble: a local version of %d bytes", len(said))
	}

	return Local{
		HCIVersion:    said[0],
		HCIRevision:   binary.LittleEndian.Uint16(said[1:]),
		LMPVersion:    said[3],
		Manufacturer:  binary.LittleEndian.Uint16(said[4:]),
		LMPSubversion: binary.LittleEndian.Uint16(said[6:]),
	}, nil
}

type Features [8]byte

func (f Features) Classic() bool { return f[4]&0x20 == 0 }

func (f Features) LE() bool { return f[4]&0x40 != 0 }

func (f Features) EDR() bool { return f[3]&0x80 != 0 }

func (f Features) String() string {
	return fmt.Sprintf("classic %v, le %v, edr %v  [% x]", f.Classic(), f.LE(), f.EDR(), f[:])
}

func ReadFeatures(p *Port) (Features, error) {
	var none Features

	cmd, err := command(hciReadLocalFeatures)
	if err != nil {
		return none, err
	}
	if _, err := p.Write(cmd); err != nil {
		return none, fmt.Errorf("ble: writing: %w", err)
	}

	said, err := waitReturn(p, hciReadLocalFeatures, answers)
	if err != nil {
		return none, err
	}
	if len(said) < 8 {
		return none, fmt.Errorf("ble: %d feature bytes, want 8", len(said))
	}
	return Features(said[:8]), nil
}

func Reset(p *Port) error {
	cmd, err := command(hciReset)
	if err != nil {
		return err
	}
	if _, err := p.Write(cmd); err != nil {
		return fmt.Errorf("resetting: %w", err)
	}
	if _, err := waitReturn(p, hciReset, answers); err != nil {
		return fmt.Errorf("resetting: %w", err)
	}

	// The patched firmware announces itself as soon as it runs.
	p.Settle(switched)
	return nil
}

func waitReturn(p *Port, opcode uint16, within time.Duration) ([]byte, error) {
	deadline := time.Now().Add(within)

	var saw []string
	for {
		left := time.Until(deadline)
		if left <= 0 {
			if len(saw) == 0 {
				return nil, fmt.Errorf("ble: %#04x was not answered in %v", opcode, within)
			}
			return nil, fmt.Errorf("ble: %#04x was not answered in %v, but the chip said %s",
				opcode, within, strings.Join(saw, ", "))
		}

		e, err := p.Read(left)
		if err != nil {
			return nil, err
		}
		if e.Code == eventCommandComplete && len(e.Params) >= 3 &&
			binary.LittleEndian.Uint16(e.Params[1:]) == opcode {
			return complete(e, opcode)
		}
		saw = append(saw, fmt.Sprintf("%#02x [% x]", e.Code, e.Params))
	}
}

func Ask(p *Port) (Version, error) {
	cmd, err := version()
	if err != nil {
		return Version{}, err
	}

	if _, err := p.Write(cmd); err != nil {
		return Version{}, fmt.Errorf("ble: writing: %w", err)
	}

	said, err := waitFor(p, edlVersionResult, answers)
	if err != nil {
		return Version{}, err
	}
	if len(said) < 12 {
		return Version{}, fmt.Errorf("ble: a version of %d bytes", len(said))
	}

	return Version{
		ProductID: binary.LittleEndian.Uint32(said[0:]),
		Patch:     binary.LittleEndian.Uint16(said[4:]),
		ROM:       binary.LittleEndian.Uint16(said[6:]),
		SOC:       binary.LittleEndian.Uint32(said[8:]),
		Raw:       said,
	}, nil
}

// The chip sends a command complete before the vendor event, and events of its own.
func waitFor(p *Port, want byte, within time.Duration) ([]byte, error) {
	deadline := time.Now().Add(within)

	for {
		left := time.Until(deadline)
		if left <= 0 {
			return nil, fmt.Errorf("ble: no answer to %#02x in %v", want, within)
		}

		e, err := p.Read(left)
		if err != nil {
			return nil, err
		}
		if e.Code != eventVendor || len(e.Params) < 2 || e.Params[1] != want {
			continue
		}
		return answered(e, want)
	}
}

// The chip does not answer the baud rate command.
func Fast(p *Port) error {
	cmd, err := command(baudOpcode, baud3M)
	if err != nil {
		return err
	}

	if err := p.Flow(false); err != nil {
		return err
	}
	if _, err := p.Write(cmd); err != nil {
		return fmt.Errorf("asking for the higher rate: %w", err)
	}
	if err := p.Drain(answers); err != nil {
		return err
	}
	time.Sleep(switched)

	if err := p.Speed(fast); err != nil {
		return err
	}
	return p.Flow(true)
}

func Send(p *Port, b Blob) error {
	segments := b.Segments()
	acked := b.Acked()

	for i, piece := range segments {
		cmd, err := download(piece)
		if err != nil {
			return err
		}

		if _, err := p.Write(cmd); err != nil {
			return fmt.Errorf("segment %d of %d: %w", i+1, len(segments), err)
		}

		// Download mode 0x03 suppresses the vendor event and the completion for every segment.
		if !acked {
			continue
		}
		if err := taken(p); err != nil {
			return fmt.Errorf("segment %d of %d: %w", i+1, len(segments), err)
		}
	}

	if !acked {
		quiet(p)
	}
	return nil
}

var trailing = 300 * time.Millisecond

func quiet(p *Port) {
	for {
		e, err := p.Read(trailing)
		if err != nil {
			return
		}
		slog.Debug("bluetooth after a silent download",
			"event", fmt.Sprintf("%#02x", e.Code), "params", fmt.Sprintf("% x", e.Params))
	}
}

type BringUpOptions struct {
	Node               string
	UART               string
	FirmwareDirectory  string
	PatchFile, NVMFile string
	Power              func(bool) (bool, error)
	ListenAfterReset   bool
}
