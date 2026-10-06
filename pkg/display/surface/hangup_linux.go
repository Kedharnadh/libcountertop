package surface

import "golang.org/x/sys/unix"

func (c *Client) hungUp() bool {
	raw, err := c.c.SyscallConn()
	if err != nil {
		return false
	}
	gone := false
	raw.Control(func(fd uintptr) {
		p := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLRDHUP}}
		if n, _ := unix.Poll(p, 0); n > 0 && p[0].Revents&(unix.POLLRDHUP|unix.POLLHUP|unix.POLLERR) != 0 {
			gone = true
		}
	})
	return gone
}
