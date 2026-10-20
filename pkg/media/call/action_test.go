package call

import (
	"testing"

	discovery "github.com/ygelfand/libcountertop/pkg/network/peer"
)

func TestASpokenNameFindsItsDevice(t *testing.T) {
	peers := []discovery.Peer{
		{ID: "abc", Name: "Kitchen Blueberry"},
		{ID: "def", Name: "Athena"},
		{ID: "ghi", Name: "Office Echo"},
		{ID: "jkl", Name: "Office Show"},
	}
	for _, c := range []struct {
		said, want string
	}{
		{"athena", "def"},
		{"the Athena", "def"},
		{"kitchen blueberry", "abc"},
		{"blueberry", "abc"},
		{"def", "def"},
		{"office", ""},
		{"mom", ""},
		{"", ""},
	} {
		p, ok := Find(peers, c.said)
		if got := map[bool]string{true: p.ID, false: ""}[ok]; got != c.want {
			t.Errorf("Find(%q) = %q, want %q", c.said, got, c.want)
		}
	}
}
