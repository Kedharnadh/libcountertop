package sendspin

import (
	"context"
	"net"

	"github.com/libp2p/zeroconf/v2"

	core "github.com/ygelfand/libcountertop/pkg/audio/sendspin"
	network "github.com/ygelfand/libcountertop/pkg/network/advertise"
)

// The path record is required by the Sendspin spec.
const (
	service = core.Service
	domain  = core.Domain
	path    = core.Path
)

// zeroconf falls back to loopback without addresses.
func advertise(ctx context.Context, name string, port int) {
	network.Advertise(ctx, "sendspin", func(ips []net.IP) (func(), error) {
		srv, err := zeroconf.RegisterProxy(
			name, service, domain, port, name, network.Strings(ips),
			[]string{"path=" + path, "name=" + name},
			nil,
		)
		if err != nil {
			return nil, err
		}
		return srv.Shutdown, nil
	})
}
