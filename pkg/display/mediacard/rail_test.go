package mediacard

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/drawer"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	media "github.com/ygelfand/libcountertop/pkg/media/source"
)

func TestTheRailOpensTheCardWithNothingPlaying(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	c := New(Dependencies{Player: testPlayer{&testSource{}}, Idle: testIdle{}, Shell: stack})

	rail := c.Rail()
	if rail.Order != drawer.OrderPlayer || rail.Name() == "" || rail.Glyph() == "" {
		t.Fatalf("rail entry %+v", rail)
	}
	rail.Open()
	if stack.Top() != c.Page() {
		t.Error("the rail did not put the card up")
	}
}

func TestTheRailOpensWhatIsPlaying(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	src := &testSource{now: media.Now{Playing: true}}
	c := New(Dependencies{Player: testPlayer{src}, Idle: testIdle{}, Shell: stack})
	c.Began(src)

	c.Rail().Open()
	if stack.Top() != c.Page() {
		t.Error("the rail did not open the playing card")
	}
}
