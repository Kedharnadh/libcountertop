package control

import (
	"fmt"
	"slices"
	"strings"
)

type Setting[T any] struct {
	Name  string
	Field string
	Read  func(T) string
	Write func(string) error
}

func Map[T, U any](rows []Setting[U], view func(T) U) []Setting[T] {
	out := make([]Setting[T], len(rows))
	for i, row := range rows {
		out[i] = Setting[T]{
			Name:  row.Name,
			Field: row.Field,
			Read:  func(t T) string { return row.Read(view(t)) },
			Write: row.Write,
		}
	}
	return out
}

func Settings[T any](read func() T, rows []Setting[T], args []string) (string, error) {
	if len(args) == 0 {
		rows = slices.Clone(rows)
		slices.SortFunc(rows, func(a, b Setting[T]) int { return strings.Compare(a.Name, b.Name) })
		state := read()
		var out []string
		for _, row := range rows {
			out = append(out, fmt.Sprintf("%-20s %s", row.Name, row.Read(state)))
		}
		return strings.Join(out, "\n"), nil
	}
	for _, row := range rows {
		if !strings.EqualFold(row.Name, args[0]) {
			continue
		}
		if len(args) > 1 {
			if err := row.Write(strings.Join(args[1:], " ")); err != nil {
				return "", fmt.Errorf("%s: %w", row.Name, err)
			}
		}
		return row.Read(read()), nil
	}
	return "", fmt.Errorf(
		"no such setting %q; run settings without arguments to list names",
		args[0],
	)
}
