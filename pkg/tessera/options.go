package tessera

import (
	"encoding/json"
	"strconv"
)

const (
	namesMost = 400
	nameMost  = 48
)

type Names struct {
	Entity string
	Page   int
	Pages  int
	Names  []string
}

func (s *Screen) AskNames(entity string, page int) {
	s.mu.Lock()
	s.names = &Names{Entity: entity, Page: page}
	s.mu.Unlock()
	s.Tell("screen_options", map[string]string{"entity": entity, "page": strconv.Itoa(page)})
}

func (s *Screen) Names(entity string) (Names, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.names == nil || s.names.Entity != entity || s.names.Pages == 0 {
		return Names{}, false
	}
	n := *s.names
	n.Names = append([]string(nil), n.Names...)
	return n, true
}

func (s *Screen) answerNames(raw json.RawMessage) (string, bool) {
	var m struct {
		Entity string   `json:"e"`
		Page   int      `json:"i"`
		Pages  int      `json:"n"`
		Names  []string `json:"o"`
	}
	if json.Unmarshal(raw, &m) != nil || !validEntity(m.Entity) || m.Pages < 1 || m.Page < 0 ||
		m.Page >= m.Pages {
		return "", false
	}
	if s.names == nil || s.names.Entity != m.Entity || s.names.Page != m.Page {
		return "Synced", true
	}
	names := make([]string, 0, min(len(m.Names), namesMost))
	for _, n := range m.Names[:min(len(m.Names), namesMost)] {
		names = append(names, cut(n, nameMost))
	}
	s.names = &Names{Entity: m.Entity, Page: m.Page, Pages: m.Pages, Names: names}
	return "Synced", true
}
