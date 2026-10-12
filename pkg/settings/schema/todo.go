package schema

import "github.com/ygelfand/libcountertop/pkg/settings/storage"

type Todo struct {
	Entity    string `json:"entity"`
	Completed bool   `json:"completed"`
}

func DefaultTodo() Todo { return Todo{} }

type TodoWriter struct{ s storage.Section[Todo] }

func NewTodoWriter(s storage.Section[Todo]) TodoWriter { return TodoWriter{s: s} }

func (w TodoWriter) Entity(v string) error {
	return w.s.Update(func(c *Todo) { c.Entity = v })
}

func (w TodoWriter) Completed(v bool) error {
	return w.s.Update(func(c *Todo) { c.Completed = v })
}
