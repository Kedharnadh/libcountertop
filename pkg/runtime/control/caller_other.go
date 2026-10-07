//go:build !linux

package control

import "net"

// Caller allows host test connections; device access checks use Linux peer credentials.
func Caller(net.Conn) (int, bool) { return -1, true }
