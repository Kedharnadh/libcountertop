package cast

import (
	"fmt"
	"slices"
	"sync"

	"github.com/libp2p/zeroconf/v2"
)

type Publisher interface {
	SetText(text []string)
	Shutdown()
}

type Advertiser struct {
	mu sync.Mutex

	on     Publisher
	device Device
	last   []string
	port   int
}

func NewAdvertiser(d Device, port int) (*Advertiser, error) {
	if err := d.Valid(); err != nil {
		return nil, err
	}
	return &Advertiser{device: d, last: d.Records(), port: port}, nil
}

func (a *Advertiser) On(host string, ips []string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.on != nil {
		a.on.Shutdown()
		a.on = nil
	}

	// zeroconf wants the trailing dot on the domain.
	server, err := zeroconf.RegisterProxy(
		a.device.Instance(),
		ServiceType,
		"local.",
		a.port,
		host,
		ips,
		a.last,
		nil,
	)
	if err != nil {
		return fmt.Errorf("cast: advertising %q: %w", a.device.Name, err)
	}
	a.on = server
	return nil
}

func Announcing(on Publisher, d Device) *Advertiser {
	return &Advertiser{on: on, device: d, last: d.Records()}
}

func (a *Advertiser) Update(d Device) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	records := d.Records()
	if slices.Equal(records, a.last) {
		return false
	}

	a.device = d
	a.last = records
	if a.on != nil {
		a.on.SetText(records)
	}

	return true
}

func (a *Advertiser) Playing(running bool, status string) bool {
	a.mu.Lock()
	d := a.device
	a.mu.Unlock()

	d.Running = running
	d.Status = status

	return a.Update(d)
}

func (a *Advertiser) Device() Device {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.device
}

func (a *Advertiser) Close() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.on != nil {
		a.on.Shutdown()
		a.on = nil
	}
}
