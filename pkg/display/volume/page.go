package volume

import (
	"slices"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/settings/schema"
)

const settle = 400 * time.Millisecond

type Levels interface {
	Level(schema.Stream) int
	Set(schema.Stream, int)
}
type Chimes interface {
	Chime() schema.Chime
	SetChime(schema.Chime)
	Preview(schema.Stream)
}

type PageOptions struct {
	Levels    Levels
	Streams   []schema.Stream
	MainSteps int
	Chimes    Chimes
}

func NewPage(o PageOptions) *shell.Page {
	streams := slices.Clone(o.Streams)
	hear := &preview{chimes: o.Chimes}
	return &shell.Page{Title: say.T("settings.volume"), Build: func() ([]menu.Row, []func(int)) {
		var rows []menu.Row
		var actions []func(int)
		for _, s := range streams {
			row := menu.Row{
				Label: say.T("sound." + string(s)),
				Kind:  menu.Slider,
				Level: o.Levels.Level(s),
			}
			if s == schema.StreamMain && o.MainSteps > 0 {
				row.Snap = stepped(o.MainSteps)
			}
			rows = append(rows, row)
			actions = append(actions, func(level int) {
				o.Levels.Set(s, level)
				hear.after(s)
			})
		}
		if o.Chimes != nil {
			chimes := schema.Chimes()
			labels := make([]string, len(chimes))
			for i, c := range chimes {
				labels[i] = c.Label()
			}
			rows = append(rows, menu.Row{
				Label:   say.T("volume.chime"),
				Kind:    menu.Segments,
				Options: labels,
				Level:   max(slices.Index(chimes, o.Chimes.Chime()), 0),
			})
			actions = append(actions, func(i int) {
				o.Chimes.SetChime(chimes[i])
				o.Chimes.Preview(schema.StreamFeedback)
			})
		}
		return rows, actions
	}}
}

func stepped(steps int) func(int) int {
	return func(level int) int {
		step := (min(max(level, 0), 100)*steps + 50) / 100
		return step * 100 / steps
	}
}

type preview struct {
	chimes Chimes

	mu    sync.Mutex
	timer *time.Timer
}

func (p *preview) after(s schema.Stream) {
	if p.chimes == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.timer != nil {
		p.timer.Stop()
	}
	p.timer = time.AfterFunc(settle, func() { p.chimes.Preview(s) })
}
