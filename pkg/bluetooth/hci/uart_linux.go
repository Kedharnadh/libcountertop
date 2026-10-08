//go:build linux

package hci

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sys/unix"
)

// The chip comes out of reset at 115200.
const (
	slow = unix.B115200
	fast = unix.B3000000
)

type Port struct {
	f   *os.File
	tty bool

	baud uint32
	flow bool

	stream

	asleep atomic.Bool

	wmu sync.Mutex
}

// The chip drops data without RTS/CTS flow control.
func Open(path string) (*Port, error) {
	f, err := os.OpenFile(path, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("ble: opening %s: %w", path, err)
	}

	p := &Port{f: f, tty: true}
	if err := p.configure(slow, true); err != nil {
		f.Close()
		return nil, err
	}
	return p, nil
}

func OpenNode(path string) (*Port, error) {
	// A blocking open of /dev/stpbt never returns: the driver powers the chip on inside it.
	f, err := os.OpenFile(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("ble: opening %s: %w", path, err)
	}
	return &Port{f: f}, nil
}

func (p *Port) configure(baud uint32, flow bool) error {
	t := unix.Termios{
		Cflag: unix.CREAD | unix.CLOCAL | unix.CS8 | baud,
		Iflag: unix.IGNPAR,
	}
	if flow {
		t.Cflag |= unix.CRTSCTS
	}

	t.Cc[unix.VMIN] = 0
	t.Cc[unix.VTIME] = 1

	if err := unix.IoctlSetTermios(int(p.f.Fd()), unix.TCSETS, &t); err != nil {
		return fmt.Errorf("ble: setting the line: %w", err)
	}
	p.baud, p.flow = baud, flow
	return nil
}

func (p *Port) Speed(baud uint32) error { return p.configure(baud, p.flow) }

// Flow control must be off across a rate change.
func (p *Port) Flow(on bool) error { return p.configure(p.baud, on) }

// tcdrain sleeps uninterruptibly while CTS stays low.
func (p *Port) Drain(within time.Duration) error {
	if !p.tty {
		return nil
	}
	deadline := time.Now().Add(within)

	for {
		left, err := unix.IoctlGetInt(int(p.f.Fd()), unix.TIOCOUTQ)
		if err != nil {
			return fmt.Errorf("ble: asking what is left to send: %w", err)
		}
		if left == 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("ble: %d bytes still unsent after %v", left, within)
		}
		time.Sleep(time.Millisecond)
	}
}

func (p *Port) Flush() error {
	p.held = nil
	if !p.tty {
		return nil
	}
	return unix.IoctlSetInt(int(p.f.Fd()), unix.TCFLSH, unix.TCIOFLUSH)
}

func (p *Port) Close() error { return p.f.Close() }

// /dev/stpbt returns zero bytes instead of EAGAIN, which os.File reports as EOF.
func (p *Port) readSome(buf []byte) (int, error) {
	n, err := p.f.Read(buf)
	if !p.tty && n == 0 && errors.Is(err, io.EOF) {
		time.Sleep(time.Millisecond)
		return 0, unix.EAGAIN
	}
	return n, err
}

func (p *Port) Write(b []byte) (int, error) {
	if err := p.rouse(); err != nil {
		return 0, err
	}
	return p.send(b)
}

// With CRTSCTS a write blocks until the chip raises CTS.
func (p *Port) send(b []byte) (int, error) {
	p.wmu.Lock()
	defer p.wmu.Unlock()

	if err := p.f.SetWriteDeadline(time.Now().Add(answers)); err != nil {
		return 0, fmt.Errorf("ble: %w", err)
	}
	return p.f.Write(b)
}

func (p *Port) Send(b []byte) (int, error) { return p.send(b) }

func (p *Port) Wake() error {
	_, err := p.send([]byte{ibsWake})
	return err
}

func (p *Port) ibs() bool {
	if len(p.held) == 0 {
		return false
	}

	switch p.held[0] {
	case ibsSleep:
		p.asleep.Store(true)
	case ibsWakeAck:
		p.asleep.Store(false)
	case ibsWake:
		// The chip waits for the acknowledgement before sending anything else.
		p.asleep.Store(false)
		_, _ = p.send([]byte{ibsWakeAck})
	default:
		return false
	}

	p.held = p.held[1:]
	return true
}

// After 0xfe the firmware ignores everything until woken.
func (p *Port) rouse() error {
	if !p.asleep.Load() {
		return nil
	}

	deadline := time.Now().Add(answers)
	buf := make([]byte, 64)

	var asked time.Time

	for time.Now().Before(deadline) {
		if time.Since(asked) >= wakeAgain {
			if _, err := p.send([]byte{ibsWake}); err != nil {
				return fmt.Errorf("ble: waking the chip: %w", err)
			}
			asked = time.Now()
		}

		p.drain(p.ibs)
		if !p.asleep.Load() {
			return nil
		}

		until := time.Now().Add(wakeAgain)
		if until.After(deadline) {
			until = deadline
		}
		if err := p.f.SetReadDeadline(until); err != nil {
			return fmt.Errorf("ble: %w", err)
		}

		n, err := p.readSome(buf)
		switch {
		case n > 0:
			p.held = append(p.held, buf[:n]...)
		case errors.Is(err, os.ErrDeadlineExceeded), errors.Is(err, unix.EAGAIN), err == nil:
		default:
			return fmt.Errorf("ble: reading: %w", err)
		}
	}
	return fmt.Errorf("ble: the chip did not wake in %v", answers)
}

func (p *Port) Settle(within time.Duration) {
	deadline := time.Now().Add(within)
	buf := make([]byte, 512)

	for time.Now().Before(deadline) {
		if err := p.f.SetReadDeadline(deadline); err != nil {
			return
		}

		n, err := p.readSome(buf)
		if n > 0 {
			p.held = append(p.held, buf[:n]...)
			for p.ibs() {
			}
			continue
		}
		if err != nil && !errors.Is(err, unix.EAGAIN) {
			return
		}
	}
	for p.ibs() {
	}
}

func (p *Port) Ask(cmd []byte, within time.Duration) (event, error) {
	if _, err := p.Write(cmd); err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return event{}, fmt.Errorf(
				"ble: the line would not take %d bytes in %v, so the chip is "+
					"not raising CTS",
				len(cmd),
				answers,
			)
		}
		return event{}, fmt.Errorf("ble: writing: %w", err)
	}
	return p.Read(within)
}

func (p *Port) Raw(within time.Duration) ([]byte, error) {
	deadline := time.Now().Add(within)

	got := p.held
	p.held = nil

	buf := make([]byte, 512)
	for time.Now().Before(deadline) {
		if err := p.f.SetReadDeadline(deadline); err != nil {
			return got, fmt.Errorf("ble: %w", err)
		}

		n, err := p.readSome(buf)
		if n > 0 {
			got = append(got, buf[:n]...)
			continue
		}
		if errors.Is(err, os.ErrDeadlineExceeded) {
			break
		}
		if err != nil && !errors.Is(err, unix.EAGAIN) {
			return got, fmt.Errorf("ble: reading: %w", err)
		}
	}
	return got, nil
}

func (p *Port) Read(within time.Duration) (event, error) { return p.read(within) }

func (p *Port) Asleep() bool { return p.asleep.Load() }

func (p *Port) Sniff(within time.Duration) []string {
	deadline := time.Now().Add(within)
	buf := make([]byte, 512)

	for time.Now().Before(deadline) {
		if err := p.f.SetReadDeadline(deadline); err != nil {
			break
		}

		n, err := p.readSome(buf)
		if n > 0 {
			p.held = append(p.held, buf[:n]...)
		} else if err != nil && !errors.Is(err, unix.EAGAIN) && !errors.Is(err, os.ErrDeadlineExceeded) {
			break
		}

		p.drain(p.ibs)
	}

	var out []string
	for {
		e, ok := p.take()
		if !ok {
			break
		}
		out = append(out, fmt.Sprintf("event %#02x  % x", e.Code, e.Params))
	}

	if len(p.held) > 0 {
		out = append(out, fmt.Sprintf("unparsed: % x", p.held))
	}
	return out
}

func (p *Port) read(within time.Duration) (event, error) {
	var e event
	var ok bool

	if err := p.pump(within, func() bool { e, ok = p.take(); return ok }); err != nil {
		return event{}, err
	}
	return e, nil
}

func (p *Port) readACL(within time.Duration) (aclPacket, error) {
	var a aclPacket
	var ok bool

	if err := p.pump(within, func() bool { a, ok = p.takeACL(); return ok }); err != nil {
		return aclPacket{}, err
	}
	return a, nil
}

func (p *Port) next(within time.Duration) (packet, error) {
	var got packet

	err := p.pump(within, func() bool {
		if e, ok := p.take(); ok {
			got = packet{kind: typeEvent, event: e}
			return true
		}
		if a, ok := p.takeACL(); ok {
			got = packet{kind: typeACL, acl: a}
			return true
		}
		return false
	})
	if err != nil {
		return packet{}, err
	}
	return got, nil
}

func (p *Port) pump(within time.Duration, got func() bool) error {
	deadline := time.Now().Add(within)
	buf := make([]byte, 512)

	for {
		if got() {
			return nil
		}

		if p.drain(p.ibs) {
			continue
		}

		if !time.Now().Before(deadline) {
			return fmt.Errorf("%w in %v", errQuiet, within)
		}

		if err := p.f.SetReadDeadline(deadline); err != nil {
			return fmt.Errorf("ble: %w", err)
		}

		switch n, err := p.readSome(buf); {
		case n > 0:
			p.held = append(p.held, buf[:n]...)
		case errors.Is(err, os.ErrDeadlineExceeded):
			return fmt.Errorf("%w in %v", errQuiet, within)
		case errors.Is(err, unix.EAGAIN), err == nil:
		default:
			return fmt.Errorf("ble: reading: %w", err)
		}
	}
}
