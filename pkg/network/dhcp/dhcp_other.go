//go:build !linux

package dhcp

import (
	"context"
	"errors"
	"net"
)

func dhcpRequest(context.Context, string, string, net.IP) (*Lease, error) {
	return nil, errors.New("dhcp: only works on the device")
}
