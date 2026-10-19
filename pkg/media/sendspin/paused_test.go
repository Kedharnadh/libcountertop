package sendspin

import "testing"

func TestAStoppedGroupIsPausedOnlyWhileItShowsATrack(t *testing.T) {
	p := New(Dependencies{Settings: &memory{}})
	p.plays(track{Title: "Africa", Artist: "Toto"})
	p.grouped("stopped")

	if _, paused := p.Playing(); !paused {
		t.Fatal("stopped with a track on show should read as paused")
	}

	p.plays(track{})
	if _, paused := p.Playing(); paused {
		t.Error("stopped with the metadata cleared still reads as paused")
	}
}

func TestAPausedGroupStaysPausedWithoutATitle(t *testing.T) {
	p := New(Dependencies{Settings: &memory{}})
	p.grouped("paused")

	if _, paused := p.Playing(); !paused {
		t.Error("a paused group should read as paused")
	}
}
