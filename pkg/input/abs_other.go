//go:build !linux

package input

import "errors"

func (d *Device) Abs(uint16) (int32, error) {
	return 0, errors.New("input: absolute axes are Linux only")
}
