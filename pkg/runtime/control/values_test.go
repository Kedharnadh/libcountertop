package control

import (
	"log/slog"
	"strings"
	"testing"
)

type face string

func (f face) Label() string { return strings.ToUpper(string(f[:1])) + string(f[1:]) }

var faces = []face{"analog", "digital", "word"}

func TestChooseTakesTheLabelOrTheStoredName(t *testing.T) {
	for _, want := range faces {
		for _, said := range []string{want.Label(), string(want), strings.ToUpper(string(want))} {
			var got face
			if err := Choose(faces, func(f face) { got = f })(said); err != nil {
				t.Errorf("%q: %v", said, err)
				continue
			}
			if got != want {
				t.Errorf("%q chose %q, want %q", said, got, want)
			}
		}
	}
}

func TestChooseRefusesWhatIsNotAnOption(t *testing.T) {
	var called bool
	err := Choose(faces, func(face) { called = true })("sundial")

	if err == nil {
		t.Fatal("a face that does not exist was accepted")
	}
	if called {
		t.Error("the setter ran anyway")
	}
	if !strings.Contains(err.Error(), "Analog") {
		t.Errorf("the error does not list the options: %v", err)
	}
}

func TestToggleReadsTheUsualWords(t *testing.T) {
	for said, want := range map[string]bool{
		"on": true, "true": true, "yes": true, "1": true, "ON": true,
		"off": false, "false": false, "no": false, "0": false,
	} {
		var got bool
		if err := Toggle(func(v bool) { got = v })(said); err != nil {
			t.Errorf("%q: %v", said, err)
			continue
		}
		if got != want {
			t.Errorf("%q read as %v, want %v", said, got, want)
		}
	}
}

func TestToggleRefusesAnythingElse(t *testing.T) {
	for _, said := range []string{"maybe", ""} {
		if err := Toggle(func(bool) {})(said); err == nil {
			t.Errorf("%q was accepted as a switch", said)
		}
	}
}

func TestNumberRefusesOutOfRange(t *testing.T) {
	for _, said := range []string{"-1", "101", "howbright"} {
		var called bool
		if err := Number(0, 100, func(int) { called = true })(said); err == nil {
			t.Errorf("%q was accepted", said)
		}
		if called {
			t.Errorf("%q ran the setter", said)
		}
	}

	var got int
	if err := Number(0, 100, func(v int) { got = v })("76"); err != nil {
		t.Fatalf("76: %v", err)
	}
	if got != 76 {
		t.Errorf("76 read as %d", got)
	}
}

func TestMapReadsThroughTheView(t *testing.T) {
	rows := Map(
		[]Setting[string]{{Name: "n", Field: "F", Read: func(s string) string { return s }}},
		func(n int) string { return strings.Repeat("x", n) },
	)
	if rows[0].Read(3) != "xxx" || rows[0].Field != "F" || rows[0].Name != "n" {
		t.Errorf("mapped row is %+v reading %q", rows[0], rows[0].Read(3))
	}
}

func TestLogLevelChangesTheThreshold(t *testing.T) {
	var v slog.LevelVar
	if got, err := LogLevel(&v, []string{"debug"}); err != nil || got != "debug" {
		t.Fatalf("debug: %q, %v", got, err)
	}
	if got, _ := LogLevel(&v, nil); got != "debug" {
		t.Errorf("reads back as %q", got)
	}
	if _, err := LogLevel(&v, []string{"loud"}); err == nil {
		t.Error("loud was accepted as a level")
	}
}
