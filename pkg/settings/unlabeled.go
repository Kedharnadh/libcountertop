package setting

import (
	"slices"
	"strings"
)

func (r *Registry) Unlabeled() []string {
	var out []string
	for _, b := range r.All() {
		d, ok := b.(Described)
		if !ok {
			out = append(out, b.Domain()+": no field descriptions")
			continue
		}
		domain := b.Domain()
		for _, f := range d.Fields() {
			if strings.HasPrefix(f.Group, domain+".group.") {
				out = append(out, f.Group)
			}
			if strings.HasPrefix(f.Title, domain+".setting.") {
				out = append(out, f.Title)
			}
			for _, c := range f.Choices {
				if strings.HasPrefix(c.Label, domain+".option.") {
					out = append(out, c.Label)
				}
			}
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}
