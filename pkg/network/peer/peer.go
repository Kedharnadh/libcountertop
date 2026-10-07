package peer

import (
	"net"
	"slices"
	"time"
)

type Peer struct {
	ID      string
	Name    string
	Model   string
	Board   string
	Project string
	Version string
	Caps    []string
	Host    string
	Addrs   []net.IP
	Port    int
	Seen    time.Time
}

func (p Peer) Video() bool { return slices.Contains(p.Caps, "video") }
