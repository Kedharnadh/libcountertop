package vectors

import "embed"

//go:embed vectors/*.sbc
var files embed.FS

func Read(name string) ([]byte, error) { return files.ReadFile("vectors/" + name) }
