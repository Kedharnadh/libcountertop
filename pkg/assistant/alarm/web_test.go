package alarm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

func TestTheWebPageAddsEnablesAndDeletesAlarms(t *testing.T) {
	r := newRig(t)
	srv := httptest.NewServer(r.alarms.Handler(func() []setting.Option {
		return []setting.Option{{Value: "custom:rooster", Label: "Rooster"}}
	}))
	t.Cleanup(srv.Close)

	call := func(method, path, body string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(method, srv.URL+path, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { resp.Body.Close() })
		return resp
	}

	resp := call(
		http.MethodPost,
		"/",
		`{"name":"Work","time":"06:45","days":["mon","fri"],"enabled":true,`+
			`"sound":"timer","snooze_minutes":9,"volume":"static"}`,
	)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("adding answered %d, want 200", resp.StatusCode)
	}
	var added Alarm
	if err := json.NewDecoder(resp.Body).Decode(&added); err != nil {
		t.Fatal(err)
	}
	if added.ID == "" || added.Clock() != "06:45" || len(added.Days) != 2 {
		t.Fatalf("added %+v, want an id, 06:45 on two days", added)
	}

	if got := call(
		http.MethodPost,
		"/"+added.ID+"/enable",
		`{"enabled":false}`,
	); got.StatusCode != http.StatusNoContent {
		t.Fatalf("disabling answered %d, want 204", got.StatusCode)
	}
	var v view
	if err := json.NewDecoder(call(http.MethodGet, "/", "").Body).Decode(&v); err != nil {
		t.Fatal(err)
	}
	if len(v.Alarms) != 1 || v.Alarms[0].On {
		t.Errorf("listed %+v, want the one alarm, off", v.Alarms)
	}
	if last := v.Sounds[len(v.Sounds)-1]; last.Label != "Rooster" {
		t.Errorf("last sound %+v, want the custom tone", last)
	}

	if got := call(
		http.MethodPut,
		"/"+added.ID,
		`{"time":"25:00"}`,
	); got.StatusCode != http.StatusBadRequest {
		t.Errorf("an impossible time answered %d, want 400", got.StatusCode)
	}
	if got := call(http.MethodDelete, "/"+added.ID, ""); got.StatusCode != http.StatusNoContent {
		t.Fatalf("deleting answered %d, want 204", got.StatusCode)
	}
	if got := call(
		http.MethodDelete,
		"/"+added.ID,
		"",
	); got.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("deleting it again answered %d, want 422", got.StatusCode)
	}
}
