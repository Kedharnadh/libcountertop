//go:build !linux

package whistle

import "syscall"

func childAttr() *syscall.SysProcAttr { return nil }
