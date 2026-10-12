package radar

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestKeptTilesLiveOnDisk(t *testing.T) {
	dir := t.TempDir()
	newCache(dir).put("https://tiles/base/1/2/3", []byte("jpeg"), true, time.Time{})

	got, ok := newCache(dir).get("https://tiles/base/1/2/3", true)
	if !ok || !bytes.Equal(got, []byte("jpeg")) {
		t.Errorf("a fresh cache read %q, %v; want the kept tile", got, ok)
	}
}

func TestExpiringTilesLiveOnDiskUntilTheyExpire(t *testing.T) {
	dir := t.TempDir()
	c := newCache(dir)
	c.put("live", []byte("png"), false, time.Now().Add(time.Hour))
	c.put("stale", []byte("png"), false, time.Now().Add(-time.Second))

	fresh := newCache(dir)
	if got, ok := fresh.get("live", false); !ok || !bytes.Equal(got, []byte("png")) {
		t.Errorf("a restarted cache read %q, %v; want the live tile", got, ok)
	}
	if _, ok := fresh.get("stale", false); ok {
		t.Error("a restarted cache served an expired tile")
	}
	if _, err := os.Stat(fresh.path("stale", false)); !os.IsNotExist(err) {
		t.Errorf("the expired tile is still on disk: %v", err)
	}
}

func TestKeptAndExpiringTilesDoNotShareFiles(t *testing.T) {
	c := newCache(t.TempDir())
	if c.path("u", true) == c.path("u", false) {
		t.Error("kept and expiring tiles share a path")
	}
}

func TestMemoryTilesExpire(t *testing.T) {
	c := newCache("")
	c.put("live", []byte("png"), false, time.Now().Add(time.Hour))
	c.put("stale", []byte("png"), false, time.Now().Add(-time.Second))

	if _, ok := c.get("live", false); !ok {
		t.Error("a live tile was not served")
	}
	if _, ok := c.get("stale", false); ok {
		t.Error("an expired tile was served")
	}
}

func TestMemoryStaysUnderItsCap(t *testing.T) {
	c := newCache("")
	tile := make([]byte, 1<<20)
	until := time.Now().Add(time.Hour)
	for i := range 40 {
		c.put(string(rune('a'+i)), tile, false, until)
	}
	if c.bytes > memoryMost {
		t.Errorf("cache holds %d bytes, cap %d", c.bytes, memoryMost)
	}
	if _, ok := c.get(string(rune('a'+39)), false); !ok {
		t.Error("the newest tile was evicted")
	}
	if _, ok := c.get("a", false); ok {
		t.Error("the oldest tile survived past the cap")
	}
}
