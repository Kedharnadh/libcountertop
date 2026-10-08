package control

import (
	setting "github.com/ygelfand/libcountertop/pkg/settings"

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

type fixed struct{ value string }

func (f *fixed) Domain() string                  { return "demo" }
func (f *fixed) Names() []string                 { return []string{"level"} }
func (f *fixed) Value(string) (string, error)    { return f.value, nil }
func (f *fixed) Change(_ string, v string) error { f.value = v; return nil }

func TestRegisteredSettingsAreReachedByDomainAndName(t *testing.T) {
	var r setting.Registry
	f := &fixed{value: "3"}
	r.Add(f)
	rows := Registered[struct{}](&r)
	out, err := Settings(func() struct{} { return struct{}{} }, rows, []string{"demo.level", "7"})
	if err != nil || out != "7" || f.value != "7" {
		t.Errorf("settings demo.level 7 = %q, %v (value %q)", out, err, f.value)
	}
}
