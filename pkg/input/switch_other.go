//go:build !linux

package input

import "errors"

func (d *Device) HasSwitch(uint16) bool { return false }

func (d *Device) Switch(uint16) (bool, error) {
	return false, errors.New("input: switches are Linux only")
}
