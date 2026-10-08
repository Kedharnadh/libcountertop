package a2dp

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/bluetooth/avrcp"
	media "github.com/ygelfand/libcountertop/pkg/media/source"
)

func TestWhatThePlayerIsShown(t *testing.T) {
	state := avrcp.State{
		Track: avrcp.Track{
			Title:  "Aja",
			Artist: "Steely Dan",
			Album:  "Aja",
			Genre:  "Jazz rock",
		},
		Progress: avrcp.Progress{Status: avrcp.StatusPlaying},
	}

	got := nowOf(state, true, nil)

	if !got.Playing || got.Paused {
		t.Errorf("playing=%v paused=%v, want playing", got.Playing, got.Paused)
	}
	if got.Title != "Aja" || got.Artist != "Steely Dan" || got.Album != "Aja" {
		t.Errorf("the card would read %q by %q on %q", got.Title, got.Artist, got.Album)
	}
	if !got.Can.Has(media.CanPause | media.CanNext | media.CanPrevious) {
		t.Errorf("the transport offered is %b, want pause, next and previous", got.Can)
	}
	if got.Can.Has(media.CanStop) {
		t.Error("the card offers stop as well as pause")
	}
}

func TestPausedIsNeitherPlayingNorStopped(t *testing.T) {
	got := nowOf(avrcp.State{Progress: avrcp.Progress{Status: avrcp.StatusPaused}}, true, nil)

	if got.Playing {
		t.Error("a paused phone reads as playing")
	}
	if !got.Paused {
		t.Error("a paused phone does not read as paused")
	}
}

func TestEverythingElseIsNeither(t *testing.T) {
	for _, status := range []byte{
		avrcp.StatusStopped, avrcp.StatusForward, avrcp.StatusReverse, avrcp.StatusError,
	} {
		got := nowOf(avrcp.State{Progress: avrcp.Progress{Status: status}}, false, nil)

		if got.Playing || got.Paused {
			t.Errorf("status %#02x with no stream reads playing=%v paused=%v, want neither",
				status, got.Playing, got.Paused)
		}
	}
}

func TestNothingConnectedIsBlankAndSafe(t *testing.T) {
	s := source{sink: testSink()}

	if got := s.Now(); got.Playing || got.Paused || got.Title != "" {
		t.Errorf("a sink with no phone reads %+v", got)
	}

	s.Play()
	s.Pause()
	s.Stop()
	s.Next()
	s.Previous()
}

func TestEachButtonIsItsOwnOpcode(t *testing.T) {
	seen := map[byte]bool{}
	for _, op := range []byte{
		avrcp.OpPlay, avrcp.OpPause, avrcp.OpStop, avrcp.OpNext, avrcp.OpPrevious,
	} {
		if seen[op] {
			t.Errorf("opcode %#02x is used for two buttons", op)
		}
		seen[op] = true
	}
}

func TestAudioArrivingIsWhatPlayingMeans(t *testing.T) {
	silent := avrcp.State{Progress: avrcp.Progress{Status: avrcp.StatusStopped}}

	if got := nowOf(silent, true, nil); !got.Playing {
		t.Error("audio is arriving and the card says nothing is playing")
	}
	if got := nowOf(silent, false, nil); got.Playing {
		t.Error("no audio is arriving and the card says something is playing")
	}

	paused := avrcp.State{Progress: avrcp.Progress{Status: avrcp.StatusPaused}}
	if got := nowOf(paused, true, nil); got.Playing || !got.Paused {
		t.Errorf(
			"a paused phone with the stream up reads playing=%v paused=%v",
			got.Playing,
			got.Paused,
		)
	}
}
