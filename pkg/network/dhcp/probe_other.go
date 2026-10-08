//go:build !linux

package dhcp

import "net"

func probe(string, net.IP) (bool, error) { return false, nil }
