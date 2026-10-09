package message

import (
	"image"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"

	esphome "github.com/ygelfand/go-esphome-device"

	"github.com/ygelfand/libcountertop/pkg/display/theme"
)

func TestToneFallsBackToInfo(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want schema.Tone
	}{
		{"named", "warning", schema.ToneWarning},
		{"alert", "alert", schema.ToneAlert},
		{"success", "success", schema.ToneSuccess},
		{"info", "info", schema.ToneInfo},
		{"nothing given", "", schema.ToneInfo},
		{"something else entirely", "urgent!!", schema.ToneInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tone(tt.in); got != tt.want {
				t.Errorf("tone(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSeconds(t *testing.T) {
	tests := []struct {
		name string
		in   int
		want time.Duration
	}{
		{"unset takes the default", 0, DefaultSeconds * time.Second},
		{"negative takes the default", -5, DefaultSeconds * time.Second},
		{"a normal ask", 30, 30 * time.Second},
		{"the limit", MaxSeconds, MaxSeconds * time.Second},
		{"past the limit is clamped", 99999, MaxSeconds * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Hold(tt.in); got != tt.want {
				t.Errorf("Hold(%d) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestEveryToneHasAColor(t *testing.T) {
	palette := theme.Default()

	for _, tone := range schema.Tones() {
		c := tone.Color(palette)
		if c == palette.Surface {
			t.Errorf("%s is the same color as the card it sits on", tone)
		}
	}
}

func TestTonesAreLabelledDistinctly(t *testing.T) {
	seen := map[string]bool{}
	for _, tone := range schema.Tones() {
		label := tone.Label()
		if label == "" {
			t.Errorf("%s has no label", tone)
		}
		if seen[label] {
			t.Errorf("two tones are labeled %q", label)
		}
		seen[label] = true
	}
}

func TestActions(t *testing.T) {
	acts := New().Actions()
	if len(acts) != 2 {
		t.Fatalf("%d actions, want show and clear", len(acts))
	}

	var found bool
	for _, a := range acts {
		if a.Name != "show_message" {
			continue
		}
		found = true

		if _, err := a.Run(esphome.Call{}); err == nil {
			t.Error("show_message with no body was accepted")
		}
	}
	if !found {
		t.Error("there is no show_message action")
	}
}

func TestDismissWithNothingShowing(t *testing.T) {
	m := New()

	m.mu.Lock()
	m.timer = nil
	m.mu.Unlock()

	m.Dismiss(0, image.Point{})
}

func TestNegativeContactsAreTakenOnlyWhenDismissing(t *testing.T) {
	m := New()
	defer m.Close()
	m.Dismiss(-1, image.Point{})
	if m.Took(-1) {
		t.Fatal("a touch with nothing showing was consumed")
	}
	m.Show(Message{Body: "up"}, time.Hour)
	m.Dismiss(-2, image.Point{})
	if !m.Took(-2) || m.Took(-1) {
		t.Fatal("the dismissing touch was not the one consumed")
	}
}

func TestATapOutsideTheCaughtAreaPassesThrough(t *testing.T) {
	m := New()
	defer m.Close()
	m.Show(Message{Body: "up"}, time.Hour)
	m.Catch(image.Rect(0, 0, 100, 40))
	m.Dismiss(1, image.Pt(50, 200))
	if m.Took(1) {
		t.Fatal("a tap below the banner was consumed")
	}
	if _, up := m.Current(); !up {
		t.Fatal("a tap below the banner closed it")
	}
	m.Dismiss(2, image.Pt(50, 20))
	if !m.Took(2) {
		t.Fatal("a tap on the banner was not consumed")
	}
	if _, up := m.Current(); up {
		t.Fatal("a tap on the banner left it up")
	}
}

func TestANewMessageCatchesEverywhereUntilPlaced(t *testing.T) {
	m := New()
	defer m.Close()
	m.Show(Message{Body: "first"}, time.Hour)
	m.Catch(image.Rect(0, 0, 100, 40))
	m.Show(Message{Body: "second"}, time.Hour)
	m.Dismiss(3, image.Pt(50, 200))
	if !m.Took(3) {
		t.Fatal("the next message kept the last one's area")
	}
}
