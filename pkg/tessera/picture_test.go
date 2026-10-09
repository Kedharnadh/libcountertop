package tessera

import (
	"testing"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
)

func TestOnlyTheLatestPictureLinkIsKept(t *testing.T) {
	c := committed(t)
	var told []component.Event
	c.s.Told.Listen(func(e component.Event) { told = append(told, e) })
	c.s.AskPicture(PictureFull, map[string]string{"entity": "camera.door"})
	c.s.AskPicture(PictureFull, map[string]string{"entity": "camera.door"})
	if len(told) != 2 || told[1].Name != "esphome.screen_camera" || told[1].Data["view"] != "2" ||
		told[1].Data["entity"] != "camera.door" {
		t.Fatalf("asked %+v", told)
	}
	answer := func(view int, url string) {
		c.send(
			map[string]any{"op": "camera", "t": "full", "e": "camera.door", "u": url, "view": view},
		)
	}
	answer(1, "http://10.0.0.2/old.bmp")
	if _, _, ok := c.s.Picture(PictureFull); ok {
		t.Fatal("a link for an older question was kept")
	}
	answer(2, "https://10.0.0.2/new.bmp")
	if _, _, ok := c.s.Picture(PictureFull); ok {
		t.Fatal("a link that is not http was kept")
	}
	answer(2, "http://10.0.0.2/new.bmp")
	entity, url, ok := c.s.Picture(PictureFull)
	if !ok || entity != "camera.door" || url != "http://10.0.0.2/new.bmp" {
		t.Errorf("picture %q %q %v", entity, url, ok)
	}
}
