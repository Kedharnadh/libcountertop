package saytest

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/ygelfand/libcountertop/pkg/say"
)

var asks = regexp.MustCompile(`\b(?:say|text)\.[TFN]\("([^"]+)"`)

func Catalogue(t *testing.T, tree string) {
	t.Helper()

	root, err := filepath.Abs(tree)
	if err != nil {
		t.Fatal(err)
	}

	ids, err := say.Ids()
	if err != nil {
		t.Fatal(err)
	}

	has := map[string]bool{}
	for _, id := range ids {
		has[id] = true
	}

	for _, file := range sources(t, root) {
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}

		for _, m := range asks.FindAllSubmatch(body, -1) {
			id := string(m[1])

			if strings.HasSuffix(id, ".") {
				continue
			}

			if !has[id] {
				where, _ := filepath.Rel(root, file)
				t.Errorf("%s asks for %q, which en.yaml does not have", where, id)
			}
		}
	}
}

func sources(t *testing.T, root string) []string {
	t.Helper()

	var out []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "bin", "node_modules", "testdata":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(out) < 100 {
		t.Fatalf("%d Go files under %s, want at least 100", len(out), root)
	}
	return out
}
