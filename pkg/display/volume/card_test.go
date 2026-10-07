package volume

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestCardSelectionAndOldExpiryCannotHideRefreshedCard(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	c := NewCard(stack)
	defer c.Close()
	c.Show(schema.StreamVoice)
	old := c.generation
	c.Show(schema.StreamMedia)
	c.expire(old)
	if stream, up := c.Selected(); !up || stream != schema.StreamMedia {
		t.Fatal("old expiry hid refreshed card", stream, up)
	}
	c.Expand()
	if _, open := c.Picked(); !open {
		t.Fatal("card did not expand")
	}
	c.Pick(schema.StreamAlerts)
	if stream, open := c.Picked(); stream != schema.StreamAlerts || open {
		t.Fatal("selection did not fold the card")
	}
	c.Close()
	if _, up := c.Selected(); up {
		t.Fatal("closed card retained its hold")
	}
}
