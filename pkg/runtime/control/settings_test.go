package control

import (
	"errors"
	"strings"
	"testing"
)

func TestSettingsUseSetterAndReadFreshState(t *testing.T) {
	state := "old"
	rows := []Setting[string]{
		{
			Name:  "title",
			Read:  func(s string) string { return s },
			Write: func(s string) error { state = s; return nil },
		},
	}
	got, err := Settings(
		func() string { return state },
		rows,
		[]string{"TITLE", "Kitchen", "Display"},
	)
	if err != nil || got != "Kitchen Display" {
		t.Fatalf("settings=%q, %v", got, err)
	}
	failure := errors.New("save failed")
	rows[0].Write = func(string) error { return failure }
	if _, err := Settings(
		func() string { return state },
		rows,
		[]string{"title", "new"},
	); !errors.Is(
		err,
		failure,
	) {
		t.Fatalf("setter error=%v", err)
	}
	if state != "Kitchen Display" {
		t.Fatal("failed setter changed state")
	}
	if _, err := Settings(
		func() string { return state },
		rows,
		[]string{"missing"},
	); err == nil ||
		!strings.Contains(err.Error(), "settings") {
		t.Fatalf("unknown-setting error=%v", err)
	}
}
