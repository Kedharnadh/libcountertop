package setting

import (
	"fmt"
	"strconv"
	"strings"
)

type Labeled interface {
	~string
	Label() string
}

func Labels[V Labeled](values []V) []Option {
	out := make([]Option, len(values))
	for i, v := range values {
		out[i] = Option{Value: string(v), Label: v.Label()}
	}
	return out
}

func Flag[T any](s Setting[T], field func(*T) *bool) Setting[T] {
	s.Kind = Toggle
	s.Read = func(c *T) string { return OnOff(*field(c)) }
	s.Write = func(c *T, v string) error {
		on, ok := Boolean(v)
		if !ok {
			return fmt.Errorf("%s takes on or off, not %q", s.Name, v)
		}
		*field(c) = on
		return nil
	}
	return s
}

func Pick[T any, V Labeled](s Setting[T], values []V, field func(*T) *V) Setting[T] {
	s.Kind = Choice
	s.Options = Labels(values)
	s.Read = func(c *T) string { return string(*field(c)) }
	s.Write = func(c *T, v string) error {
		for _, want := range values {
			if strings.EqualFold(string(want), v) || strings.EqualFold(want.Label(), v) {
				*field(c) = want
				return nil
			}
		}
		return fmt.Errorf("%s takes %s, not %q", s.Name, strings.Join(s.Values(), " or "), v)
	}
	return s
}

func Words[T any](s Setting[T], most int, field func(*T) *string) Setting[T] {
	s.Kind = Text
	s.Read = func(c *T) string { return *field(c) }
	s.Write = func(c *T, v string) error {
		r := []rune(strings.TrimSpace(v))
		if len(r) > most {
			r = r[:most]
		}
		*field(c) = string(r)
		return nil
	}
	return s
}

func Count[T any](s Setting[T], field func(*T) *int) Setting[T] {
	s.Kind = Number
	s.Read = func(c *T) string { return strconv.Itoa(*field(c)) }
	s.Write = func(c *T, v string) error {
		n, err := s.Number(v)
		if err != nil {
			return err
		}
		*field(c) = n
		return nil
	}
	return s
}
