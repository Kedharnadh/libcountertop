package peer

import (
	"errors"
	"net"
	"slices"
	"strconv"
	"time"
)

var ErrNoRoute = errors.New("peer has no IPv4 address")

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

func (p Peer) Syncs() bool { return slices.Contains(p.Caps, "sync") }

func (p Peer) Addr() (string, error) {
	for _, a := range p.Addrs {
		if a.To4() != nil {
			return net.JoinHostPort(a.String(), strconv.Itoa(p.Port)), nil
		}
	}
	return "", ErrNoRoute
}
