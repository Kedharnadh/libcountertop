package call

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type homeCalls struct {
	actions []string
	data    []map[string]string
}

func (h *homeCalls) Call(_ context.Context, action string, data map[string]string) error {
	h.actions = append(h.actions, action)
	h.data = append(h.data, data)
	return nil
}

func TestTheHomeLineCarriesAnswersAndEndsAsActions(t *testing.T) {
	home := &homeCalls{}
	l := homeLine{home: home, session: "s1"}

	if status, err := l.send(context.Background(), "answer", envelope{SDP: "v=0"}); err != nil ||
		status != http.StatusOK {
		t.Fatalf("answer went out as %d, %v, want 200", status, err)
	}
	if status, _ := l.send(
		context.Background(),
		"end",
		envelope{Reason: ReasonDeclined},
	); status != http.StatusOK {
		t.Fatalf("end went out as %d, want 200", status)
	}
	if status, _ := l.send(
		context.Background(),
		"offer",
		envelope{},
	); status != http.StatusNotImplemented {
		t.Fatalf("an offer over the home line returned %d, want 501", status)
	}

	want := []string{"echolocal.call_answer", "echolocal.call_end"}
	if len(home.actions) != len(want) || home.actions[0] != want[0] || home.actions[1] != want[1] {
		t.Fatalf("called %v, want %v", home.actions, want)
	}
	if d := home.data[0]; d["session"] != "s1" || d["sdp"] != "v=0" {
		t.Errorf("answer carried %v", d)
	}
	if d := home.data[1]; d["session"] != "s1" || d["reason"] != string(ReasonDeclined) {
		t.Errorf("end carried %v", d)
	}
}

func TestADeviceOfferCannotClaimAUser(t *testing.T) {
	var m envelope
	if err := json.Unmarshal(
		[]byte(`{"call":"c1","sdp":"v=0","User":"u1","user":"u1"}`),
		&m,
	); err != nil {
		t.Fatal(err)
	}
	if m.User != "" {
		t.Errorf("a device's offer set the user to %q", m.User)
	}
}

func TestHomeCallActionsNeedAHome(t *testing.T) {
	settings := &callSettings{value: schema.DefaultCall()}
	without := New(Dependencies{Settings: settings})
	t.Cleanup(without.Close)
	if got := without.Actions(); len(got) != 0 {
		t.Errorf("without a home it publishes %d actions, want none", len(got))
	}
	with := New(Dependencies{Settings: settings, Home: &homeCalls{}})
	t.Cleanup(with.Close)
	if got := with.Actions(); len(got) != 2 || got[0].Name != "call_offer" ||
		got[1].Name != "call_end" {
		t.Errorf("with a home it publishes %v", got)
	}
}

func TestAHomeOfferIsRefusedOrBusyBeforeRinging(t *testing.T) {
	settings := &callSettings{value: schema.DefaultCall()}
	c := New(Dependencies{Settings: settings, Home: &homeCalls{}})
	t.Cleanup(c.Close)
	offer := envelope{Call: "s1", SDP: "v=0"}
	l := homeLine{home: &homeCalls{}, session: "s1"}

	settings.value.Incoming = false
	caller := discovery.Peer{ID: homeCaller}
	if got := c.incoming(offer, caller, l); got != http.StatusForbidden {
		t.Errorf("with incoming calls off it answered %d, want 403", got)
	}

	settings.value.Incoming = true
	c.now = &session{}
	if got := c.incoming(offer, caller, l); got != http.StatusConflict {
		t.Errorf("during another call it answered %d, want 409", got)
	}
	c.now = nil
}
