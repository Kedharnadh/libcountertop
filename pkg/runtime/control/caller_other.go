//go:build !linux

package control

import "net"

func Caller(net.Conn) (int, bool) { return -1, true }
