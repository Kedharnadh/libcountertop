package setting

import "testing"

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
