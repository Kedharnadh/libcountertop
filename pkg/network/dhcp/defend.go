package dhcp

import (
	"net"
	"sync"
	"time"
)

// RFC 5227 DEFEND_INTERVAL: 10 s.
const defendInterval = 10 * time.Second

type answer int

const (
	ignore answer = iota

	defend

	// RFC 5227 2.4: a second conflict within DEFEND_INTERVAL gives the address up.
	surrender
)

func (a answer) String() string {
	switch a {
	case defend:
		return "defend"
	case surrender:
		return "surrender"
	}
	return "ignore"
}

// RFC 5227: a probe carries a zero sender protocol address.
func conflicting(frame []byte, mine net.HardwareAddr, ours net.IP) bool {
	ours = ours.To4()
	if ours == nil {
		return false
	}

	got, ok := readARP(frame)
	if !ok || got.ours(mine) {
		return false
	}
	return got.senderIP.Equal(ours)
}

type defender struct {
	mu     sync.Mutex
	last   time.Time
	strong bool
}

func (d *defender) decide(now time.Time) answer {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.strong && now.Sub(d.last) < defendInterval {
		d.strong = false
		return surrender
	}

	d.last, d.strong = now, true
	return defend
}

func (d *defender) settled(now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.strong && now.Sub(d.last) >= defendInterval {
		d.strong = false
	}
}
