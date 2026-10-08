package dhcp

import (
	"context"
	"net"
	"testing"
	"time"
)

func instead(t *testing.T, fn func(string, net.IP) (bool, error)) *int {
	t.Helper()

	wasProbe, wasDecline := probing, declining
	t.Cleanup(func() { probing, declining = wasProbe, wasDecline })

	calls := 0
	probing = func(iface string, ip net.IP) (bool, error) {
		calls++
		return fn(iface, ip)
	}
	declining = func(string, net.IP, net.IP, string) error { return nil }
	return &calls
}

func told(t *testing.T) *int {
	t.Helper()

	was := declining
	t.Cleanup(func() { declining = was })

	sent := 0
	declining = func(string, net.IP, net.IP, string) error {
		sent++
		return nil
	}
	return &sent
}

func leaseFor(ip string) *Lease {
	return &Lease{Address: net.IPNet{IP: net.ParseIP(ip), Mask: net.CIDRMask(24, 32)}}
}

func TestARenewalOfTheSameAddressIsNotProbed(t *testing.T) {
	calls := instead(t, func(string, net.IP) (bool, error) { return false, nil })

	c := &Client{}
	c.set(leaseFor("10.100.101.106"))

	taken, err := c.contested(context.Background(), leaseFor("10.100.101.106"))
	if err != nil {
		t.Fatal(err)
	}
	if taken {
		t.Error("a renewal was refused")
	}
	if *calls != 0 {
		t.Errorf("the address was probed %d times, want none", *calls)
	}
}

func TestANewAddressIsProbed(t *testing.T) {
	calls := instead(t, func(string, net.IP) (bool, error) { return false, nil })

	c := &Client{}
	c.set(leaseFor("10.100.101.106"))

	if _, err := c.contested(context.Background(), leaseFor("10.100.101.50")); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Errorf("the address was probed %d times, want once", *calls)
	}
}

func TestTheFirstAddressIsProbed(t *testing.T) {
	calls := instead(t, func(string, net.IP) (bool, error) { return false, nil })

	if _, err := (&Client{}).contested(
		context.Background(),
		leaseFor("10.100.101.106"),
	); err != nil {
		t.Fatal(err)
	}
	if *calls != 1 {
		t.Errorf("the address was probed %d times, want once", *calls)
	}
}

func TestAnAddressSomebodyElseHoldsIsRefused(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) { return true, nil })

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	taken, err := (&Client{}).contested(ctx, leaseFor("10.100.101.106"))
	if !taken {
		t.Error("an address somebody else holds was taken")
	}
	if err == nil {
		t.Error("the wait after a refusal did not notice the context was done")
	}
}

func TestARefusedAddressIsDeclinedToTheServer(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) { return true, nil })
	sent := told(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := (&Client{}).contested(ctx, leaseFor("10.100.101.106")); err == nil {
		t.Fatal("the wait after a refusal did not notice the context was done")
	}
	if *sent != 1 {
		t.Errorf("%d declines were sent, want one", *sent)
	}
}

func TestAnAddressThatIsFreeIsNotDeclined(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) { return false, nil })
	sent := told(t)

	if _, err := (&Client{}).contested(
		context.Background(),
		leaseFor("10.100.101.106"),
	); err != nil {
		t.Fatal(err)
	}
	if *sent != 0 {
		t.Errorf("%d declines were sent for a free address", *sent)
	}
}

func TestAProbeThatCannotRunDeclinesNothing(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) {
		return false, net.UnknownNetworkError("no packet socket")
	})
	sent := told(t)

	if _, err := (&Client{}).contested(
		context.Background(),
		leaseFor("10.100.101.106"),
	); err != nil {
		t.Fatal(err)
	}
	if *sent != 0 {
		t.Errorf("%d declines were sent after a probe that did not run", *sent)
	}
}

func TestAProbeThatCannotRunTakesTheAddress(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) {
		return false, net.UnknownNetworkError("no packet socket")
	})

	taken, err := (&Client{}).contested(context.Background(), leaseFor("10.100.101.106"))
	if err != nil {
		t.Fatalf("a failed probe was reported as a failure to get an address: %v", err)
	}
	if taken {
		t.Error("a failed probe was read as the address being taken")
	}
}

func TestTheWaitAfterARefusalIsRespected(t *testing.T) {
	instead(t, func(string, net.IP) (bool, error) { return true, nil })

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	if _, err := (&Client{}).contested(ctx, leaseFor("10.100.101.106")); err == nil {
		t.Fatal("the wait ended early without the context being done")
	}
	if took := time.Since(start); took < 40*time.Millisecond {
		t.Errorf("came back after %v, so nothing waited", took)
	}
	if backoff != 10*time.Second {
		t.Errorf("the wait is %v, want the ten seconds RFC 5227 asks for", backoff)
	}
}
