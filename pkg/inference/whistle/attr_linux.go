package whistle

import "syscall"

func childAttr() *syscall.SysProcAttr { return &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL} }
