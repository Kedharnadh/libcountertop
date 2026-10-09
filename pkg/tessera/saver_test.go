package tessera

import (
	"encoding/json"
	"testing"
)

func TestSaverDecodes(t *testing.T) {
	shown := func(raw string) (Saver, bool) { return Layout{Saver: json.RawMessage(raw)}.Shown() }

	if _, ok := shown(``); ok {
		t.Error("no saver op showed something")
	}
	if _, ok := shown(`{"k":""}`); ok {
		t.Error("the empty choice showed something")
	}
	if _, ok := shown(`{"k":"media"}`); ok {
		t.Error("media without an entity showed something")
	}

	m, ok := shown(
		`{"k":"media","e":"media_player.kitchen","n":"Kitchen","t":"Song","s":"playing",` +
			`"f":16385,"m":1,"x":{"artist":"Band","album":"Record","pic":"ab12"}}`,
	)
	if !ok || m.Title != "Song" || m.Artist != "Band" || m.Album != "Record" || m.Mark != "ab12" ||
		!m.Playing || !m.Muted || m.Features != 16385 {
		t.Errorf("media %+v", m)
	}

	c, ok := shown(
		`{"k":"clock","w":"21°","wi":[{"k":"text","t":"Bins out"},{"k":"ago","e":1700000000}]}`,
	)
	if !ok || c.Temperature != "21°" || len(c.Items) != 2 {
		t.Errorf("clock %+v", c)
	}

	c, ok = shown(`{"k":"clock","wi":[{"k":"text","t":"ok"},{"k":"wifi"}]}`)
	if !ok || len(c.Items) != 0 {
		t.Errorf("a row with another kind kept items: %+v", c)
	}
}
