package settings

import (
	"cmp"
	widget "github.com/ygelfand/libcountertop/pkg/display/model"
	"github.com/ygelfand/libcountertop/pkg/display/shell"
	"github.com/ygelfand/libcountertop/pkg/say"
	"slices"
)

type Feature struct {
	Key  string
	On   bool
	Set  func(bool)
	Page func() shell.View
}

func FeaturesPage(read func() []Feature, push func(shell.View)) *shell.Page {
	return &shell.Page{Title: say.T("features.title"), Build: func() ([]widget.Row, []func(int)) {
		all := slices.Clone(read())
		order := map[string]int{"features.sendspin": 1, "features.cast": 2, "call.settings": 3, "features.proxy": 4, "features.speaker": 5}
		slices.SortStableFunc(all, func(a, b Feature) int {
			return cmp.Or(cmp.Compare(order[a.Key], order[b.Key]), cmp.Compare(a.Key, b.Key))
		})
		rows := []widget.Row{}
		acts := []func(int){}
		for _, f := range all {
			row := widget.Row{Label: say.T(f.Key), Hint: say.T(f.Key + ".hint"), Kind: widget.Toggle, On: f.On}
			act := func(int) { f.Set(!f.On) }
			if f.Page != nil {
				row.Kind = widget.Chevron
				act = func(int) { push(f.Page()) }
			}
			rows = append(rows, row)
			acts = append(acts, act)
		}
		return rows, acts
	}}
}
