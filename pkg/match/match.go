package match

import (
	"slices"
	"strings"

	"github.com/antzucaro/matchr"

	"github.com/ygelfand/libcountertop/pkg/voice"
)

const (
	Threshold = 0.85
	margin    = 0.05
	sounds    = 0.93
	shortest  = 3
)

type Candidate struct {
	Key  string
	Name string
}

type scored struct {
	key   string
	score float64
}

func Best(said string, candidates []Candidate) (string, bool) {
	words := voice.Normalize(said)
	words = strings.TrimPrefix(words, "the ")
	want := compact(words)
	if want == "" {
		return "", false
	}

	var all []scored
	for _, c := range candidates {
		all = append(all, scored{key: c.Key, score: score(want, forms(c))})
	}
	slices.SortStableFunc(all, func(a, b scored) int {
		switch {
		case a.score > b.score:
			return -1
		case a.score < b.score:
			return 1
		}
		return 0
	})
	if len(all) == 0 || all[0].score < Threshold {
		return "", false
	}
	if len(all) > 1 && all[1].key != all[0].key && all[0].score-all[1].score < margin {
		return "", false
	}
	return all[0].key, true
}

func forms(c Candidate) []string {
	name := voice.Normalize(c.Name)
	out := []string{compact(name), strings.ToLower(c.Key)}
	for w := range strings.FieldsSeq(name) {
		if len(w) >= shortest {
			out = append(out, w)
		}
	}
	return out
}

func score(want string, forms []string) float64 {
	best := 0.0
	wa, wb := matchr.DoubleMetaphone(want)
	for _, f := range forms {
		if f == "" {
			continue
		}
		if f == want {
			return 1
		}
		s := matchr.JaroWinkler(want, f, false)
		if fa, fb := matchr.DoubleMetaphone(f); alike(wa, wb, fa, fb) {
			s = max(s, sounds)
		}
		best = max(best, s)
	}
	return best
}

func alike(wa, wb, fa, fb string) bool {
	for _, w := range []string{wa, wb} {
		for _, f := range []string{fa, fb} {
			if w != "" && w == f {
				return true
			}
		}
	}
	return false
}

func compact(s string) string { return strings.ReplaceAll(s, " ", "") }
