package radio

import (
	"encoding/json"
	"testing"
)

func TestParse(t *testing.T) {
	raw := json.RawMessage(`{"t": "Popular", "n": 3, "i": [
		["Jazz FM", "media-source://radio_browser/abc", true, false, "https://x/logo.png"],
		["By Category", "media-source://radio_browser/tag", false, true, ""],
		["broken", "", true, false, ""],
		["short"]
	]}`)
	got, err := parse(raw)
	if err != nil {
		t.Fatalf("parse gave %v", err)
	}
	want := Page{Title: "Popular", Total: 3, Items: []Item{
		{
			Title:    "Jazz FM",
			ID:       "media-source://radio_browser/abc",
			Playable: true,
			Thumb:    "https://x/logo.png",
		},
		{Title: "By Category", ID: "media-source://radio_browser/tag", Expandable: true},
	}}
	if got.Title != want.Title || got.Total != want.Total || len(got.Items) != len(want.Items) {
		t.Fatalf("parse = %+v, want %+v", got, want)
	}
	for i := range want.Items {
		if got.Items[i] != want.Items[i] {
			t.Errorf("item %d = %+v, want %+v", i, got.Items[i], want.Items[i])
		}
	}
	if _, err := parse(json.RawMessage(`[]`)); err == nil {
		t.Error("parse of a list gave no error")
	}
}
