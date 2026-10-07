package control

import (
	"net"
	"os"

	"golang.org/x/sys/unix"
)

// Caller permits root, adb shell, and the daemon's own user on a local Unix socket.
func Caller(conn net.Conn) (int, bool) {
	uc, ok := conn.(*net.UnixConn)
	if !ok {
		return -1, false
	}
	raw, err := uc.SyscallConn()
	if err != nil {
		return -1, false
	}
	var cred *unix.Ucred
	var credErr error
	if err := raw.Control(func(fd uintptr) {
		cred, credErr = unix.GetsockoptUcred(int(fd), unix.SOL_SOCKET, unix.SO_PEERCRED)
	}); err != nil || credErr != nil || cred == nil {
		return -1, false
	}
	uid := int(cred.Uid)
	return uid, uid == 0 || uid == 2000 || uid == os.Getuid()
}
