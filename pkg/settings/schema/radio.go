package schema

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/settings/storage"
)

type Station struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Thumb string `json:"thumb,omitempty"`
}

type Radio struct {
	Favorites []Station `json:"favorites"`
	Last      string    `json:"last,omitempty"`
	LastTitle string    `json:"last_title,omitempty"`
}

func DefaultRadio() Radio { return Radio{} }

func (r Radio) Clone() Radio {
	r.Favorites = slices.Clone(r.Favorites)
	return r
}

func (r Radio) Favorite(id string) bool {
	return slices.ContainsFunc(r.Favorites, func(s Station) bool { return s.ID == id })
}

type RadioWriter struct{ s storage.Section[Radio] }

func NewRadioWriter(s storage.Section[Radio]) RadioWriter { return RadioWriter{s: s} }

func (w RadioWriter) Toggle(s Station) error {
	return w.s.Update(func(r *Radio) {
		if i := slices.IndexFunc(
			r.Favorites,
			func(f Station) bool { return f.ID == s.ID },
		); i >= 0 {
			r.Favorites = slices.Delete(r.Favorites, i, i+1)
			return
		}
		r.Favorites = append(r.Favorites, s)
	})
}

func (w RadioWriter) Last(s Station) error {
	return w.s.Update(func(r *Radio) { r.Last, r.LastTitle = s.ID, s.Title })
}
