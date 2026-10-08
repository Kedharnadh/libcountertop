package dhcp

import (
	"log/slog"
	"net"
)

func remembered(said string) net.IP {
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

func keep(addr net.IP, store Store) {
	said := ""
	if addr.To4() != nil {
		said = addr.To4().String()
	}
	if store.Address() == said {
		return
	}
	if err := store.SetAddress(said); err != nil {
		slog.Warn("could not remember the address", "address", said, "err", err)
	}
}
