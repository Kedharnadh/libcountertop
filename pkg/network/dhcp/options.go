package dhcp

import (
	"net"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

type Options struct {
	Interface string
	Hostname  func() string
	Store     Store
	Network   func() string
}

type Store interface {
	Address() string
	SetAddress(address string) error
}

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
func (c *Client) network() string {
	if c.options.Network == nil {
		return ""
	}
	return c.options.Network()
}
func (c *Client) previous() net.IP {
	if c.options.Store == nil {
		return nil
	}
	return remembered(c.options.Store.Address())
}
func (c *Client) remember(ip net.IP) {
	if c.options.Store != nil {
		keep(ip, c.options.Store)
	}
}

func RequestOptions(hostname string, want net.IP) []dhcpv4.Modifier {
	mods := []dhcpv4.Modifier{
		dhcpv4.WithOption(dhcpv4.OptHostName(hostname)),
		dhcpv4.WithRequestedOptions(dhcpv4.OptionNTPServers),
	}
	if want.To4() != nil {
		mods = append(mods, dhcpv4.WithOption(dhcpv4.OptRequestedIPAddress(want)))
	}
	return mods
}

// RFC 2131 4.3.2: a SELECTING request names the offered address; only the discover may name a remembered one.
func selecting(hostname string, want net.IP) (discover, request []dhcpv4.Modifier) {
	return RequestOptions(hostname, want), RequestOptions(hostname, nil)
}
