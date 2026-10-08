//go:build !linux

package dhcp

import (
	"context"
	"errors"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

func dhcpRequest(context.Context, string, ...dhcpv4.Modifier) (*Lease, error) {
	return nil, errors.New("dhcp: only works on the device")
}
