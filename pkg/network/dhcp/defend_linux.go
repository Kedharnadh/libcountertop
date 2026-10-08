//go:build linux

package dhcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

const watchPoll = time.Second

func watching(ctx context.Context, iface string, ip net.IP, lost func()) error {
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

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_ARP)))
	if err != nil {
		return fmt.Errorf("dhcp: packet socket: %w", err)
	}
	defer unix.Close(fd)

	if err := unix.Bind(fd, &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ARP),
		Ifindex:  link.Index,
	}); err != nil {
		return fmt.Errorf("dhcp: binding to %s: %w", iface, err)
	}

	to := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ARP),
		Ifindex:  link.Index,
		Halen:    6,
	}
	copy(to.Addr[:], broadcast)

	frame := arpAnnounce(link.HardwareAddr, addr)
	buf := make([]byte, frameSize)

	var d defender
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		if err := setTimeout(fd, watchPoll); err != nil {
			return fmt.Errorf("dhcp: watching %s: %w", ip, err)
		}

		n, _, err := unix.Recvfrom(fd, buf, 0)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) || errors.Is(err, unix.EWOULDBLOCK) ||
				errors.Is(err, unix.EINTR) {
				continue
			}
			return fmt.Errorf("dhcp: watching %s: %w", ip, err)
		}

		now := time.Now()
		d.settled(now)

		if !conflicting(buf[:n], link.HardwareAddr, addr) {
			continue
		}

		switch got := d.decide(now); got {
		case defend:
			slog.Warn("something else is claiming our address, defending it", "address", ip)
			if err := unix.Sendto(fd, frame, 0, to); err != nil {
				slog.Warn("could not defend the address", "address", ip, "err", err)
			}
		case surrender:
			slog.Warn("giving up an address something else insists on", "address", ip)
			lost()
			return nil
		}
	}
}
