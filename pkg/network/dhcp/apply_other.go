//go:build !linux

package dhcp

import "errors"

func apply(string, *Lease) error { return errors.New("dhcp: only works on the device") }

func release(string) {}
