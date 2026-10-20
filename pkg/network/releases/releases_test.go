package releases

import (
	"strings"
	"testing"
)

const feed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Release notes from echolocal</title>
  <entry>
    <id>tag:github.com,2008:Repository/1/boot-images</id>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/boot-images"/>
  </entry>
  <entry>
    <id>tag:github.com,2008:Repository/1/0.0.11-dev.2</id>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/0.0.11-dev.2"/>
  </entry>
  <entry>
    <id>tag:github.com,2008:Repository/1/0.0.10</id>
    <link rel="alternate" type="text/html" href="https://github.com/ygelfand/echolocal/releases/tag/0.0.10"/>
  </entry>
</feed>`

func TestTheNewestReleaseIsTheFirstThatIsNotSkipped(t *testing.T) {
	got, err := Parse(strings.NewReader(feed), Skip)
	if err != nil || got != "0.0.11-dev.2" {
		t.Fatalf("Parse = %q, %v; want 0.0.11-dev.2", got, err)
	}
	if got, _ := Parse(strings.NewReader(feed), nil); got != "boot-images" {
		t.Errorf("with nothing skipped got %q, want boot-images", got)
	}
}

func TestAFeedWithNoReleaseIsAnError(t *testing.T) {
	if _, err := Parse(
		strings.NewReader(`<feed xmlns="http://www.w3.org/2005/Atom"></feed>`),
		Skip,
	); err == nil {
		t.Fatal("an empty feed gave a release")
	}
	if _, err := Parse(strings.NewReader("not xml"), Skip); err == nil {
		t.Fatal("a broken feed gave a release")
	}
}
