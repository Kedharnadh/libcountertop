package wifi

import "testing"

func TestRepeatedConnectedDoesNotQuerySupplicant(t *testing.T) {
	r := &Radio{up: true, ssid: "home", ctl: &Control{}}
	for range 100 {
		r.connected()
	}
	if !r.up || r.ssid != "home" {
		t.Fatal("repeated completion changed the association")
	}
}

func TestConnectedEventCompletesStartupWithoutVendorStateEvent(t *testing.T) {
	r := &Radio{ssid: "home"}
	r.connected()
	if p := r.Startup(); !p.Done || r.Network() != "home" || !r.Configured() {
		t.Fatalf("CONNECTED did not update association: %+v", p)
	}
	r.apply(false, "")
	if p := r.Startup(); p.Done || r.Network() != "" {
		t.Fatalf("disconnect retained association: %+v", p)
	}
}

func TestUnknownConfigurationDoesNotPretendThereIsNoNetwork(t *testing.T) {
	r := &Radio{}
	if p := r.Startup(); p.Failed {
		t.Fatalf("unchecked configuration reported failure: %+v", p)
	}
	r.configuredKnown = true
	if p := r.Startup(); !p.Failed {
		t.Fatalf("known empty configuration did not report failure: %+v", p)
	}
}
