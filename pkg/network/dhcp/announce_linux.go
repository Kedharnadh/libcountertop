//go:build linux

package dhcp

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// RFC 5227 1.1: ANNOUNCE_NUM 2, ANNOUNCE_INTERVAL 2 s.
const (
	announceCount = 2
	announceEvery = 2 * time.Second
)

func announce(iface string, ip net.IP) error {
	link, err := net.InterfaceByName(iface)
	if err != nil {
		return fmt.Errorf("dhcp: %s: %w", iface, err)
	}

	addr := ip.To4()
	if addr == nil {
		return fmt.Errorf("dhcp: %s is not an IPv4 address", ip)
	}
	if len(link.HardwareAddr) != 6 {
		return fmt.Errorf("dhcp: %s has no ethernet address", iface)
	}

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, 0)
	if err != nil {
		return fmt.Errorf("dhcp: packet socket: %w", err)
	}
	defer unix.Close(fd)

	to := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ARP),
		Ifindex:  link.Index,
		Halen:    6,
	}
	copy(to.Addr[:], broadcast)

	frame := arpAnnounce(link.HardwareAddr, addr)
	for i := range announceCount {
		if err := unix.Sendto(fd, frame, 0, to); err != nil {
			return fmt.Errorf("dhcp: announcing %s: %w", ip, err)
		}
		if i < announceCount-1 {
			time.Sleep(announceEvery)
		}
	}
	return nil
}

func htons(v uint16) uint16 { return v<<8 | v>>8 }
