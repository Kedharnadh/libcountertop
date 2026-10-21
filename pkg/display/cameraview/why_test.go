package cameraview

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/homeassistant"
	"github.com/ygelfand/libcountertop/pkg/say"
)

func TestEachStreamFailureHasItsOwnReason(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"undecodable", errUndecodable, say.T("cameras.why.format")},
		{"no picture", errNoPicture, say.T("cameras.why.nopicture")},
		{"lost", fmt.Errorf("%w: failed", errLost), say.T("cameras.why.lost")},
		{"no webrtc", homeassistant.ErrNoWebRTC, say.T("cameras.why.webrtc")},
		{"slow", fmt.Errorf("asking: %w", context.DeadlineExceeded), say.T("cameras.why.slow")},
		{"other", errors.New("something else"), say.T("cameras.why.failed")},
	}
	for _, c := range cases {
		if got := reason(c.err); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
	seen := map[string]string{}
	for _, c := range cases {
		if other, ok := seen[c.want]; ok {
			t.Errorf("%s and %s give the same reason %q", c.name, other, c.want)
		}
		seen[c.want] = c.name
	}
}

func TestARefusalQuotesHomeAssistant(t *testing.T) {
	err := fmt.Errorf(
		"opening: %w",
		&homeassistant.Failed{Message: "Camera does not support WebRTC"},
	)
	got := reason(err)
	if !strings.Contains(got, "Camera does not support WebRTC") {
		t.Errorf("the reason %q leaves out what Home Assistant said", got)
	}
}
