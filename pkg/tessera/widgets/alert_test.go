package widgets

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/message"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

func TestAlerterReportsHowAlertsEnd(t *testing.T) {
	m := message.New()
	defer m.Close()
	show, clear := Alerter(m)
	var (
		mu    sync.Mutex
		ended []string
	)
	seen := func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(ended)
	}
	alert := func(buttons ...string) tessera.Alert {
		a := tessera.Alert{Title: "door", Ended: func(action string) {
			mu.Lock()
			ended = append(ended, action)
			mu.Unlock()
		}}
		for _, b := range buttons {
			a.Buttons = append(a.Buttons, tessera.AlertButton{Text: b})
		}
		return a
	}

	show(alert("OK"))
	if cur, _ := m.Current(); len(cur.Keys) != 1 || cur.Keys[0].Text != "OK" {
		t.Fatalf("keys %+v", cur.Keys)
	}
	m.Press(0)

	show(alert("Open", "Snooze"))
	if cur, _ := m.Current(); cur.Keys[0].Text != "Snooze" || cur.Keys[1].Text != "Open" {
		t.Fatalf("the second button is not on the left: %+v", cur.Keys)
	}
	m.Press(0)
	show(alert("Open", "Snooze"))
	m.Press(1)

	show(alert("OK"))
	show(alert("OK"))
	clear()

	a := alert("OK")
	a.Timeout = time.Millisecond
	show(a)
	deadline := time.Now().Add(time.Second)
	for len(seen()) < 6 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	got := seen()

	want := []string{
		tessera.AlertOK,
		tessera.AlertSecond,
		tessera.AlertOK,
		tessera.AlertReplaced,
		tessera.AlertRemote,
		tessera.AlertTimeout,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("ended %v, want %v", got, want)
	}
}

func TestKeyedMessageIgnoresStrayTaps(t *testing.T) {
	m := message.New()
	defer m.Close()
	show, _ := Alerter(m)
	show(tessera.Alert{Buttons: []tessera.AlertButton{{Text: "OK"}}})
	m.Dismiss(3)
	if m.Took(3) {
		t.Fatal("a tap meant for a button was swallowed")
	}
	if _, up := m.Current(); !up {
		t.Fatal("a stray tap closed an alert that waits for its button")
	}
}
