package dhcp

import (
	"net"
	"testing"
	"time"
)

var (
	us    = net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01}
	them  = net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x02}
	held  = net.IPv4(192, 168, 1, 40)
	other = net.IPv4(192, 168, 1, 41)
)

func claim(sender net.HardwareAddr, addr net.IP) []byte {
	return arpAnnounce(sender, addr)
}

func TestAnotherHostClaimingOurAddressIsAConflict(t *testing.T) {
	if !conflicting(claim(them, held), us, held) {
		t.Error("another host claiming our address was not a conflict")
	}
}

func TestOurOwnAnnouncementIsNotAConflict(t *testing.T) {
	if conflicting(claim(us, held), us, held) {
		t.Error("the device's own announcement was read as a conflict")
	}
}

func TestAClaimToSomeOtherAddressIsNotAConflict(t *testing.T) {
	if conflicting(claim(them, other), us, held) {
		t.Error("a claim to a different address was a conflict")
	}
}

func TestAProbeForOurAddressIsNotAConflict(t *testing.T) {
	if conflicting(arpProbe(them, held), us, held) {
		t.Error("a probe was read as an ongoing conflict")
	}
}

func TestRubbishIsNotAConflict(t *testing.T) {
	for _, frame := range [][]byte{
		nil,
		make([]byte, 4),
		make([]byte, ethHeader+arpPayload),
	} {
		if conflicting(frame, us, held) {
			t.Errorf("a frame of %d bytes was read as a conflict", len(frame))
		}
	}
}

func TestAConflictAgainstNoAddressIsNotOne(t *testing.T) {
	if conflicting(claim(them, held), us, nil) {
		t.Error("a device with no address had one conflicted")
	}
}

func TestTheFirstConflictIsDefendedAndTheSecondGivesUp(t *testing.T) {
	var d defender
	at := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

	if got := d.decide(at); got != defend {
		t.Errorf("the first conflict was answered with %v, want defend", got)
	}
	if got := d.decide(at.Add(time.Second)); got != surrender {
		t.Errorf("a second conflict a second later was answered with %v, want surrender", got)
	}
}

func TestAConflictPastTheIntervalIsDefendedAgain(t *testing.T) {
	var d defender
	at := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

	if got := d.decide(at); got != defend {
		t.Fatalf("the first conflict was answered with %v", got)
	}
	if got := d.decide(at.Add(defendInterval + time.Second)); got != defend {
		t.Errorf("a conflict after the interval was answered with %v, want defend", got)
	}
}

func TestTheIntervalBoundaryDefends(t *testing.T) {
	var d defender
	at := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

	d.decide(at)

	if got := d.decide(at.Add(defendInterval)); got != defend {
		t.Errorf("a conflict exactly on the interval was answered with %v, want defend", got)
	}
}

func TestAnArgumentThatSettledIsForgotten(t *testing.T) {
	var d defender
	at := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

	d.decide(at)
	d.settled(at.Add(time.Hour))

	if got := d.decide(at.Add(time.Hour)); got != defend {
		t.Errorf("after settling, the next conflict was answered with %v, want defend", got)
	}
}

func TestSettlingInsideTheIntervalDoesNotForget(t *testing.T) {
	var d defender
	at := time.Date(2026, time.September, 20, 12, 0, 0, 0, time.UTC)

	d.decide(at)
	d.settled(at.Add(time.Second))

	if got := d.decide(at.Add(2 * time.Second)); got != surrender {
		t.Errorf("a continuing argument was answered with %v, want surrender", got)
	}
}

func TestADefenceIsAnAnnouncementOfOurOwn(t *testing.T) {
	frame := arpAnnounce(us, held)

	got, ok := readARP(frame)
	if !ok {
		t.Fatal("the defence is not a readable ARP frame")
	}
	if got.op != arpRequest {
		t.Errorf("the defence is operation %d, want a request", got.op)
	}
	if !got.senderIP.Equal(held.To4()) {
		t.Errorf("the defence claims %v, want %v", got.senderIP, held)
	}
	if got.senderMAC.String() != us.String() {
		t.Errorf("the defence is from %v, want %v", got.senderMAC, us)
	}

	if !conflicting(frame, them, held) {
		t.Error("our defence would not register as a claim on the other host")
	}
}

func TestTheAnswersAreNamed(t *testing.T) {
	for _, at := range []struct {
		a    answer
		want string
	}{{ignore, "ignore"}, {defend, "defend"}, {surrender, "surrender"}} {
		if got := at.a.String(); got != at.want {
			t.Errorf("%d is %q, want %q", at.a, got, at.want)
		}
	}
}
