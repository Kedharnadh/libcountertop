package hci

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	leAdvertisingParams = 0x2006
	leAdvertisingData   = 0x2008
	leAdvertisingEnable = 0x200A

	// advertisingInterval is in units of 0.625 ms: one advertisement a second.
	advertisingInterval = 1600

	advNonconnInd = 0x03
	allChannels   = 0x07
)

// Advertise broadcasts data as a non-connectable LE advertisement until StopAdvertising.
func (r *Radio) Advertise(data []byte) error {
	if len(data) > 31 {
		return errors.New("ble: advertising data exceeds 31 bytes")
	}
	s, err := r.Line()
	if err != nil {
		return err
	}

	params := make([]byte, 15)
	binary.LittleEndian.PutUint16(params[0:], advertisingInterval)
	binary.LittleEndian.PutUint16(params[2:], advertisingInterval)
	params[4] = advNonconnInd
	params[13] = allChannels

	payload := make([]byte, 32)
	payload[0] = byte(len(data))
	copy(payload[1:], data)

	for _, c := range []struct {
		what   string
		opcode uint16
		params []byte
	}{
		{"advertising parameters", leAdvertisingParams, params},
		{"advertising data", leAdvertisingData, payload},
		{"advertising enable", leAdvertisingEnable, []byte{0x01}},
	} {
		if _, err := s.Command(c.opcode, c.params...); err != nil {
			return fmt.Errorf("ble: %s: %w", c.what, err)
		}
	}
	return nil
}

// StopAdvertising ends what Advertise started.
func (r *Radio) StopAdvertising() error {
	s, err := r.Line()
	if err != nil {
		return err
	}
	_, err = s.Command(leAdvertisingEnable, 0x00)
	return err
}
