package castreceiver

import (
	"strconv"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestAnEmptySkipListIsSavedEmpty(t *testing.T) {
	c := schema.DefaultCast()
	if err := NewTable().MustRow("youtube.skip").Write(&c, ""); err != nil {
		t.Fatal(err)
	}
	if c.YouTube.Skip == nil || len(c.YouTube.Skip) != 0 {
		t.Fatalf("skip is %#v", c.YouTube.Skip)
	}
}

func TestALiveDelayOutOfRangeIsRefused(t *testing.T) {
	c := schema.DefaultCast()
	row := NewTable().MustRow("youtube.livedelay")
	if err := row.Write(&c, strconv.Itoa(schema.LiveDelayMost+1)); err == nil {
		t.Error("a live delay past the most was accepted")
	}
	if err := row.Write(
		&c,
		strconv.Itoa(schema.LiveDelayMost),
	); err != nil ||
		c.YouTube.LiveDelay != schema.LiveDelayMost {
		t.Errorf("the most: %v, %d", err, c.YouTube.LiveDelay)
	}
}
