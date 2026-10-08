//go:build !linux

package dhcp

import (
	"errors"
	"net"
)

func sendDecline(string, net.IP, net.IP, string) error {
	return errors.New("dhcp: declining only works on the device")
}
