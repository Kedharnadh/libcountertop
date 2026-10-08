package hci

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
)

// Twelve bits; the top four of the word are flags.
type Handle uint16

const handleMask = 0x0fff

const (
	boundaryStart     = 0x00 // host to controller, first fragment
	boundaryContinue  = 0x01
	boundaryFlushable = 0x02 // controller to host, first fragment
	boundaryComplete  = 0x03 // a whole PDU in one packet, LE only
)

// H4 type, handle and flags, length.
const aclHeader = 5

// Length and channel ID.
const l2capHeader = 4

type aclPacket struct {
	Handle    Handle
	Boundary  byte
	Broadcast byte
	Data      []byte
}

func (p aclPacket) starts() bool {
	return p.Boundary == boundaryStart || p.Boundary == boundaryFlushable ||
		p.Boundary == boundaryComplete
}

func parseACL(buf []byte) (aclPacket, int, error) {
	if len(buf) < aclHeader {
		return aclPacket{}, 0, errShort
	}
	if buf[0] != typeACL {
		return aclPacket{}, 0, fmt.Errorf("ble: packet type %#02x is not acl", buf[0])
	}

	word := binary.LittleEndian.Uint16(buf[1:])
	n := int(binary.LittleEndian.Uint16(buf[3:]))

	if len(buf) < aclHeader+n {
		return aclPacket{}, 0, errShort
	}

	return aclPacket{
		Handle:    Handle(word & handleMask),
		Boundary:  byte(word >> 12 & 0x3),
		Broadcast: byte(word >> 14 & 0x3),
		Data:      buf[aclHeader : aclHeader+n],
	}, aclHeader + n, nil
}

// mtu is the controller's ACL data packet length, payload only.
func aclFrames(h Handle, pdu []byte, mtu int) ([][]byte, error) {
	switch {
	case len(pdu) == 0:
		return nil, fmt.Errorf("ble: nothing to send on handle %#x", h)
	case mtu <= 0:
		return nil, fmt.Errorf("ble: the controller takes %d bytes a packet", mtu)
	case h&^handleMask != 0:
		return nil, fmt.Errorf("ble: %#x is more than a twelve bit handle", h)
	}

	var out [][]byte
	boundary := byte(boundaryStart)

	for len(pdu) > 0 {
		n := min(mtu, len(pdu))

		frame := make([]byte, aclHeader+n)
		frame[0] = typeACL
		binary.LittleEndian.PutUint16(frame[1:], uint16(h)|uint16(boundary)<<12)
		binary.LittleEndian.PutUint16(frame[3:], uint16(n))
		copy(frame[aclHeader:], pdu[:n])

		out = append(out, frame)
		pdu = pdu[n:]
		boundary = boundaryContinue
	}
	return out, nil
}

type reassembler struct {
	parts map[Handle][]byte
}

func newReassembler() *reassembler { return &reassembler{parts: map[Handle][]byte{}} }

func (r *reassembler) add(p aclPacket) ([]byte, error) {
	held, going := r.parts[p.Handle]

	switch {
	case p.starts() && going:
		delete(r.parts, p.Handle)
		return nil, fmt.Errorf("ble: handle %#x started a message with %d bytes of one outstanding",
			p.Handle, len(held))

	case !p.starts() && !going:
		return nil, fmt.Errorf("ble: handle %#x continued a message that never started", p.Handle)
	}

	buf := append(held, p.Data...)

	// A first fragment shorter than the L2CAP header is legal.
	if len(buf) < l2capHeader {
		r.parts[p.Handle] = buf
		return nil, nil
	}

	want := l2capHeader + int(binary.LittleEndian.Uint16(buf))
	switch {
	case len(buf) < want:
		r.parts[p.Handle] = buf
		return nil, nil

	case len(buf) > want:
		delete(r.parts, p.Handle)
		return nil, fmt.Errorf("ble: handle %#x sent %d bytes for a %d byte pdu",
			p.Handle, len(buf), want)
	}

	delete(r.parts, p.Handle)
	return buf, nil
}

func (r *reassembler) forget(h Handle) { delete(r.parts, h) }

func (r *reassembler) pending() int { return len(r.parts) }

// The controller drops writes past its buffer count without saying so.
type credit struct {
	mu   sync.Mutex
	more chan struct{}

	size  int
	total int
	free  int
}

func newCredit(size, total int) (*credit, error) {
	if size <= 0 || total <= 0 {
		return nil, fmt.Errorf("ble: the controller reported %d buffers of %d bytes", total, size)
	}
	return &credit{more: make(chan struct{}, 1), size: size, total: total, free: total}, nil
}

func (c *credit) Size() int { return c.size }

func (c *credit) take(ctx context.Context) error {
	for {
		c.mu.Lock()
		if c.free > 0 {
			c.free--
			c.mu.Unlock()
			return nil
		}
		c.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-c.more:
		}
	}
}

func (c *credit) give(n int) error {
	c.mu.Lock()

	c.free += n
	over := c.free > c.total
	if over {
		c.free = c.total
	}
	c.mu.Unlock()

	select {
	case c.more <- struct{}{}:
	default:
	}

	if over {
		return fmt.Errorf("ble: the controller gave back %d buffers, more than the %d it has",
			n, c.total)
	}
	return nil
}

func (c *credit) available() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.free
}

const (
	hciReadBufferSize     = 0x1005
	eventCompletedPackets = 0x13
)

func parseBufferSize(params []byte) (size, total int, err error) {
	// ACL length, SCO length, ACL count, SCO count.
	const want = 2 + 1 + 2 + 2

	if len(params) < want {
		return 0, 0, fmt.Errorf("ble: %d bytes of buffer sizes, want %d", len(params), want)
	}

	size = int(binary.LittleEndian.Uint16(params[0:]))
	total = int(binary.LittleEndian.Uint16(params[3:]))

	if size == 0 || total == 0 {
		return 0, 0, fmt.Errorf("ble: the controller reported %d buffers of %d bytes", total, size)
	}
	return size, total, nil
}

type completed struct {
	Handle Handle
	Count  int
}

// One event covers several handles.
func parseCompletedPackets(e event) ([]completed, error) {
	if e.Code != eventCompletedPackets {
		return nil, fmt.Errorf("ble: event %#02x is not number of completed packets", e.Code)
	}
	if len(e.Params) < 1 {
		return nil, errShort
	}

	n := int(e.Params[0])
	rest := e.Params[1:]

	if len(rest) < n*4 {
		return nil, fmt.Errorf("ble: %d handles reported and %d bytes to say it in", n, len(rest))
	}

	out := make([]completed, 0, n)
	for i := range n {
		at := rest[i*4:]
		out = append(out, completed{
			Handle: Handle(binary.LittleEndian.Uint16(at) & handleMask),
			Count:  int(binary.LittleEndian.Uint16(at[2:])),
		})
	}
	return out, nil
}
