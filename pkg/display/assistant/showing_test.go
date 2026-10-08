package assistant

import (
	"testing"
)

func TestTheReplyIsRevealedByCharacter(t *testing.T) {
	const text = "The kitchen light is on"

	for _, tc := range []struct {
		part float64
		want string
	}{
		{0, ""},
		{-1, ""},
		{1, text},
		{2, text},
	} {
		if got := Revealed(text, tc.part); got != tc.want {
			t.Errorf("at %v: %q, want %q", tc.part, got, tc.want)
		}
	}

	half := Revealed(text, 0.5)
	if len(half) == 0 || len(half) >= len(text) {
		t.Errorf("half way through reads %q", half)
	}
	if text[:len(half)] != half {
		t.Errorf("%q is not the start of %q", half, text)
	}
}

func TestARevealNeverCutsARuneInHalf(t *testing.T) {
	const text = "påslaget"

	for i := range 40 {
		got := Revealed(text, float64(i)/40)
		for _, r := range got {
			if r == '�' {
				t.Fatalf("at %d/40 the reveal reads %q", i, got)
			}
		}
	}
}
