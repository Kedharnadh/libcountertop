package dhcp

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"
)

const timeout = 5 * time.Second

var (
	mu      sync.Mutex
	servers []string
	install sync.Once
)

// There is no /etc/resolv.conf on the device; bionic resolves through netd.
func useDNS(ips []net.IP) {
	addrs := make([]string, 0, len(ips))
	for _, ip := range ips {
		addrs = append(addrs, ip.String())
	}

	mu.Lock()
	servers = addrs
	mu.Unlock()

	install.Do(func() { net.DefaultResolver = &net.Resolver{PreferGo: true, Dial: dial} })
	slog.Info("resolver configured", "nameservers", joinIPs(ips))
}

func dial(ctx context.Context, network, _ string) (net.Conn, error) {
	mu.Lock()
	addrs := servers
	mu.Unlock()

	if len(addrs) == 0 {
		return nil, errors.New("dhcp: the lease named no nameservers")
	}

	var err error
	for _, server := range addrs {
		d := net.Dialer{Timeout: timeout}

		var conn net.Conn
		if conn, err = d.DialContext(ctx, network, net.JoinHostPort(server, "53")); err == nil {
			return conn, nil
		}
	}
	return nil, err
}
