package setting

import "github.com/ygelfand/libcountertop/pkg/settings/storage"

func Store[T any](at storage.Section[T], s Setting[T], v string) error {
	probe := at.Read()
	if err := s.Write(&probe, v); err != nil {
		return err
	}
	return at.Update(func(c *T) { _ = s.Write(c, v) })
}
