package dhcp

import (
	"net"
	"testing"
)

// A renewal usually comes back with the address the device already had, and that one must survive
// the flush: deleting it takes every socket bound to it, so Home Assistant, the web server and
// mDNS would all drop each time the lease came round.
func TestKeepingTheAddressARenewalReturns(t *testing.T) {
	have := net.IPv4(10, 100, 101, 106)

	if !keeping(have, 24, net.IPv4(10, 100, 101, 106), 24) {
		t.Error("the address the lease returned was not kept")
	}
}

// A lease onto a different address has to take the old one off, or the device answers on both.
func TestKeepingDropsAnAddressTheLeaseDidNotReturn(t *testing.T) {
	if keeping(net.IPv4(10, 100, 101, 5), 24, net.IPv4(10, 100, 101, 106), 24) {
		t.Error("an address the lease did not return was kept")
	}
}

// The same address on a different prefix is a different address as far as routing is concerned.
func TestKeepingDropsADifferentPrefix(t *testing.T) {
	have := net.IPv4(10, 100, 101, 106)

	if keeping(have, 16, have, 24) {
		t.Error("the same address on a wider prefix was kept")
	}
}

// Releasing keeps nothing: the device is letting the address go rather than renewing it.
func TestKeepingNothingWhenThereIsNothingToKeep(t *testing.T) {
	if keeping(net.IPv4(10, 100, 101, 106), 24, nil, 0) {
		t.Error("an address was kept when the lease was being released")
	}
	if keeping(nil, 24, net.IPv4(10, 100, 101, 106), 24) {
		t.Error("an address with no value was kept")
	}
}

// The kernel reports an address in whichever form it likes, and net.IP compares by value, so the
// four byte and sixteen byte spellings of the same address have to match.
func TestKeepingMatchesEitherSpelling(t *testing.T) {
	four := net.IPv4(10, 100, 101, 106).To4()
	sixteen := net.IPv4(10, 100, 101, 106).To16()

	if !keeping(four, 24, sixteen, 24) {
		t.Error("the same address written two ways did not match")
	}
}
