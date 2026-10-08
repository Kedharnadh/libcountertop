//go:build linux

package dhcp

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

// An ARP frame on ethernet is 42 bytes.
const frameSize = 128

// RFC 5227 2.1: PROBE_WAIT 1 s, PROBE_NUM 3, PROBE_MIN 1 s, PROBE_MAX 2 s.
const (
	probeCount = 3
	probeWait  = time.Second
	probeMin   = time.Second
	probeMax   = 2 * time.Second
)

func probe(iface string, ip net.IP) (conflict bool, err error) {
	link, err := net.InterfaceByName(iface)
	if err != nil {
		return false, fmt.Errorf("dhcp: %s: %w", iface, err)
	}

	addr := ip.To4()
	if addr == nil {
		return false, fmt.Errorf("dhcp: %s is not an IPv4 address", ip)
	}
	if len(link.HardwareAddr) != 6 {
		return false, fmt.Errorf("dhcp: %s has no ethernet address", iface)
	}

	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(unix.ETH_P_ARP)))
	if err != nil {
		return false, fmt.Errorf("dhcp: packet socket: %w", err)
	}
	defer unix.Close(fd)

	bind := &unix.SockaddrLinklayer{Protocol: htons(unix.ETH_P_ARP), Ifindex: link.Index}
	if err := unix.Bind(fd, bind); err != nil {
		return false, fmt.Errorf("dhcp: binding to %s: %w", iface, err)
	}

	to := &unix.SockaddrLinklayer{
		Protocol: htons(unix.ETH_P_ARP),
		Ifindex:  link.Index,
		Halen:    6,
	}
	copy(to.Addr[:], broadcast)

	frame := arpProbe(link.HardwareAddr, addr)

	if listen(fd, link.HardwareAddr, addr, wait(0, probeWait)) {
		return true, nil
	}

	for i := range probeCount {
		if err := unix.Sendto(fd, frame, 0, to); err != nil {
			return false, fmt.Errorf("dhcp: probing for %s: %w", ip, err)
		}

		if listen(fd, link.HardwareAddr, addr, wait(probeMin, probeMax)) {
			slog.Warn(
				"something else already holds the offered address",
				"address",
				ip,
				"probe",
				i+1,
			)
			return true, nil
		}
	}
	return false, nil
}

func listen(fd int, mine net.HardwareAddr, want net.IP, d time.Duration) bool {
	deadline := time.Now().Add(d)
	buf := make([]byte, frameSize)

	for {
		left := time.Until(deadline)
		if left <= 0 {
			return false
		}
		if err := setTimeout(fd, left); err != nil {
			return false
		}

		n, _, err := unix.Recvfrom(fd, buf, 0)
		if err != nil {
			return false
		}
		if taken(buf[:n], mine, want) {
			return true
		}
	}
}

func setTimeout(fd int, d time.Duration) error {
	tv := unix.NsecToTimeval(int64(d))
	return unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv)
}

func wait(low, high time.Duration) time.Duration {
	if high <= low {
		return low
	}
	return low + time.Duration(rand.Int64N(int64(high-low)))
}
