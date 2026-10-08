package address

import (
	"context"
	"net"
	"time"
)

func Settled(ctx context.Context, within time.Duration) []net.IP {
	return settled(ctx, within, Addresses)
}

func settled(ctx context.Context, within time.Duration, read func() []net.IP) []net.IP {
	was := read()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(within):
		}

		now := read()
		if len(now) > 0 && AddressKey(now) == AddressKey(was) {
			return now
		}
		was = now
	}
}
