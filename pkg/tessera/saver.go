package tessera

import (
	"encoding/json"
	"slices"
)

const (
	SaverMedia  = "media"
	SaverCamera = "camera"
	SaverClock  = "clock"

	saverNameMost = 80
	saverWordMost = 8
)

type Saver struct {
	Kind        string
	Entity      string
	Name        string
	Title       string
	Artist      string
	Album       string
	Mark        string
	Playing     bool
	Features    int
	Muted       bool
	Temperature string
	Items       []BarItem
}

func (l Layout) Shown() (Saver, bool) {
	var m struct {
		Kind     string          `json:"k"`
		Entity   string          `json:"e"`
		Name     string          `json:"n"`
		Title    string          `json:"t"`
		State    string          `json:"s"`
		Features int             `json:"f"`
		Muted    int             `json:"m"`
		Word     string          `json:"w"`
		Items    json.RawMessage `json:"wi"`
		Extra    struct {
			Artist string `json:"artist"`
			Album  string `json:"album"`
			Mark   string `json:"pic"`
		} `json:"x"`
	}
	if len(l.Saver) == 0 || json.Unmarshal(l.Saver, &m) != nil {
		return Saver{}, false
	}
	s := Saver{Kind: m.Kind}
	switch m.Kind {
	case SaverMedia, SaverCamera:
		if !validEntity(m.Entity) {
			return Saver{}, false
		}
		s.Entity, s.Name = m.Entity, cut(m.Name, saverNameMost)
		if m.Kind == SaverMedia {
			s.Title, s.Artist, s.Album = m.Title, m.Extra.Artist, m.Extra.Album
			s.Mark = cut(m.Extra.Mark, 16)
			s.Playing, s.Features, s.Muted = m.State == "playing", m.Features, m.Muted == 1
		}
	case SaverClock:
		s.Temperature = cut(m.Word, saverWordMost)
		if items, ok := bar(m.Items); ok && !slices.ContainsFunc(items, func(it BarItem) bool {
			return it.Kind != "text" && it.Kind != "ago"
		}) {
			s.Items = items
		}
	default:
		return Saver{}, false
	}
	return s, true
}
