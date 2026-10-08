package sdp

import (
	"slices"
	"testing"
)

func TestARecordsUUIDsAreNotJustItsClasses(t *testing.T) {
	got := AudioSink(1, "Panel", 0x0019, FeatureSpeaker).UUIDs()

	for _, want := range []uint32{
		UUIDAudioSink,  // the class list
		UUIDBrowseRoot, // the browse group
		UUIDL2CAP,      // nested two deep in the protocol list
		UUIDAVDTP,
		UUIDA2DP, // nested in the profile list
	} {
		if !slices.Contains(got, want) {
			t.Errorf("%#x is in the record and not in its uuids: %#x", want, got)
		}
	}
}

func TestNumbersAreNotCollectedAsUUIDs(t *testing.T) {
	got := AudioSink(1, "Panel", 0x0019, FeatureSpeaker).UUIDs()

	// The record handle, the feature bits and the profile version are all plain integers.
	for _, wrong := range []uint32{1, uint32(FeatureSpeaker), A2DPVersion} {
		if slices.Contains(got, wrong) {
			t.Errorf("%#x was collected as a uuid", wrong)
		}
	}
}

func TestTheRemoteControlRecordsCarryTheirClasses(t *testing.T) {
	target := RemoteControl(2, "Panel", 0x0017, 0, UUIDAVRemoteControlTarget, CategoryAmplifier)

	got := target.UUIDs()
	for _, want := range []uint32{UUIDAVRemoteControlTarget, UUIDAVRemoteControl, UUIDAVCTP} {
		if !slices.Contains(got, want) {
			t.Errorf("%#x is missing from %#x", want, got)
		}
	}

	controller := RemoteControl(
		3,
		"Panel",
		0x0017,
		0x001b,
		UUIDAVRemoteControlController,
		CategoryPlayer,
	)
	if slices.Contains(controller.UUIDs(), uint32(UUIDAVRemoteControlTarget)) {
		t.Error("the controller record claims to be the target as well")
	}

	if !slices.Contains(controller.UUIDs(), uint32(UUIDAVRemoteControl)) {
		t.Error("the controller record does not carry the generic remote control class")
	}
}

func TestBrowsingIsAdvertisedWithItsChannel(t *testing.T) {
	plain := RemoteControl(3, "Panel", 0x0017, 0, UUIDAVRemoteControlController, CategoryPlayer)

	if f, _ := plain.Attribute(AttrFeatures); f.Type != TypeNil {
		if v, _ := f.Uint(); v&FeatureBrowsing != 0 {
			t.Error("a record with no browsing channel still claims browsing")
		}
	}
	if _, ok := plain.Attribute(AttrMoreProtocols); ok {
		t.Error("a record with no browsing channel published a second protocol list")
	}

	browsing := RemoteControl(
		3,
		"Panel",
		0x0017,
		0x001b,
		UUIDAVRemoteControlController,
		CategoryPlayer,
	)

	f, ok := browsing.Attribute(AttrFeatures)
	if !ok {
		t.Fatal("no supported features at all")
	}
	if v, _ := f.Uint(); v&FeatureBrowsing == 0 {
		t.Errorf("features %#04x does not claim browsing", v)
	}

	more, ok := browsing.Attribute(AttrMoreProtocols)
	if !ok {
		t.Fatal("browsing is claimed and no channel is published for it")
	}

	if len(more.Children) != 1 {
		t.Fatalf("%d protocol descriptor lists, want one", len(more.Children))
	}

	psm, ok := PSM(more.Children[0])
	if !ok {
		t.Fatal("the published list carries no l2cap channel")
	}
	if psm != 0x001b {
		t.Errorf("browsing published on %#04x, want 0x001b", psm)
	}
}

func TestWantsNeedsEveryUUIDInThePattern(t *testing.T) {
	record := AudioSink(1, "Panel", 0x0019, FeatureSpeaker)

	both := Search{Pattern: []uint32{UUIDAudioSink, UUIDA2DP}}
	if !both.Wants(record.UUIDs()) {
		t.Error("a sink searched for by both its class and its profile did not match")
	}

	missing := Search{Pattern: []uint32{UUIDAudioSink, UUIDAVRemoteControlTarget}}
	if missing.Wants(record.UUIDs()) {
		t.Error("the sink matched a pattern naming a class it does not have")
	}
}
