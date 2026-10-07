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

func TestParseEmptyIsNotNil(t *testing.T) {
	got, err := ParseEntities(json.RawMessage(`"[]"`))
	if err != nil || got == nil {
		t.Errorf("got %v, %v; want an empty list", got, err)
	}
}
