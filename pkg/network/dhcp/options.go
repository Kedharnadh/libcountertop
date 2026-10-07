package dhcp

import (
	"github.com/insomniacslk/dhcp/dhcpv4"
	"net"
)

// Options supplies identity and saved-address policy; the client owns lease acquisition and renewal.
type Options struct {
	Interface string
	Hostname  func() string
	Previous  func() net.IP
	Remember  func(net.IP)
}

// New creates a lease client. An empty interface selects wlan0.
func New(options Options) *Client { return &Client{options: options} }
func (c *Client) iface() string {
	if c.options.Interface != "" {
		return c.options.Interface
	}
	return Interface
}
func (c *Client) hostname() string {
	if c.options.Hostname != nil {
		if s := c.options.Hostname(); s != "" {
			return s
		}
	}
	return "countertop"
}
func (c *Client) previous() net.IP {
	if c.options.Previous != nil {
		return c.options.Previous()
	}
	return nil
}
func (c *Client) remember(ip net.IP) {
	if c.options.Remember != nil {
		c.options.Remember(ip)
	}
}

// RequestOptions carries the hostname, NTP request and optional remembered IPv4 address.
func RequestOptions(hostname string, want net.IP) []dhcpv4.Modifier {
	mods := []dhcpv4.Modifier{dhcpv4.WithOption(dhcpv4.OptHostName(hostname)), dhcpv4.WithRequestedOptions(dhcpv4.OptionNTPServers)}
	if want.To4() != nil {
		mods = append(mods, dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(want)))
	}
	return mods
}
