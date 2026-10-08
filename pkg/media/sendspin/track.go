package sendspin

import (
	"github.com/Sendspin/sendspin-go/pkg/protocol"
)

type track struct {
	Title  string
	Artist string
	Album  string
}

// The wire is tristate: an absent key leaves, null clears, a value sets.
func (t track) merge(m *protocol.MetadataState) track {
	t.Title = field(m, "title", m.Title, t.Title)
	t.Artist = field(m, "artist", m.Artist, t.Artist)
	t.Album = field(m, "album", m.Album, t.Album)
	return t
}

func (t track) empty() bool { return t == track{} }

func field(m *protocol.MetadataState, key string, now *string, was string) string {
	if !m.HasField(key) {
		return was
	}
	if now == nil {
		return ""
	}
	return *now
}
