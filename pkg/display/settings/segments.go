package settings

import (
	"slices"

	"github.com/ygelfand/libcountertop/pkg/display/menu"
)

type labeled interface {
	comparable
	Label() string
}

func SegmentRow[T labeled](label string, options []T, chosen T) menu.Row {
	labels := make([]string, len(options))
	for i, o := range options {
		labels[i] = o.Label()
	}
	return menu.Row{
		Label:   label,
		Kind:    menu.Segments,
		Options: labels,
		Level:   max(slices.Index(options, chosen), 0),
	}
}

func SegmentPick[T any](options []T, set func(T)) func(int) {
	return func(i int) { set(options[i]) }
}
