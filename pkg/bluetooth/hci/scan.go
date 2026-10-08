package hci

import (
	"context"
	"encoding/binary"
	"fmt"
)

const (
	hciLEScanParams = 0x200b
	hciLEScanEnable = 0x200c
)

const (
	eventLEMeta = 0x3e

	leAdvertisingReport = 0x02
)

// 0.625 ms units: 18.75 ms every 200 ms.
const (
	scanInterval = 320
	scanWindow   = 30
)

type Advertisement struct {
	Address     [6]byte
	AddressType byte
	RSSI        int8
	Data        []byte
}

func (a Advertisement) Addr() uint64 {
	var v uint64
	for _, b := range a.Address {
		v = v<<8 | uint64(b)
	}
	return v
}

func Scan(ctx context.Context, s *Session, active bool, found func(Advertisement)) error {
	got, done := s.Reports()
	defer done()

	if err := scanning(s, active, true); err != nil {
		return err
	}

	defer scanning(s, active, false)

	for {
		select {
		case <-ctx.Done():
			return nil
		case g, ok := <-got:
			if !ok {
				return fmt.Errorf("ble: scanning: %w", s.Err())
			}
			reports(g.event.Params, found)
		}
	}
}

func scanning(s *Session, active, on bool) error {
	if on {
		params := make([]byte, 7)
		if active {
			params[0] = 0x01
		}
		binary.LittleEndian.PutUint16(params[1:], scanInterval)
		binary.LittleEndian.PutUint16(params[3:], scanWindow)

		if _, err := s.Command(hciLEScanParams, params...); err != nil {
			return fmt.Errorf("ble: the scan parameters: %w", err)
		}
	}

	enable := byte(0x00)
	if on {
		enable = 0x01
	}
	if _, err := s.Command(hciLEScanEnable, enable, 0x00); err != nil {
		return fmt.Errorf("ble: turning the scan %v: %w", on, err)
	}
	return nil
}

// Event type, address type, address, data length, data, RSSI.
func reports(p []byte, found func(Advertisement)) {
	if len(p) < 2 || p[0] != leAdvertisingReport {
		return
	}

	at := 2
	for range int(p[1]) {
		if at+9 > len(p) {
			return
		}

		length := int(p[at+8])
		end := at + 9 + length
		if end >= len(p) {
			return
		}

		a := Advertisement{AddressType: p[at+1], RSSI: int8(p[end])}

		// HCI carries the address least significant octet first.
		for i := range a.Address {
			a.Address[i] = p[at+7-i]
		}
		a.Data = append([]byte(nil), p[at+9:end]...)
		found(a)

		at = end + 1
	}
}
