//go:build !linux

package input

import "errors"

func (d *Device) HasKey(uint16) bool { return false }

func (d *Device) Key(uint16) (bool, error) {
	return false, errors.New("input: key state is Linux only")
}
