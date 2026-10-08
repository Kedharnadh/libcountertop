package schema

import "testing"

func TestClonesOwnEveryMutableCollection(t *testing.T) {
	w := Wake{Words: []WakeWord{{ID: "word"}}}
	c := Cast{YouTube: YouTube{Skip: []string{"sponsor"}}}
	cam := Camera{Settings: map[string]string{"scene": "auto"}}
	h := Home{
		Control: map[string]bool{"x": true},
		Group:   map[string]int{"x": 2},
		Picks: map[string]HomePick{
			"x": {Labels: []string{"label"}, Areas: []string{"area"}, Entities: []string{"entity"}},
		},
	}

	w2, c2, cam2, h2 := w.Clone(), c.Clone(), cam.Clone(), h.Clone()
	w2.Words[0].ID = "changed"
	c2.YouTube.Skip[0] = "changed"
	cam2.Settings["scene"] = "changed"
	h2.Control["x"] = false
	h2.Group["x"] = 4
	pick := h2.Picks["x"]
	pick.Labels[0] = "changed"
	pick.Areas[0] = "changed"
	pick.Entities[0] = "changed"

	if w.Words[0].ID != "word" || c.YouTube.Skip[0] != "sponsor" ||
		cam.Settings["scene"] != "auto" || !h.Control["x"] || h.Group["x"] != 2 {
		t.Fatal("a clone shares a collection with the original")
	}
	if p := h.Picks["x"]; p.Labels[0] != "label" || p.Areas[0] != "area" ||
		p.Entities[0] != "entity" {
		t.Fatal("a cloned pick shares its lists with the original", p)
	}
}
