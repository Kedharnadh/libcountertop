package storage

// Section is atomic access to one part of a saved configuration.
type Section[T any] interface {
	Read() T
	Update(func(*T)) error
}

// Binding selects a section of a store without loading it during construction.
type Binding[Root, Part any] struct {
	source func() *Store[Root]
	field  func(*Root) *Part
}

// Bind keeps section updates inside the root store's transaction and clone policy.
// source is resolved for every operation, including after a store is replaced.
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
