package wifi

import (
	"context"

	"github.com/ygelfand/libcountertop/pkg/network/wpa"
)

func (r *Radio) Scan(ctx context.Context) ([]wpa.Network, error) { return r.networks.Scan(ctx) }

func (r *Radio) Networks() ([]string, error) { return r.networks.Networks() }

func (r *Radio) Join(ctx context.Context, ssid, passphrase string) error {
	return r.networks.Join(ctx, ssid, passphrase)
}

func (r *Radio) Forget(ssid string) error { return r.networks.Forget(ssid) }
