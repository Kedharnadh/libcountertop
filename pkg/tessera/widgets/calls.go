package widgets

import (
	"context"
	"errors"
	"log/slog"
	"time"

	gogui "github.com/go-gui-org/go-gui/gui"

	"github.com/ygelfand/libcountertop/pkg/say"
	"github.com/ygelfand/libcountertop/pkg/tessera"
)

const (
	watchMost = 4
	watchTime = 8 * time.Second
	showTime  = 4 * time.Second
	slowWish  = time.Second
	armTime   = 4 * time.Second
)

type outcome int

const (
	waiting outcome = iota
	answered
	refused
	silent
)

type watch struct {
	from    string
	wish    string
	since   time.Time
	outcome outcome
	until   time.Time
}

func (v *View) act(
	w *gogui.Window,
	t tessera.Tile,
	action string,
	data, templates map[string]string,
) {
	wish := ""
	if flips(action) {
		wish = "on"
		if active(t) {
			wish = "off"
		}
	}
	v.send(w, t, action, data, templates, wish)
}

func (v *View) send(
	w *gogui.Window,
	t tessera.Tile,
	action string,
	data, templates map[string]string,
	wish string,
) {
	if data == nil {
		data = map[string]string{}
	}
	data["entity_id"] = t.Entity
	v.mu.Lock()
	watched := v.watching() < watchMost
	if watched {
		v.watches[t.Entity] = &watch{from: t.State, wish: wish, since: time.Now()}
	}
	v.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		err := v.o.Perform(ctx, action, data, templates)
		if err != nil {
			slog.Warn(
				"tessera tile action failed",
				"action",
				action,
				"entity",
				t.Entity,
				"err",
				err,
			)
		}
		if watched {
			v.heard(w, t.Entity, err)
		}
	}()
}

func (v *View) watching() int {
	n := 0
	for _, wt := range v.watches {
		if wt.outcome == waiting {
			n++
		}
	}
	return n
}

func (v *View) heard(w *gogui.Window, entity string, err error) {
	v.mu.Lock()
	wt, ok := v.watches[entity]
	if ok {
		switch {
		case err == nil:
			wt.outcome = answered
		case errors.Is(err, context.DeadlineExceeded):
			wt.outcome, wt.wish, wt.until = silent, "", time.Now().Add(showTime)
		default:
			wt.outcome, wt.wish, wt.until = refused, "", time.Now().Add(showTime)
		}
	}
	v.mu.Unlock()
	v.redraw(w, 0)
	v.redraw(w, showTime)
}

func (v *View) redraw(w *gogui.Window, after time.Duration) {
	if w == nil {
		return
	}
	time.AfterFunc(after, func() { w.QueueCommand(func(w *gogui.Window) { w.InvalidateLayout() }) })
}

func (v *View) seen(t tessera.Tile, now time.Time) *watch {
	v.mu.Lock()
	defer v.mu.Unlock()
	wt, ok := v.watches[t.Entity]
	if !ok {
		return nil
	}
	switch {
	case t.State != wt.from:
		delete(v.watches, t.Entity)
		return nil
	case wt.outcome == answered && now.Sub(wt.since) > watchTime:
		delete(v.watches, t.Entity)
		return nil
	case (wt.outcome == refused || wt.outcome == silent) && now.After(wt.until):
		delete(v.watches, t.Entity)
		return nil
	}
	copied := *wt
	return &copied
}

func (wt *watch) line(now time.Time) string {
	switch wt.outcome {
	case refused:
		return say.T("tessera.refused")
	case silent:
		return say.T("tessera.silent")
	}
	if now.Sub(wt.since) > slowWish {
		return say.T("tessera.updating")
	}
	return ""
}

func (v *View) guard(w *gogui.Window, t tessera.Tile) {
	if t.State != "locked" {
		v.act(w, t, "lock.lock", nil, nil)
		return
	}
	if t.Guard == "lock_only" {
		return
	}
	v.mu.Lock()
	at, ok := v.armed[t.Entity]
	confirm := ok && time.Since(at) < armTime
	if confirm {
		delete(v.armed, t.Entity)
	} else {
		v.armed[t.Entity] = time.Now()
	}
	v.mu.Unlock()
	if confirm {
		v.act(w, t, "lock.unlock", nil, nil)
		return
	}
	v.redraw(w, armTime)
}

func (v *View) arming(entity string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	at, ok := v.armed[entity]
	return ok && time.Since(at) < armTime
}
