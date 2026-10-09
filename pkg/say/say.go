package say

import (
	"embed"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

//go:embed messages/*.yaml
var files embed.FS

var English = language.English

var (
	mu     sync.RWMutex
	bundle *i18n.Bundle
	local  *i18n.Localizer
	chosen string

	names = map[string]string{}

	// go-i18n Localize falls back to the default language and never reports a missing string.
	carries = map[string]map[string]bool{}

	missed sync.Map
)

func init() {
	bundle = i18n.NewBundle(English)
	bundle.RegisterUnmarshalFunc("yaml", yaml.Unmarshal)

	entries, err := files.ReadDir("messages")
	if err != nil {
		slog.Error("no messages are embedded, so the interface will show identifiers", "err", err)
		return
	}

	for _, e := range entries {
		name := path.Join("messages", e.Name())

		raw, err := files.ReadFile(name)
		if err != nil {
			slog.Error("a message file would not read", "file", e.Name(), "err", err)
			continue
		}
		if err := add(strings.TrimSuffix(e.Name(), ".yaml"), name, raw); err != nil {
			slog.Error("a message file would not load", "file", e.Name(), "err", err)
		}
	}

	local = i18n.NewLocalizer(bundle, English.String())
	chosen = English.String()
}

func Use(want string) string {
	mu.Lock()
	defer mu.Unlock()

	if want == "" {
		want = English.String()
	}

	local = i18n.NewLocalizer(bundle, want, English.String())
	chosen = settled(want)

	return chosen
}

func Chosen() string {
	mu.RLock()
	defer mu.RUnlock()
	return chosen
}

func add(tag, name string, raw []byte) error {
	flat, err := read(raw)
	if err != nil {
		return fmt.Errorf("say: reading %s: %w", name, err)
	}
	lang, err := language.Parse(tag)
	if err != nil {
		return fmt.Errorf("say: %s is not a language: %w", name, err)
	}

	messages := make([]*i18n.Message, 0, len(flat))
	has := map[string]bool{}
	for id, text := range flat {
		messages = append(messages, &i18n.Message{ID: id, Other: text})
		has[id] = true
	}
	if err := bundle.AddMessages(lang, messages...); err != nil {
		return err
	}
	carries[tag] = has

	if own := flat["language.name"]; own != "" {
		names[tag] = own
	}

	return nil
}

func read(raw []byte) (map[string]string, error) {
	var tree map[string]any
	if err := yaml.Unmarshal(raw, &tree); err != nil {
		return nil, err
	}
	flat := map[string]string{}
	return flat, flatten("", tree, flat)
}

func flatten(prefix string, tree map[string]any, flat map[string]string) error {
	for key, value := range tree {
		id := key
		switch {
		case key == "_":
			id = prefix
		case prefix != "":
			id = prefix + "." + key
		}
		switch v := value.(type) {
		case string:
			if _, dup := flat[id]; dup {
				return fmt.Errorf("%s is defined twice", id)
			}
			flat[id] = v
		case map[string]any:
			if err := flatten(id, v, flat); err != nil {
				return err
			}
		default:
			return fmt.Errorf("%s is %T, not text", id, value)
		}
	}
	return nil
}

func Name(tag string) string {
	mu.RLock()
	defer mu.RUnlock()

	if own := names[tag]; own != "" {
		return own
	}
	return tag
}

func settled(want string) string {
	if carries[want] != nil {
		return want
	}

	if base, _, ok := strings.Cut(want, "-"); ok && carries[base] != nil {
		return base
	}
	return English.String()
}

func Languages() []string {
	mu.RLock()
	defer mu.RUnlock()

	var out []string
	for tag := range carries {
		if tag == English.String() {
			continue
		}
		out = append(out, tag)
	}
	sort.Strings(out)

	return append([]string{English.String()}, out...)
}

func T(id string) string { return look(id, &i18n.LocalizeConfig{MessageID: id}) }

func F(id string, with map[string]any) string {
	return look(id, &i18n.LocalizeConfig{MessageID: id, TemplateData: with})
}

func N(id string, count int) string {
	return look(id, &i18n.LocalizeConfig{
		MessageID:    id,
		PluralCount:  count,
		TemplateData: map[string]any{"Count": count},
	})
}

// go-i18n returns the default language's text alongside an error for a missing identifier.
func look(id string, what *i18n.LocalizeConfig) string {
	mu.RLock()
	if local == nil {
		mu.RUnlock()
		return id
	}

	out, err := local.Localize(what)
	mu.RUnlock()
	if out != "" {
		return out
	}
	if err != nil {
		if _, said := missed.LoadOrStore(id, true); !said {
			slog.Warn("no text for an identifier", "id", id, "err", err)
		}
	}
	return id
}

func Ids() ([]string, error) {
	raw, err := files.ReadFile("messages/en.yaml")
	if err != nil {
		return nil, err
	}

	flat, err := read(raw)
	if err != nil {
		return nil, fmt.Errorf("say: reading en.yaml: %w", err)
	}

	out := make([]string, 0, len(flat))
	for id := range flat {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

func Untranslated(tag string) ([]string, error) {
	ids, err := Ids()
	if err != nil {
		return nil, err
	}

	mu.RLock()
	has := carries[tag]
	mu.RUnlock()

	var out []string
	for _, id := range ids {
		if !has[id] {
			out = append(out, id)
		}
	}
	return out, nil
}

func Load(tag string, yamlBody string) error {
	mu.Lock()
	defer mu.Unlock()

	return add(tag, "messages/"+tag+".yaml", []byte(yamlBody))
}

func Missing(id string) bool {
	if !strings.Contains(id, ".") {
		return false
	}

	mu.RLock()
	defer mu.RUnlock()
	if local == nil {
		return true
	}

	out, _ := local.Localize(&i18n.LocalizeConfig{MessageID: id})
	return out == "" || out == id
}
