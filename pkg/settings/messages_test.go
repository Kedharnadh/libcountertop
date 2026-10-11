package setting

import "testing"

func TestFilteredTableKeepsMetadataAndOnlyRetainedAliases(t *testing.T) {
	full := NewTable("call", []Group{"Calls"}, []Setting[bool]{
		{Name: "incoming", Group: "Calls", ID: "call_incoming"},
		{Name: "stream", Group: "Calls"},
	}, Messages{Text: func(key string) string { return key }}).
		Alias("accept", "incoming").Alias("video", "stream")
	filtered := full.Without("stream")
	if len(full.Rows()) != 2 || len(filtered.Rows()) != 1 {
		t.Fatal("filter changed the source table or retained the excluded row")
	}
	if _, ok := filtered.Find("video"); ok {
		t.Fatal("alias still reaches an excluded setting")
	}
	row, ok := filtered.Find("accept")
	if !ok || row.ID != "call_incoming" || row.Named() != "call.setting.incoming" {
		t.Fatalf("retained row lost metadata: %+v", row)
	}
}

func TestTablesKeepTheirCatalogAndResolveLanguageAtReadTime(t *testing.T) {
	language := "one:"
	a := NewTable(
		"camera",
		[]Group{"Picture"},
		[]Setting[bool]{{Name: "enabled", Kind: Toggle}},
		Messages{
			Text:    func(key string) string { return language + key },
			Missing: func(string) bool { return true },
		},
	)
	b := NewTable(
		"screen",
		[]Group{"Panel"},
		[]Setting[bool]{{Name: "enabled", Kind: Toggle}},
		Messages{
			Text:    func(key string) string { return "other:" + key },
			Missing: func(string) bool { return true },
		},
	)
	if got := a.MustRow("enabled").Named(); got != "one:camera.setting.enabled" {
		t.Fatal(got)
	}
	language = "two:"
	if got := a.MustRow("enabled").Title(); got != "two:camera.setting.enabled" {
		t.Fatal(got)
	}
	if got := b.Title("Panel"); got != "other:screen.group.panel" {
		t.Fatal(got)
	}
}
