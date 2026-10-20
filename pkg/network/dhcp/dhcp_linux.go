package dhcp

import (
	"context"
	"fmt"
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4/nclient4"
)

// AOSP dropped dhcpcd in O; DHCP lives inside system_server.
func dhcpRequest(ctx context.Context, iface, hostname string, want net.IP) (*Lease, error) {
	c, err := nclient4.New(iface)
	if err != nil {
		return nil, fmt.Errorf("dhcp: %w", err)
	}
	defer c.Close()

	discover, request := selecting(hostname, want)
	offer, err := c.DiscoverOffer(ctx, discover...)
	if err != nil {
		return nil, fmt.Errorf("dhcp: %w", err)
	}
	got, err := c.RequestFromOffer(ctx, offer, request...)
	if err != nil {
		return nil, fmt.Errorf("dhcp: %w", err)
	}
	return leaseFrom(got.ACK)
}
