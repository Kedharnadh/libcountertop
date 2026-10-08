package component

import "testing"

type coming struct {
	name string
	p    Progress
}

func (c *coming) Name() string      { return c.name }
func (c *coming) Startup() Progress { return c.p }

type quiet struct{ name string }

func (q *quiet) Name() string { return q.name }

func TestProgressOnlyCollectsWhatSpeaks(t *testing.T) {
	r := &Registry[testConfig]{}
	r.Add(Hardware, func() Component { return &coming{name: "wifi", p: Progress{}} })
	r.Add(Hardware, func() Component { return &quiet{name: "display"} })

	got := r.Progress()
	if len(got) != 1 {
		t.Fatalf("collected %d, want only the one that speaks: %+v", len(got), got)
	}
	if got[0].Name != "wifi" {
		t.Errorf("collected %q", got[0].Name)
	}
}

func TestProgressCarriesTheName(t *testing.T) {
	r := &Registry[testConfig]{}
	r.Add(Hardware, func() Component { return &coming{name: "dhcp", p: Progress{Doing: "asking"}} })

	got := r.Progress()
	if got[0].Name != "dhcp" || got[0].Doing != "asking" {
		t.Errorf("got %+v", got[0])
	}
}

func TestReadyWaitsForWhatHolds(t *testing.T) {
	r := &Registry[testConfig]{}
	r.Add(Hardware, func() Component { return &coming{name: "wifi", p: Progress{}} })

	if r.Ready() {
		t.Error("ready while something that holds is not done")
	}
}

func TestEverythingElseOnTheScreenIsWaitedFor(t *testing.T) {
	r := &Registry[testConfig]{}
	r.Add(Hardware, func() Component { return &coming{name: "wifi", p: Progress{Done: true}} })
	r.Add(Network, func() Component {
		return &coming{name: "dhcp", p: Progress{Doing: "asking for an address"}}
	})

	if r.Ready() {
		t.Error("ready while something on the screen is still coming up")
	}

	for _, p := range r.Progress() {
		if p.Name == "dhcp" && p.Done {
			t.Error("dhcp reported done while it is still asking")
		}
	}
}

func TestReadyWithNothingToWaitFor(t *testing.T) {
	r := &Registry[testConfig]{}
	r.Add(Hardware, func() Component { return &quiet{name: "display"} })

	if !r.Ready() {
		t.Error("a device with nothing to wait for is not ready")
	}
}
