package status

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/runtime/startup"
	"github.com/ygelfand/libcountertop/pkg/system/metrics"
)

func page() Page {
	return Page{
		Name:   "Kitchen",
		Model:  "Panel",
		Parts:  []Part{{"wifi", "ready", ""}},
		Groups: []Group{{Title: "Time", Rows: []Row{{"Zone", "UTC"}}}},
	}
}

func TestTheRootIsTheStatusPage(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(page)(rec, httptest.NewRequest(http.MethodGet, "http://10.0.0.5/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<!doctype html>") {
		t.Error("the root did not render a page")
	}
	if !strings.Contains(body, "Kitchen") {
		t.Error("the page does not carry the device name")
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
}

func TestStatusRefusesAnythingElse(t *testing.T) {
	rec := httptest.NewRecorder()
	Handler(page)(rec, httptest.NewRequest(http.MethodGet, "http://10.0.0.5/nonesuch", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("an unknown path gave %d, want 404", rec.Code)
	}
}

func TestComponentsReportWhatTheySayTheyAre(t *testing.T) {
	got := Parts([]startup.Progress{
		{Name: "panel", Done: true},
		{Name: "wifi", Doing: "associating"},
		{Name: "api", Failed: true, Doing: "no key"},
	})

	want := []Part{
		{"panel", "ready", ""},
		{"wifi", "waiting", "associating"},
		{"api", "failed", "no key"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestNoProgressMeansNoParts(t *testing.T) {
	if got := Parts(nil); len(got) != 0 {
		t.Errorf("got %+v, want nothing", got)
	}
}

func TestTheStatusPageCarriesNoSecrets(t *testing.T) {
	for _, field := range reflect.VisibleFields(reflect.TypeFor[Page]()) {
		switch strings.ToLower(field.Name) {
		case "key", "secret", "password", "passphrase", "token", "psk":
			t.Errorf("Page carries a %s", field.Name)
		}
	}

	for _, banned := range []string{".Key", ".Secret", ".Password", ".Token", ".PSK"} {
		if strings.Contains(statusHTML, banned) {
			t.Errorf("the template renders %s", banned)
		}
	}
}

func TestTheZoneSaysWhereItCameFrom(t *testing.T) {
	tests := []struct {
		name           string
		chosen, home   string
		wantZone, want string
	}{
		{
			"chosen wins",
			"Europe/Berlin",
			"EST5EDT,M3.2.0,M11.1.0",
			"Europe/Berlin",
			"Set on the device",
		},
		{
			"home otherwise",
			"",
			"EST5EDT,M3.2.0,M11.1.0",
			"EST5EDT,M3.2.0,M11.1.0",
			"From Home Assistant",
		},
		{"neither", "", "", Unknown, "Nothing has said"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := rowsOf(Clock(tt.chosen, tt.home))
			if got := rows["Zone"]; got != tt.wantZone {
				t.Errorf("zone = %q, want %q", got, tt.wantZone)
			}
			if got := rows["Source"]; got != tt.want {
				t.Errorf("source = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOnlyTheNamedTemperaturesAreShown(t *testing.T) {
	root := t.TempDir()
	zone(t, root, 0, "apc0-cpu0-usr", "33400")
	zone(t, root, 1, "deca-cpu-max-step", "35500")
	zone(t, root, 2, "gpu0-usr", "31900")

	got := temperatures(
		metrics.Reader{Root: root},
		[]Thermal{{"CPU", "deca-cpu-max-step"}, {"GPU", "gpu0-usr"}},
	)
	want := []Row{{"CPU", "35.5 °C"}, {"GPU", "31.9 °C"}}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestMemoryIsReportedFromKilobytes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proc"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(root, "proc/meminfo"),
		"MemTotal:        2027056 kB\nMemAvailable:     1648992 kB")

	got := rowsOf(System(metrics.Reader{Root: root}, root, nil))["Memory free"]
	if want := "1.6 GiB of 1.9 GiB"; got != want {
		t.Errorf("memory = %q, want %q", got, want)
	}
}

func TestNoThermalZonesMeansNoRows(t *testing.T) {
	got := temperatures(metrics.Reader{Root: t.TempDir()}, []Thermal{{"CPU", "deca-cpu-max-step"}})
	if got != nil {
		t.Errorf("got %+v, want nothing", got)
	}
}

func zone(t *testing.T, root string, n int, name, milli string) {
	t.Helper()

	dir := filepath.Join(root, "sys/class/thermal/thermal_zone"+strconv.Itoa(n))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(dir, "type"), name)
	write(t, filepath.Join(dir, "temp"), milli)
}

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUptimeReadsAtEverySize(t *testing.T) {
	tests := []struct {
		seconds float64
		want    string
	}{
		{7, "7s"},
		{95, "1m 35s"},
		{3700, "1h 1m"},
		{100000, "1d 3h"},
		{1000000, "11d 13h"},
	}

	for _, tt := range tests {
		if got := since(tt.seconds); got != tt.want {
			t.Errorf("since(%v) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestSizesPickTheUnitTheyFill(t *testing.T) {
	tests := []struct {
		v    float64
		want string
	}{
		{512, "512 B"},
		{2048, "2.0 KiB"},
		{5 * 1024 * 1024, "5.0 MiB"},
		{3 * 1024 * 1024 * 1024, "3.0 GiB"},
	}

	for _, tt := range tests {
		if got := bytes(tt.v); got != tt.want {
			t.Errorf("bytes(%v) = %q, want %q", tt.v, got, tt.want)
		}
	}
}

func TestOffsetsReadAsUTC(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, "UTC+00:00"},
		{3600, "UTC+01:00"},
		{-18000, "UTC-05:00"},
		{19800, "UTC+05:30"},
		{-34200, "UTC-09:30"},
	}

	for _, tt := range tests {
		if got := utc(tt.seconds); got != tt.want {
			t.Errorf("utc(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestAMissingReadingSaysSo(t *testing.T) {
	if got := or("   "); got != Unknown {
		t.Errorf("or(blank) = %q, want %q", got, Unknown)
	}
	if got := or("kitchen"); got != "kitchen" {
		t.Errorf("or(%q) = %q, want it unchanged", "kitchen", got)
	}
}

func rowsOf(g Group) map[string]string {
	out := map[string]string{}
	for _, r := range g.Rows {
		out[r.Name] = r.Value
	}
	return out
}
