package dhcp

import (
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

type memory struct {
	address string
	writes  int
}

func (m *memory) Address() string { return m.address }

func (m *memory) SetAddress(s string) error {
	m.address = s
	m.writes++
	return nil
}

func remembering(m *memory) *Client {
	return New(Options{Hostname: func() string { return "kitchen" }, Store: m})
}

func what(t *testing.T, want net.IP) *dhcpv4.DHCPv4 {
	t.Helper()

	msg, err := dhcpv4.New(RequestOptions("kitchen", want)...)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	return msg
}

func TestAFreshDeviceAsksForNoAddress(t *testing.T) {
	c := remembering(&memory{})

	if got := c.previous(); got != nil {
		t.Errorf("a fresh device remembered %v", got)
	}
	if got := what(t, c.previous()).RequestedIPAddress(); got != nil {
		t.Errorf("a fresh device asked for %v", got)
	}
}

func TestAClientWithNoStoreRemembersNothing(t *testing.T) {
	c := New(Options{})

	c.remember(net.IPv4(10, 0, 0, 7))
	if got := c.previous(); got != nil {
		t.Errorf("remembered %v with nowhere to keep it", got)
	}
}

// RFC 2131 section 4.4.1: a client that had an address names it when it asks for one.
func TestARememberedAddressIsAskedFor(t *testing.T) {
	want := net.IPv4(192, 168, 1, 40)
	c := remembering(&memory{address: want.String()})

	if got := what(t, c.previous()).RequestedIPAddress(); !got.Equal(want.To4()) {
		t.Errorf("asked for %v, want %v", got, want)
	}
}

func TestWhatIsAlwaysAskedFor(t *testing.T) {
	for _, want := range []net.IP{nil, net.IPv4(192, 168, 1, 40)} {
		msg := what(t, want)

		if msg.HostName() == "" {
			t.Error("the request carries no hostname")
		}
		if !msg.ParameterRequestList().Has(dhcpv4.OptionNTPServers) {
			t.Error("the request does not ask for NTP servers")
		}
	}
}

func TestRubbishInTheStoreIsIgnored(t *testing.T) {
	for _, said := range []string{"not an address", "2001:db8::1", "999.1.1.1", " "} {
		if got := remembering(&memory{address: said}).previous(); got != nil {
			t.Errorf("%q was read as %v", said, got)
		}
	}
}

func TestTheGrantedAddressIsRemembered(t *testing.T) {
	m := &memory{}
	c := remembering(m)

	c.remember(net.IPv4(10, 0, 0, 7))

	if m.address != "10.0.0.7" {
		t.Errorf("remembered %q, want 10.0.0.7", m.address)
	}
	if got := c.previous(); !got.Equal(net.IPv4(10, 0, 0, 7).To4()) {
		t.Errorf("read back %v", got)
	}
}

func TestRememberingTheSameAddressDoesNotWriteAgain(t *testing.T) {
	m := &memory{}
	c := remembering(m)

	c.remember(net.IPv4(10, 0, 0, 7))
	c.remember(net.IPv4(10, 0, 0, 7))

	if m.writes != 1 {
		t.Errorf("wrote %d times, want once", m.writes)
	}
}

func TestRememberingADifferentAddressWrites(t *testing.T) {
	m := &memory{}
	c := remembering(m)

	c.remember(net.IPv4(10, 0, 0, 7))
	c.remember(net.IPv4(10, 0, 0, 8))

	if m.writes != 2 {
		t.Errorf("wrote %d times, want twice", m.writes)
	}
	if m.address != "10.0.0.8" {
		t.Errorf("the store says %q, want 10.0.0.8", m.address)
	}
}

func TestRememberingNothingForgets(t *testing.T) {
	m := &memory{}
	c := remembering(m)

	c.remember(net.IPv4(10, 0, 0, 7))
	c.remember(nil)

	if m.address != "" {
		t.Errorf("forgetting left %q", m.address)
	}
}
