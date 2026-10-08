package dhcp

import (
	"errors"
	"net"
	"testing"

	"github.com/insomniacslk/dhcp/dhcpv4"
)

func leased() *Lease {
	return &Lease{
		Address: net.IPNet{IP: net.IPv4(192, 168, 1, 40), Mask: net.CIDRMask(24, 32)},
		Router:  net.IPv4(192, 168, 1, 1),
		Server:  net.IPv4(192, 168, 1, 1),
	}
}

func released(t *testing.T) *dhcpv4.DHCPv4 {
	t.Helper()

	l := leased()
	payload, err := releaseMessage(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01},
		l.Address.IP, l.Server)
	if err != nil {
		t.Fatalf("building a release: %v", err)
	}

	msg, err := dhcpv4.FromBytes(payload)
	if err != nil {
		t.Fatalf("reading the release back: %v", err)
	}
	return msg
}

func TestAReleaseSaysItIsOne(t *testing.T) {
	if got := released(t).MessageType(); got != dhcpv4.MessageTypeRelease {
		t.Errorf("the message type is %v, want a release", got)
	}
}

func TestTheAddressIsInCiaddr(t *testing.T) {
	msg := released(t)

	if want := leased().Address.IP.To4(); !msg.ClientIPAddr.Equal(want) {
		t.Errorf("ciaddr is %v, want %v", msg.ClientIPAddr, want)
	}
}

func TestTheRequestedAddressOptionIsAbsent(t *testing.T) {
	if got := released(t).RequestedIPAddress(); got != nil {
		t.Errorf("the requested address option is set to %v", got)
	}
}

func TestTheServerIsNamed(t *testing.T) {
	msg := released(t)

	if want := leased().Server.To4(); !msg.ServerIdentifier().Equal(want) {
		t.Errorf("the server identifier is %v, want %v", msg.ServerIdentifier(), want)
	}
}

func TestTheHardwareAddressIsCarried(t *testing.T) {
	want := net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01}

	if got := released(t).ClientHWAddr; got.String() != want.String() {
		t.Errorf("the hardware address is %v, want %v", got, want)
	}
}

func TestAReleaseDoesNotAskToBeBroadcastAt(t *testing.T) {
	if released(t).IsBroadcast() {
		t.Error("the release set the broadcast flag")
	}
}

func TestAReleaseWithNoServerIsRefused(t *testing.T) {
	_, err := releaseMessage(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01},
		net.IPv4(192, 168, 1, 40), nil)

	if err == nil {
		t.Fatal("a release with no server was built anyway")
	}
}

func TestAReleaseOfSomethingThatIsNotIPv4IsRefused(t *testing.T) {
	_, err := releaseMessage(net.HardwareAddr{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01},
		net.ParseIP("2001:db8::1"), net.IPv4(192, 168, 1, 1))

	if err == nil {
		t.Fatal("an IPv6 address was released over DHCPv4")
	}
}

func TestReleasingNothingIsNotAnError(t *testing.T) {
	if err := sendRelease("lo", nil); err != nil {
		t.Errorf("releasing no lease reported %v", err)
	}
}

func TestStoppingReleasesTheLease(t *testing.T) {
	was := releasing
	defer func() { releasing = was }()

	var got *Lease
	releasing = func(_ string, l *Lease) error {
		got = l
		return nil
	}

	c := &Client{}
	c.set(leased())

	if err := c.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}

	if got == nil {
		t.Fatal("stopping did not release the lease")
	}
	if !got.Address.IP.Equal(leased().Address.IP) {
		t.Errorf("released %v, want %v", got.Address.IP, leased().Address.IP)
	}
}

func TestStoppingWithoutALeaseReleasesNothing(t *testing.T) {
	was := releasing
	defer func() { releasing = was }()

	var called bool
	releasing = func(string, *Lease) error {
		called = true
		return nil
	}

	c := &Client{}
	if err := c.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}

	if called {
		t.Error("a client with no lease released one")
	}
}

func TestAFailedReleaseStillStops(t *testing.T) {
	was := releasing
	defer func() { releasing = was }()

	releasing = func(string, *Lease) error { return errors.New("no route to host") }

	c := &Client{}
	c.set(leased())

	if err := c.Close(); err != nil {
		t.Errorf("closing reported %v, want it to stop anyway", err)
	}
	if c.Lease() != nil {
		t.Error("the lease was kept after stopping")
	}
}
