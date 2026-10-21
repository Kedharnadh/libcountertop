package homeassistant

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEveryDomainWithoutAFilter(t *testing.T) {
	tmpl := Filter{}.Template()
	if strings.Contains(tmpl, "selectattr") || !strings.Contains(tmpl, "for s in states -%}") {
		t.Errorf("an empty filter should walk every state:\n%s", tmpl)
	}
}

func TestParseEntitiesSortsByAreaThenName(t *testing.T) {
	list := `[{"id":"light.z","name":"zed","area":""},{"id":"light.b","name":"Bee","area":"kitchen"},{"id":"light.a","name":"Ay","area":"Kitchen"},{"id":"light.c","name":"Cee","area":"Attic"}]`
	quoted, _ := json.Marshal(list)
	for _, raw := range []string{list, string(quoted)} {
		got, err := ParseEntities(json.RawMessage(raw))
		if err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, e := range got {
			ids = append(ids, e.ID)
		}
		if strings.Join(ids, " ") != "light.c light.a light.b light.z" {
			t.Errorf("order %v", ids)
		}
	}
}

func TestAskedAttributesAreInTheTemplate(t *testing.T) {
	if tmpl := (Filter{Attributes: []string{"user_id"}}).Template(); !strings.Contains(
		tmpl,
		`["user_id"]`,
	) {
		t.Errorf("the template does not ask for user_id:\n%s", tmpl)
	}
}

func TestParseKeepsAttributes(t *testing.T) {
	got, err := ParseEntities(json.RawMessage(
		`[{"id":"person.ada","name":"Ada","attributes":{"user_id":"u1"}},{"id":"person.bo","name":"Bo","attributes":{}}]`,
	))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Attributes["user_id"] != "u1" || got[1].Attributes["user_id"] != "" {
		t.Errorf("parsed %+v", got)
	}
}

func TestParseEmptyIsNotNil(t *testing.T) {
	got, err := ParseEntities(json.RawMessage(`"[]"`))
	if err != nil || got == nil {
		t.Errorf("got %v, %v; want an empty list", got, err)
	}
}
