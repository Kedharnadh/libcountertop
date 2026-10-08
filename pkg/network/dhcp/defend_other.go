//go:build !linux

package dhcp

import (
	"context"
	"net"
)

func watching(context.Context, string, net.IP, func()) error { return nil }
