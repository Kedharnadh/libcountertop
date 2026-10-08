package schema

import "testing"

func TestNoneIsOfferedFirst(t *testing.T) {
	got := Chimes()

	if len(got) == 0 {
		t.Fatal("nothing is offered")
	}
	if got[0] != ChimeNone {
		t.Errorf("the first choice is %v, want None", got[0])
	}
}

func TestOnlyNoneIsSilent(t *testing.T) {
	if !ChimeNone.Silent() {
		t.Error("None makes a sound")
	}
	for _, c := range Chimes() {
		if c != ChimeNone && c.Silent() {
			t.Errorf("%v is silent, and only None should be", c)
		}
	}
}

func TestEveryChimeHasItsOwnLabel(t *testing.T) {
	seen := map[string]Chime{}

	for _, c := range Chimes() {
		label := c.Label()
		if had, ok := seen[label]; ok {
			t.Errorf("%v and %v are both shown as %q", had, c, label)
		}
		seen[label] = c
	}
}
