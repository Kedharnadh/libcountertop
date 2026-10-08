package nativeapi

import (
	"context"
	"net"

	"github.com/ygelfand/go-esphome-device/mdns"

	"github.com/ygelfand/libcountertop/pkg/network/advertise"
)

// The mdns library advertises loopback when given no addresses.
func (a *Server) advertise(ctx context.Context, port int) {
	advertise.Advertise(ctx, "esphome", func(ips []net.IP) (func(), error) {
		adv, err := mdns.Advertise(mdns.Config{
			Name:          a.srv.Info.Name,
			FriendlyName:  a.srv.Info.FriendlyName,
			Port:          port,
			MACAddress:    a.srv.Info.MACAddress,
			Version:       a.srv.Info.Version,
			Platform:      a.id.Platform,
			Board:         a.id.Board,
			Encrypted:     a.srv.PSK != nil && !a.srv.PSK.IsZero(),
			Provisionable: a.srv.PSK != nil && a.srv.PSK.IsZero(),
			IPs:           ips,
		})
		if err != nil {
			return nil, err
		}
		return func() { _ = adv.Close() }, nil
	})
}
