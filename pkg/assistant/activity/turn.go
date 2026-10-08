package activity

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"sync"
	"time"

	"github.com/ygelfand/libcountertop/pkg/runtime/component"
)

const Version = "1"

type Outcome string

const (
	Completed Outcome = "completed"
	Cancelled Outcome = "cancelled"
	Timeout   Outcome = "timeout"
	Failed    Outcome = "failed"
)

type Turn struct {
	mu   sync.Mutex
	opts Options

	id   string
	slot int
	word string

	heard string
	reply string

	audio float64

	listening time.Time
	thinking  time.Time
	replying  time.Time
	ended     time.Time
}

func (l *Log) Begin(slot int, word string) *Turn {
	return &Turn{id: newID(), slot: slot, word: word, opts: l.opts}
}

func (t *Turn) Listening() {
	if t == nil {
		return
	}
	t.at(&t.listening)
}

func (t *Turn) Heard(text string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ended.IsZero() {
		return
	}
	t.heard = text
	if t.thinking.IsZero() {
		t.thinking = time.Now()
	}
}

func (t *Turn) Replying(text string) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.ended.IsZero() {
		return
	}
	t.reply = text
	if t.replying.IsZero() {
		t.replying = time.Now()
	}
}

func (t *Turn) Ends(how Outcome) {
	if t == nil {
		return
	}

	t.mu.Lock()
	if !t.ended.IsZero() {
		t.mu.Unlock()
		return
	}
	t.ended = time.Now()
	fields := t.fields(how)
	t.mu.Unlock()

	if t.opts.Emit != nil {
		t.opts.Emit(fields)
	}
}

func (t *Turn) Records(seconds float64) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.audio = seconds
}

func (t *Turn) ID() string {
	if t == nil {
		return ""
	}
	return t.id
}

func (t *Turn) at(when *time.Time) {
	if t == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.ended.IsZero() && when.IsZero() {
		*when = time.Now()
	}
}

func (t *Turn) fields(how Outcome) map[string]string {
	out := map[string]string{
		"version": Version,
		"id":      t.id,
		"slot":    strconv.Itoa(t.slot),
		"outcome": string(how),
	}

	if t.word != "" {
		out["wake_word"] = component.FitEvent(t.word)
	}
	if t.heard != "" {
		out["heard"] = component.FitEvent(t.heard)
	}
	if t.reply != "" {
		out["reply"] = component.FitEvent(t.reply)
	}
	if t.audio > 0 {
		out["audio_seconds"] = strconv.FormatFloat(t.audio, 'f', 1, 64)
	}

	marks := []struct {
		key string
		at  time.Time
	}{
		{"listen_ms", t.listening},
		{"think_ms", t.thinking},
		{"speak_ms", t.replying},
	}

	for i, mark := range marks {
		if mark.at.IsZero() {
			continue
		}

		until := t.ended
		for _, next := range marks[i+1:] {
			if !next.at.IsZero() {
				until = next.at
				break
			}
		}
		if ms := until.Sub(mark.at).Milliseconds(); ms > 0 {
			out[mark.key] = strconv.FormatInt(ms, 10)
		}
	}
	return out
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 16)
	}
	return hex.EncodeToString(b)
}
