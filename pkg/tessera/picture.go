package tessera

import (
	"encoding/json"
	"strconv"
	"strings"
)

const (
	refreshDefault = 15
	refreshLeast   = 5
	refreshMost    = 3600

	PictureFull  = "full"
	PictureLive  = "live"
	PictureCover = "cover"
)

type picture struct {
	view   uint64
	entity string
	url    string
}

func (s *Screen) AskPicture(kind string, fields map[string]string) {
	s.mu.Lock()
	if s.pictures == nil {
		s.pictures = map[string]*picture{}
	}
	p := s.pictures[kind]
	if p == nil {
		p = &picture{}
		s.pictures[kind] = p
	}
	p.view++
	p.url = ""
	view := p.view
	s.mu.Unlock()
	data := map[string]string{"view": strconv.FormatUint(view, 10)}
	for k, v := range fields {
		data[k] = v
	}
	s.Tell("screen_camera", data)
}

func (s *Screen) Picture(kind string) (entity, url string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.pictures[kind]
	if p == nil || p.url == "" {
		return "", "", false
	}
	return p.entity, p.url, true
}

type cameraMessage struct {
	T    string  `json:"t"`
	E    string  `json:"e"`
	U    string  `json:"u"`
	View *uint64 `json:"view"`
}

func (s *Screen) answerPicture(raw json.RawMessage) (string, bool) {
	var m cameraMessage
	if json.Unmarshal(raw, &m) != nil {
		return "", false
	}
	kind := cut(m.T, 8)
	switch kind {
	case PictureLive:
		if len(m.E) > 400 {
			return "", false
		}
		for _, e := range strings.Split(m.E, ",") {
			if e != "" && !validEntity(e) {
				return "", false
			}
		}
	case PictureFull, PictureCover, "alert", "lib", "saver":
		if !validEntity(m.E) {
			return "", false
		}
	default:
		return "", false
	}
	if len(m.U) > 240 || (m.U != "" && !strings.HasPrefix(m.U, "http://")) {
		return "", false
	}
	p := s.pictures[kind]
	if p == nil || m.View == nil || *m.View != p.view {
		return "Synced", true
	}
	p.entity, p.url = m.E, m.U
	return "Synced", true
}
