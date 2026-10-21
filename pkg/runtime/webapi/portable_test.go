package webapi

import (
	"encoding/json"
	"slices"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"

	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

type pane struct {
	domain string
	values map[string]string
	local  map[string]bool
}

func (p *pane) Domain() string { return p.domain }

func (p *pane) Names() []string {
	var out []string
	for n := range p.values {
		out = append(out, n)
	}
	slices.Sort(out)
	return out
}

func (p *pane) Value(name string) (string, error) { return p.values[name], nil }

func (p *pane) Change(name, value string) error { p.values[name] = value; return nil }

func (p *pane) Fields() []setting.Field {
	var out []setting.Field
	for _, n := range p.Names() {
		out = append(
			out,
			setting.Field{
				Name:  n,
				Title: "Title " + n,
				Group: "Group",
				Kind:  "text",
				Value: p.values[n],
			},
		)
	}
	return out
}

func (p *pane) Export() map[string]string {
	out := map[string]string{}
	for n, v := range p.values {
		if !p.local[n] {
			out[n] = v
		}
	}
	return out
}

func (p *pane) Import(values map[string]string) (int, error) {
	changed := 0
	for n, v := range values {
		if _, ok := p.values[n]; ok && !p.local[n] && p.values[n] != v {
			p.values[n] = v
			changed++
		}
	}
	return changed, nil
}

func portableFixture(t *testing.T) (*API, *pane) {
	t.Helper()
	a, _, _, _ := fixture(t)
	screen := &pane{
		domain: "screen",
		values: map[string]string{"theme": "light", "style": "standard", "size": "medium"},
		local:  map[string]bool{"size": true},
	}
	a.o.Registry.Add(screen)
	return a, screen
}

func action(t *testing.T, a *API, name string) *esphome.Action {
	t.Helper()
	for _, act := range a.Actions() {
		if act.Name == name {
			return act
		}
	}
	t.Fatalf("no %s action", name)
	return nil
}

func call(t *testing.T, args map[string]any) esphome.Call {
	t.Helper()
	c, err := esphome.NewCall(args)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestExportListsOnlyPortableSettingsWithTheirSection(t *testing.T) {
	a, _ := portableFixture(t)
	got, err := action(t, a, "settings_export").Run(call(t, nil))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range got.(map[string]any)["settings"].([]exported) {
		if e.Domain != "screen" {
			continue
		}
		names = append(names, e.Name)
		if e.Section == "" || e.Title != "Title "+e.Name {
			t.Errorf("%s came without its section or title: %+v", e.Name, e)
		}
	}
	slices.Sort(names)
	if want := []string{"style", "theme"}; !slices.Equal(names, want) {
		t.Errorf("exported %v, want %v (size is local)", names, want)
	}
}

func TestImportAppliesWhatItCanAndSaysWhatItSkipped(t *testing.T) {
	a, screen := portableFixture(t)
	body, _ := json.Marshal(setting.Snapshot{
		"screen":  {"theme": "dark", "style": "standard", "size": "large", "gone": "x"},
		"nowhere": {"thing": "1"},
	})
	got, err := action(
		t,
		a,
		"settings_import",
	).Run(call(t, map[string]any{"settings": string(body)}))
	if err != nil {
		t.Fatal(err)
	}
	r := got.(imported)
	if r.Changed != 1 || screen.values["theme"] != "dark" {
		t.Errorf("changed %d, theme %q; want 1 change to dark", r.Changed, screen.values["theme"])
	}
	if screen.values["size"] != "medium" {
		t.Errorf("a local setting was written: size %q", screen.values["size"])
	}
	why := map[string]string{}
	for _, s := range r.Skipped {
		why[s.Domain+"."+s.Name] = s.Why
	}
	want := map[string]string{
		"screen.size":   skippedLocal,
		"screen.gone":   skippedMissing,
		"nowhere.thing": skippedMissing,
	}
	for k, v := range want {
		if why[k] != v {
			t.Errorf("%s skipped as %q, want %q", k, why[k], v)
		}
	}
	if len(why) != len(want) {
		t.Errorf("skipped %v, want exactly %v", why, want)
	}
}

func TestImportRefusesSomethingThatIsNotASnapshot(t *testing.T) {
	a, _ := portableFixture(t)
	if _, err := action(
		t,
		a,
		"settings_import",
	).Run(call(t, map[string]any{"settings": "not json"})); err == nil {
		t.Error("an unreadable import was accepted")
	}
}
