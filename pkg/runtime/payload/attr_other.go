//go:build !linux

package payload

import "syscall"

func DiesWithParent() *syscall.SysProcAttr { return nil }
