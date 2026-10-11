package mediacard

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/media/nowplaying"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

func TestTheAppOpensTheCardWithNothingPlaying(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	c := New(Dependencies{Player: testPlayer{&testSource{}}, Idle: testIdle{}, Shell: stack})

	app := c.App()
	if app.ID != schema.AppPlayer || app.Label() == "" || app.Glyph() == "" {
		t.Fatalf("app %+v", app)
	}
	app.Open()
	if stack.Top() != c.Page() {
		t.Error("the app did not put the card up")
	}
}

func TestTheAppOpensWhatIsPlaying(t *testing.T) {
	stack := shell.New()
	defer stack.Close()
	src := &testSource{now: nowplaying.Now{Playing: true}}
	c := New(Dependencies{Player: testPlayer{src}, Idle: testIdle{}, Shell: stack})
	c.Began(src)

	c.App().Open()
	if stack.Top() != c.Page() {
		t.Error("the app did not open the playing card")
	}
}
