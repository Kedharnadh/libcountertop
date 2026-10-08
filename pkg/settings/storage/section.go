package storage

type Section[T any] interface {
	Read() T
	Update(func(*T)) error
}

type Binding[Root, Part any] struct {
	source func() *Store[Root]
	field  func(*Root) *Part
}

func Bind[Root, Part any](source func() *Store[Root], field func(*Root) *Part) Binding[Root, Part] {
	return Binding[Root, Part]{source: source, field: field}
}

func (s Binding[Root, Part]) Read() Part {
	root := s.source().Get()
	return *s.field(&root)
}

func (s Binding[Root, Part]) Update(change func(*Part)) error {
	return s.source().Update(func(root *Root) { change(s.field(root)) })
}
