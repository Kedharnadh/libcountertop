package releases

import (
	"slices"
	"strings"
	"testing"
)

const feed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Release notes from echolocal</title>
  <entry>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/boot-images"/>
  </entry>
  <entry>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/dev"/>
  </entry>
  <entry>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/0.0.11-dev.2"/>
  </entry>
  <entry>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/0.0.10"/>
  </entry>
</feed>`

func TestTheNewestReleaseIsTheFirstVersionTag(t *testing.T) {
	got, err := Parse(strings.NewReader(feed))
	if err != nil || got != "0.0.11-dev.2" {
		t.Fatalf("Parse = %q, %v; want 0.0.11-dev.2", got, err)
	}
}

func TestAFeedWithNoReleaseIsAnError(t *testing.T) {
	if _, err := Parse(
		strings.NewReader(`<feed xmlns="http://www.w3.org/2005/Atom"></feed>`),
	); err == nil {
		t.Fatal("an empty feed gave a release")
	}
	if _, err := Parse(strings.NewReader("not xml")); err == nil {
		t.Fatal("a broken feed gave a release")
	}
}

const list = `[
  {"tag_name": "0.0.12-dev.3", "draft": true, "prerelease": true, "assets": [{"name": "manifest.json"}]},
  {"tag_name": "0.0.12-dev.2", "prerelease": true, "assets": [{"name": "lanovod"}]},
  {"tag_name": "0.0.12-dev.1", "prerelease": true, "assets": [{"name": "manifest.json"}]},
  {"tag_name": "0.0.11", "assets": [{"name": "manifest.json"}]},
  {"tag_name": "0.0.11-dev.9", "prerelease": true, "assets": [{"name": "manifest.json"}]},
  {"tag_name": "boot-images", "prerelease": true, "assets": [{"name": "manifest.json"}]},
  {"tag_name": "0.0.10", "assets": [{"name": "manifest.json"}]},
  {"tag_name": "0.0.9", "assets": [{"name": "manifest.json"}]}
]`

func TestTheListKeepsPublishedVersionsWithAManifest(t *testing.T) {
	got, err := ParseList(strings.NewReader(list), 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []Release{
		{Tag: "0.0.12-dev.1", Prerelease: true},
		{Tag: "0.0.11"},
		{Tag: "0.0.11-dev.9", Prerelease: true},
		{Tag: "0.0.10"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ParseList = %v, want %v", got, want)
	}
}

func TestAManifestLivesUnderItsTag(t *testing.T) {
	got := Manifest("https://github.com/ygelfand/LANovo/", "0.0.4")
	if want := "https://github.com/ygelfand/LANovo/releases/download/0.0.4/manifest.json"; got != want {
		t.Errorf("Manifest = %q, want %q", got, want)
	}
}
