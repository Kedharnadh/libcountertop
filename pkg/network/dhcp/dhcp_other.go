//go:build !linux

package dhcp

import (
	"context"
	"errors"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// Nothing off the device has an interface to take a lease on, and DHCP needs AF_PACKET, which is
// Linux only.
func dhcpRequest(context.Context, string, ...dhcpv4.Modifier) (*Lease, error) {
	return nil, errors.New("dhcp: only works on the device")
}
