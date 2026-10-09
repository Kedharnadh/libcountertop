package message

import (
	"image"
	"testing"
	"time"
)

func TestReplacementIsNotHiddenByOldExpiry(t *testing.T) {
	m := New()
	defer m.Close()
	m.Show(Message{Body: "first"}, time.Hour)
	old := m.generation
	m.Show(Message{Body: "second"}, time.Hour)
	m.expire(old)
	if current, up := m.Current(); !up || current.Body != "second" {
		t.Fatal("old message timer hid the replacement")
	}
	m.Dismiss(7, image.Point{})
	if !m.Took(7) {
		t.Fatal("dismissed touch was not consumed")
	}
	if _, up := m.Current(); up {
		t.Fatal("dismissed message remained visible")
	}
}
