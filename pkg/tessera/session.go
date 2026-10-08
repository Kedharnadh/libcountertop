package tessera

import (
	"fmt"
	"strconv"
)

const (
	maxPages  = 32
	maxTiles  = 1024
	barItems  = 16
	wireItems = 6
)

type verdict int

const (
	reject verdict = iota
	duplicate
	accept
)

type beginning int

const (
	refused beginning = iota
	unchanged
	replaced
)

type session struct {
	lease, hello, revision uint64
	sequence, digest       uint32
	pages                  uint32
	tiles                  map[int]bool
	expectedPages          int
	expectedTiles          int
	granted, begun, active bool
}

func key(text string) (uint64, bool) {
	if len(text) != 16 {
		return 0, false
	}
	for _, c := range text {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(text, 16, 64)
	return v, err == nil
}

func keyText(v uint64) string { return fmt.Sprintf("%016x", v) }

func (s *session) grant(request, random uint64) uint64 {
	if s.granted && request == s.hello {
		return s.lease
	}
	s.hello, s.lease = request, random
	s.granted, s.begun = true, false
	s.sequence, s.digest = 0, 0
	return s.lease
}

func (s *session) packet(lease uint64, next, hash uint32) verdict {
	if !s.granted || lease != s.lease || next == 0 || next < s.sequence {
		return reject
	}
	if next == s.sequence {
		if hash == s.digest {
			return duplicate
		}
		return reject
	}
	return accept
}

func (s *session) accepted(next, hash uint32) { s.sequence, s.digest = next, hash }

func (s *session) begin(revision uint64, pages, tiles int) beginning {
	if !s.granted || pages == 0 || pages > maxPages || tiles > maxTiles || s.begun {
		return refused
	}
	s.begun = true
	if s.active && s.revision == revision && s.expectedPages == pages && s.expectedTiles == tiles {
		return unchanged
	}
	s.active = false
	s.revision = revision
	s.expectedPages, s.expectedTiles = pages, tiles
	s.pages = 0
	s.tiles = map[int]bool{}
	return replaced
}

func (s *session) matches(revision uint64) bool { return s.begun && s.revision == revision }

func (s *session) page(index int) bool {
	if !s.begun || s.active || index < 0 || index >= s.expectedPages || s.receivedPage(index) {
		return false
	}
	s.pages |= 1 << index
	return true
}

func (s *session) tile(index int) bool {
	if !s.begun || s.active || index < 0 || index >= s.expectedTiles || s.tiles[index] {
		return false
	}
	s.tiles[index] = true
	return true
}

func (s *session) receivedPage(index int) bool {
	return index >= 0 && index < 32 && s.pages&(1<<index) != 0
}

func (s *session) complete() bool {
	want := uint32(1)<<s.expectedPages - 1
	return s.begun && s.expectedPages > 0 && s.pages == want && len(s.tiles) == s.expectedTiles
}

func (s *session) commit() bool {
	if !s.complete() {
		return false
	}
	s.active = true
	return true
}
