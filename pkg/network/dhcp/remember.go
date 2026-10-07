package dhcp

import (
	"log/slog"
	"net"
)

func Previous(said string) net.IP {
	if said == "" {
		return nil
	}
	addr := net.ParseIP(said)
	if addr == nil || addr.To4() == nil {
		slog.Warn("ignoring a remembered address that is not IPv4", "address", said)
		return nil
	}
	return addr.To4()
}
func Remember(addr net.IP, read func() string, save func(string) error) {
	said := ""
	if addr.To4() != nil {
		said = addr.To4().String()
	}
	if read() == said {
		return
	}
	if err := save(said); err != nil {
		slog.Warn("could not remember the address", "address", said, "err", err)
	}
}
