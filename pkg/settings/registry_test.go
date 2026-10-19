package setting

import (
	"errors"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"
)

type sample struct {
	On    bool
	Mode  string
	Name  string
	Token string
}

func sampleControls(t *testing.T, saved *sample) *Controls[sample] {
	t.Helper()
	table := NewTable("sample", []Group{"All"}, []Setting[sample]{
		{
			Name: "on", Group: "All", Kind: Toggle,
			Read: func(s *sample) string { return OnOff(s.On) },
			Write: func(s *sample, v string) error {
				on, ok := Boolean(v)
				if !ok {
					return errors.New("want on or off")
				}
				s.On = on
				return nil
			},
		},
		{
			Name: "mode", Group: "All", Kind: Choice,
			Options: []Option{{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}},
			Read:    func(s *sample) string { return s.Mode },
			Write: func(s *sample, v string) error {
				if v != "a" && v != "b" {
					return errors.New("want a or b")
				}
				s.Mode = v
				return nil
			},
		},
		{
			Name: "name", Group: "All", Kind: Text,
			Read:  func(s *sample) string { return s.Name },
			Write: func(s *sample, v string) error { s.Name = v; return nil },
		},
		{
			Name: "token", Group: "All", Kind: Secret,
			Read:  func(s *sample) string { return s.Token },
			Write: func(s *sample, v string) error { s.Token = v; return nil },
		},
	}, Messages{})
	return &Controls[sample]{
		Table: table,
		Read:  func() sample { return *saved },
		Save: func(s Setting[sample], v string) error {
			next := *saved
			if err := s.Write(&next, v); err != nil {
				return err
			}
			*saved = next
			return nil
		},
	}
}

func TestBoundChangesThroughTheSavePath(t *testing.T) {
	saved := &sample{Mode: "a"}
	c := sampleControls(t, saved)

	if err := c.Change("on", "on"); err != nil || !saved.On {
		t.Fatalf("Change(on) = %v, saved %+v", err, *saved)
	}
	if err := c.Change("mode", "Beta"); err != nil || saved.Mode != "b" {
		t.Errorf("a choice by label: %v, saved %+v", err, *saved)
	}
	if err := c.Change("name", ""); err != nil {
		t.Errorf("clearing a text setting: %v", err)
	}
	if err := c.Change("on", "maybe"); err == nil || !saved.On {
		t.Errorf("a bad value was accepted: %v, saved %+v", err, *saved)
	}
	if err := c.Change("nonesuch", "x"); err == nil {
		t.Error("an unknown setting was accepted")
	}
}

func TestSecretsAreNotShown(t *testing.T) {
	saved := &sample{Token: "hunter2"}
	c := sampleControls(t, saved)
	if v, _ := c.Value("token"); v != "set" {
		t.Errorf("Value(token) = %q", v)
	}
	saved.Token = ""
	if v, _ := c.Value("token"); v != "" {
		t.Errorf("an empty secret shows %q", v)
	}
}

func TestTextSettingsBecomeTextEntities(t *testing.T) {
	saved := &sample{Name: "kitchen", Token: "k"}
	c := sampleControls(t, saved)
	var texts int
	for _, e := range c.Entities() {
		if _, ok := e.(*esphome.Text); ok {
			texts++
		}
	}
	if texts != 2 {
		t.Errorf("%d text entities, want 2", texts)
	}
}

func TestRegistryListsDomainsInOrder(t *testing.T) {
	var r Registry
	b := sampleControls(t, &sample{})
	a := &Controls[sample]{Table: NewTable[sample]("alpha", nil, nil, Messages{})}
	r.Add(b, a)
	got := r.All()
	if len(got) != 2 || got[0].Domain() != "alpha" || got[1].Domain() != "sample" {
		t.Errorf("All() = %v", got)
	}
}

func TestASnapshotCarriesOnlyPortableSettings(t *testing.T) {
	from := sample{On: true, Mode: "b", Name: "kitchen", Token: "s3cret"}
	source := sampleControls(t, &from)
	source.Table.rows[2].Local = true
	var r Registry
	r.Add(source)

	snap := r.Export()
	want := map[string]string{"on": "on", "mode": "b", "token": "s3cret"}
	if got := snap["sample"]; len(got) != len(want) || got["on"] != "on" || got["mode"] != "b" ||
		got["token"] != "s3cret" {
		t.Fatalf("snapshot = %v, want %v", got, want)
	}

	var to sample
	target := sampleControls(t, &to)
	target.Table.rows[2].Local = true
	var into Registry
	into.Add(target)
	snap["sample"]["name"] = "hallway"
	snap["sample"]["gone"] = "x"
	snap["other"] = map[string]string{"on": "off"}

	applied, err := into.Import(snap)
	if err != nil || applied != 3 {
		t.Fatalf("Import = %d, %v; want 3 settings applied", applied, err)
	}
	if !to.On || to.Mode != "b" || to.Name != "" || to.Token != "s3cret" {
		t.Fatalf("imported %+v; want on, mode and token", to)
	}

	if applied, err := into.Import(snap); err != nil || applied != 0 {
		t.Fatalf("second Import = %d, %v; want nothing left to change", applied, err)
	}
}
