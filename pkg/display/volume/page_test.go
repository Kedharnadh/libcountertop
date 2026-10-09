package volume

import (
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

type pageLevels map[schema.Stream]int

func (l pageLevels) Level(s schema.Stream) int  { return l[s] }
func (l pageLevels) Set(s schema.Stream, v int) { l[s] = v }

type pageChimes struct {
	mu      sync.Mutex
	chime   schema.Chime
	preview []schema.Stream
}

func (c *pageChimes) Chime() schema.Chime     { return c.chime }
func (c *pageChimes) SetChime(v schema.Chime) { c.chime = v }
func (c *pageChimes) Preview(s schema.Stream) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.preview = append(c.preview, s)
}
func (c *pageChimes) heard() []schema.Stream {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]schema.Stream(nil), c.preview...)
}

func TestSoundPageHasEveryStreamAndTheChime(t *testing.T) {
	levels := pageLevels{schema.StreamMain: 50, schema.StreamMedia: 40}
	chimes := &pageChimes{chime: schema.ChimeDing}
	streams := []schema.Stream{schema.StreamMain, schema.StreamMedia, schema.StreamAlerts}
	rows, actions := NewPage(PageOptions{
		Levels: levels, Streams: streams, MainSteps: 20, Chimes: chimes,
	}).Build()

	if len(rows) != len(streams)+1 || len(actions) != len(rows) {
		t.Fatalf("%d rows and %d actions", len(rows), len(actions))
	}
	if rows[0].Kind != menu.Slider || rows[0].Level != 50 || rows[0].Snap == nil {
		t.Errorf("main row %+v", rows[0])
	}
	if got := rows[0].Snap(52); got != 50 {
		t.Errorf("main snaps 52 to %d, want 50", got)
	}
	if rows[1].Snap != nil {
		t.Error("media snaps")
	}
	chime := rows[len(rows)-1]
	if chime.Kind != menu.Segments || len(chime.Options) != len(schema.Chimes()) ||
		schema.Chimes()[chime.Level] != schema.ChimeDing {
		t.Errorf("chime row %+v", chime)
	}

	actions[len(actions)-1](0)
	if chimes.chime != schema.Chimes()[0] {
		t.Errorf("picked %s", chimes.chime)
	}
}

func TestSoundPagePreviewsTheStreamOnceItSettles(t *testing.T) {
	levels := pageLevels{}
	chimes := &pageChimes{}
	_, actions := NewPage(PageOptions{
		Levels: levels, Streams: []schema.Stream{schema.StreamAlerts}, Chimes: chimes,
	}).Build()

	for v := 10; v <= 30; v += 10 {
		actions[0](v)
	}
	if levels[schema.StreamAlerts] != 30 {
		t.Errorf("alerts at %d", levels[schema.StreamAlerts])
	}
	if got := chimes.heard(); len(got) != 0 {
		t.Errorf("previewed %v while still moving", got)
	}

	deadline := time.Now().Add(5 * settle)
	for len(chimes.heard()) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if got := chimes.heard(); len(got) != 1 || got[0] != schema.StreamAlerts {
		t.Errorf("previewed %v, want alerts once", got)
	}
}
