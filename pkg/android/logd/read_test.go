package logd

import (
	"encoding/binary"
	"log/slog"
	"testing"
)

func record(hdr int, prio byte, tag, msg string) []byte {
	payload := []byte{prio}
	payload = append(payload, tag...)
	payload = append(payload, 0)
	payload = append(payload, msg...)
	payload = append(payload, 0)

	b := make([]byte, hdr)
	binary.LittleEndian.PutUint16(b[0:2], uint16(len(payload)))
	binary.LittleEndian.PutUint16(b[2:4], uint16(hdr))
	return append(b, payload...)
}

// logger_entry_v3 is 24 bytes, logger_entry_v4 is 28.
func TestEntryReadsBothHeaderSizes(t *testing.T) {
	for _, hdr := range []int{24, 28} {
		l, tag, ok := entry(record(hdr, 6, "app", "playback path up"))
		if !ok {
			t.Fatalf("hdr_size %d: not parsed", hdr)
		}
		if tag != "app" {
			t.Errorf("hdr_size %d: tag = %q, want app", hdr, tag)
		}
		if l.Text != "playback path up" {
			t.Errorf("hdr_size %d: text = %q", hdr, l.Text)
		}
		if l.Level != slog.LevelError {
			t.Errorf("hdr_size %d: level = %v, want error", hdr, l.Level)
		}
	}
}

func TestEntryFallsBackToTheSharedPrefix(t *testing.T) {
	b := record(entryPrefix, 4, "app", "hello")
	binary.LittleEndian.PutUint16(b[2:4], 0)

	l, tag, ok := entry(b)
	if !ok || tag != "app" || l.Text != "hello" {
		t.Fatalf("entry = %+v, %q, %v", l, tag, ok)
	}
}

func TestEntryRejectsShortAndBrokenRecords(t *testing.T) {
	cases := map[string][]byte{
		"empty":        {},
		"header only":  make([]byte, entryPrefix),
		"hdr past end": func() []byte { b := record(24, 4, "t", "m"); binary.LittleEndian.PutUint16(b[2:4], 900); return b }(),
		"no tag end":   append(make([]byte, 24), 4, 'e', 'c', 'h', 'o'),
	}
	for name, b := range cases {
		if _, _, ok := entry(b); ok {
			t.Errorf("%s: parsed, want rejected", name)
		}
	}
}

func TestEntryTrimsTheTrailingNewline(t *testing.T) {
	l, _, ok := entry(record(24, 4, "app", "ServiceCall error -65536\n"))
	if !ok {
		t.Fatal("not parsed")
	}
	if l.Text != "ServiceCall error -65536" {
		t.Errorf("text = %q", l.Text)
	}
}
