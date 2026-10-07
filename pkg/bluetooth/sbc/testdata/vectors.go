// Package vectors supplies the shared SBC fixtures to integration tests.
package vectors

import "embed"

//go:embed vectors/*.sbc
var files embed.FS

func Read(name string) ([]byte, error) { return files.ReadFile("vectors/" + name) }
