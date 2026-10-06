package sendspin

import "testing"

func TestSpeakerHandoffSilencesThisRoomAndRejoinsTheCurrentGroupFrame(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(48, 1000))
	o.write(microsFor(48), tone(48, 2000))
	o.Stand(true)
	for _, sample := range rendered(o, 1000, 48) {
		if sample != 0 {
			t.Fatal("held room remained audible", sample)
		}
	}
	// The group and the output clock advance while another producer owns the speaker.
	for _, sample := range rendered(o, 1048, 48) {
		if sample != 0 {
			t.Fatal("held room remained audible", sample)
		}
	}
	o.Stand(false)
	for _, sample := range rendered(o, 1048, 48) {
		if sample != 2000 {
			t.Fatal("replayed missed audio rather than rejoining the group", sample)
		}
	}
}

func TestRepeatedStandingAndReleasePreserveLocalOutputState(t *testing.T) {
	o := anchored(t)
	o.write(microsFor(0), tone(48, 1000))
	o.Stand(true)
	o.Stand(true)
	if sample := rendered(o, 1000, 48)[0]; sample != 0 {
		t.Fatal(sample)
	}
	o.Stand(false)
	o.Stand(false)
	if sample := rendered(o, 1000, 48)[0]; sample != 1000 {
		t.Fatal(sample)
	}
}
