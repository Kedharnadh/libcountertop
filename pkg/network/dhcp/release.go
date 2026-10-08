package dhcp

import (
	"fmt"
	"net"
	"time"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

// RFC 2131 table 5: DHCPRELEASE sets ciaddr and server identifier, never requested-address.
func releaseMessage(mac net.HardwareAddr, addr, server net.IP) ([]byte, error) {
	if addr.To4() == nil {
		return nil, fmt.Errorf("dhcp: %s is not an IPv4 address", addr)
	}
	if server.To4() == nil {
		return nil, fmt.Errorf("dhcp: no server to release %s to", addr)
	}

	msg, err := dhcpv4.New(
		dhcpv4.WithHwAddr(mac),
		dhcpv4.WithMessageType(dhcpv4.MessageTypeRelease),
		dhcpv4.WithOption(dhcpv4.OptServerIdentifier(server)),
	)
	if err != nil {
		return nil, fmt.Errorf("dhcp: building a release: %w", err)
	}

	msg.ClientIPAddr = addr.To4()

	return msg.ToBytes(), nil
}

const releaseWait = 2 * time.Second

// RFC 2131 4.4.4: a server sends no reply to DHCPRELEASE.
func sendRelease(iface string, lease *Lease) error {
	if lease == nil {
		return nil
	}

	link, err := net.InterfaceByName(iface)
	if err != nil {
		return fmt.Errorf("dhcp: %s: %w", iface, err)
	}

	payload, err := releaseMessage(link.HardwareAddr, lease.Address.IP, lease.Server)
	if err != nil {
		return err
	}

	conn, err := dialRelease(lease.Address.IP, lease.Server)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := conn.SetWriteDeadline(time.Now().Add(releaseWait)); err != nil {
		return fmt.Errorf("dhcp: releasing %s: %w", lease.Address.IP, err)
	}
	if _, err := conn.Write(payload); err != nil {
		return fmt.Errorf("dhcp: releasing %s: %w", lease.Address.IP, err)
	}
	return nil
}

func dialRelease(from, to net.IP) (*net.UDPConn, error) {
	server := &net.UDPAddr{IP: to, Port: serverPort}

	conn, err := net.DialUDP("udp4", &net.UDPAddr{IP: from, Port: clientPort}, server)
	if err == nil {
		return conn, nil
	}

	conn, err = net.DialUDP("udp4", &net.UDPAddr{IP: from}, server)
	if err != nil {
		return nil, fmt.Errorf("dhcp: releasing to %s: %w", to, err)
	}
	return conn, nil
}
