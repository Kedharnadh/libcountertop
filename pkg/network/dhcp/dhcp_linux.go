package dhcp

import (
	"context"
	"fmt"

	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
)

// AOSP dropped dhcpcd in O; DHCP lives inside system_server.
func dhcpRequest(ctx context.Context, iface string, options ...dhcpv4.Modifier) (*Lease, error) {
	c, err := nclient4.New(iface)
	if err != nil {
		return nil, fmt.Errorf("dhcp: %w", err)
	}
	defer c.Close()

	got, err := c.Request(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("dhcp: %w", err)
	}
	return leaseFrom(got.ACK)
}
