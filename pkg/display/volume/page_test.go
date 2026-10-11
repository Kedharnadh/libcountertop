package volume

import (
	"sync"
	"testing"
	"time"

	"github.com/ygelfand/libcountertop/pkg/audio/tone"
	"github.com/ygelfand/libcountertop/pkg/audio/tones"
	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
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

type feedbackState struct{ at schema.Feedback }

func (s *feedbackState) Read() schema.Feedback { return s.at }
func (s *feedbackState) Update(fn func(*schema.Feedback)) error {
	fn(&s.at)
	return nil
}

type feedbackPlayer struct{ played tone.Sound }

func (*feedbackPlayer) Chime(schema.Stream, []tone.Note)                     {}
func (p *feedbackPlayer) Sound(_ schema.Stream, _ float64, sound tone.Sound) { p.played = sound }

func TestSoundPageOffersAndPreviewsUploadedFeedback(t *testing.T) {
	store := tones.New(tones.Options{Dir: t.TempDir(), Rate: tones.Rate, Channels: 1})
	at := &feedbackState{at: schema.DefaultFeedback()}
	player := &feedbackPlayer{}
	feedback := tone.NewFeedback(at, player, 0, tone.FeedbackOptions{Sounds: store, Mute: true})
	var opened shell.View
	page := NewPage(PageOptions{Feedback: feedback, Push: func(v shell.View) { opened = v }})
	rows, actions := page.Build()
	if len(rows) != 5 {
		t.Fatalf("sound page has %d feedback rows, want 5", len(rows))
	}
	// Upload after the page is created: each picker must read the latest choices.
	if err := store.Write("bell", "Uploaded bell", 0, 1, []byte{1, 0, 2, 0}); err != nil {
		t.Fatal(err)
	}
	for i, field := range feedback.Table.Rows() {
		actions[i](0)
		picker, ok := opened.(*shell.Page)
		if !ok {
			t.Fatalf("%s opened %T, want a choice page", field.Name, opened)
		}
		choices, picks := picker.Build()
		index := -1
		for n, row := range choices {
			if row.Label == "Uploaded bell" {
				index = n
			}
		}
		if index < 0 {
			t.Fatalf("%s picker omits uploaded sound", field.Name)
		}
		picks[index](0)
		now := at.Read()
		if got := field.Read(&now); got != "tone:bell" {
			t.Errorf("%s saved %q", field.Name, got)
		}
		if len(player.played.Clip) != 2 {
			t.Errorf("%s preview played %+v", field.Name, player.played)
		}
	}
}
