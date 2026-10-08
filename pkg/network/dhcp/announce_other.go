//go:build !linux

package dhcp

import (
	"errors"
	"net"
)

func announce(string, net.IP) error { return errors.New("dhcp: only works on the device") }
