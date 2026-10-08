package setting

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	On  = "on"
	Off = "off"
)

func OnOff(b bool) string {
	if b {
		return On
	}
	return Off
}

func Boolean(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case On, "true", "yes", "1", "":
		return true, true
	case Off, "false", "no", "0":
		return false, true
	}
	return false, false
}

type Kind int

const (
	Toggle Kind = iota

	Number

	Choice

	Pair

	Triple

	List

	Text

	Secret
)

func (k Kind) String() string {
	switch k {
	case Toggle:
		return "toggle"
	case Number:
		return "number"
	case Choice:
		return "choice"
	case Pair:
		return "pair"
	case Triple:
		return "triple"
	case List:
		return "list"
	case Text:
		return "text"
	case Secret:
		return "secret"
	}
	return "unknown"
}

type Group string

type Scale int

const (
	Linear Scale = iota
	Doubling
)

type Option struct {
	Value string

	Label string

	Key string
}

type Setting[T any] struct {
	Name string

	Kind  Kind
	Group Group

	ID string

	Icon string

	Slider bool

	Min, Max int
	Unit     string
	OneX     int
	Scale    Scale

	Enables bool

	Options []Option

	Read  func(*T) string
	Write func(*T, string) error

	Idle func(*T) bool

	IdleAs string

	Sums func(*T) string

	domain   string
	messages Messages
}

type Table[T any] struct {
	Domain   string
	messages Messages

	rows   []Setting[T]
	groups []Group
	alias  map[string]string
}

func NewTable[T any](
	domain string,
	groups []Group,
	rows []Setting[T],
	messages Messages,
) *Table[T] {
	for i := range rows {
		rows[i].domain = domain
		rows[i].messages = messages
	}
	return &Table[T]{Domain: domain, rows: rows, groups: groups, messages: messages}
}

func (t *Table[T]) Alias(from, to string) *Table[T] {
	if t.alias == nil {
		t.alias = map[string]string{}
	}
	t.alias[from] = to
	return t
}

func (t *Table[T]) Rows() []Setting[T] { return t.rows }

func (t *Table[T]) Groups() []Group { return t.groups }

func (t *Table[T]) In(g Group) []Setting[T] {
	var out []Setting[T]
	for _, s := range t.rows {
		if s.Group == g {
			out = append(out, s)
		}
	}
	return out
}

func (t *Table[T]) Find(name string) (Setting[T], bool) {
	if to, ok := t.alias[name]; ok {
		name = to
	}
	for _, s := range t.rows {
		if s.Name == name {
			return s, true
		}
	}
	return Setting[T]{}, false
}

func (t *Table[T]) Row(name string) (Setting[T], error) {
	s, ok := t.Find(name)
	if !ok {
		return Setting[T]{}, fmt.Errorf("%s: no setting called %q", t.Domain, name)
	}
	return s, nil
}

func (t *Table[T]) MustRow(name string) Setting[T] {
	s, err := t.Row(name)
	if err != nil {
		panic(err)
	}
	return s
}

func (t *Table[T]) Title(g Group) string {
	return t.messages.text(
		t.Domain + ".group." + strings.ToLower(strings.ReplaceAll(string(g), " ", "")),
	)
}

func (t *Table[T]) Sums(g Group, c *T) string {
	in := t.In(g)

	for _, s := range in {
		if s.Enables && !s.On(c) {
			return t.messages.text(t.Domain + ".disabled")
		}
	}
	for _, s := range in {
		if s.Sums != nil {
			return t.messages.text(t.Domain + ".sums." + s.Sums(c))
		}
	}
	for _, s := range in {
		if s.Kind == Choice {
			return s.Shows(c)
		}
	}
	return ""
}

func (t *Table[T]) Apply(c *T, name, value string) error {
	s, ok := t.Find(name)
	if !ok {
		return fmt.Errorf("%s: nothing called %q can be set", t.Domain, name)
	}
	return s.Put(c, value)
}

func (t *Table[T]) Configured(start T, saved map[string]string) (T, []error) {
	var bad []error
	for name, v := range saved {
		s, ok := t.Find(name)
		if !ok {
			continue
		}
		if err := s.Put(&start, v); err != nil {
			bad = append(bad, err)
		}
	}
	return start, bad
}

func (t *Table[T]) Arg(c *T, arg string) (bool, error) {
	if name, value, ok := strings.Cut(arg, "="); ok {
		s, known := t.Find(name)
		if !known {
			return false, nil
		}
		return true, s.Put(c, value)
	}

	if s, ok := t.Find(arg); ok && s.Kind == Toggle {
		return true, s.Put(c, On)
	}
	if off, ok := strings.CutPrefix(arg, "no"); ok {
		if s, known := t.Find(off); known && s.Kind == Toggle {
			return true, s.Put(c, Off)
		}
	}
	return false, nil
}

func (s Setting[T]) Object(domain string) string {
	if s.ID != "" {
		return s.ID
	}
	return domain + "_" + strings.ReplaceAll(s.Name, ".", "_")
}

func (s Setting[T]) Title() string { return s.messages.text(s.domain + ".setting." + s.Name) }

func (s Setting[T]) Named() string {
	if at := s.domain + ".entity." + s.Name; !s.messages.missing(at) {
		return s.messages.text(at)
	}
	return s.Title()
}

func (s Setting[T]) Label(o Option) string {
	if o.Key != "" {
		return s.messages.text(s.domain + ".option." + s.Name + "." + o.Key)
	}
	return o.Label
}

func (s Setting[T]) Values() []string {
	out := make([]string, 0, len(s.Options))
	for _, o := range s.Options {
		out = append(out, o.Value)
	}
	return out
}

func (s Setting[T]) Labels() []string {
	out := make([]string, 0, len(s.Options))
	for _, o := range s.Options {
		out = append(out, s.Label(o))
	}
	return out
}

func (s Setting[T]) Shows(c *T) string {
	if s.IdleAs != "" && s.Idle != nil && s.Idle(c) {
		return s.IdleAs
	}
	if s.Kind == Secret {
		if s.Read(c) == "" {
			return ""
		}
		return "set"
	}
	return s.LabelOf(s.Read(c))
}

func (s Setting[T]) LabelOf(value string) string {
	for _, o := range s.Options {
		if o.Value == value {
			return s.Label(o)
		}
	}
	return value
}

func (s Setting[T]) ValueOf(label string) string {
	for _, o := range s.Options {
		if s.Label(o) == label {
			return o.Value
		}
	}
	return ""
}

func (s Setting[T]) On(c *T) bool { return s.Read(c) == On }

func (s Setting[T]) Set(c *T, on bool) error { return s.Write(c, OnOff(on)) }

func (s Setting[T]) Level(c *T) int {
	n, err := strconv.Atoi(s.Read(c))
	if err != nil {
		return 0
	}
	return n
}

func (s Setting[T]) Dim(c *T) bool { return s.Idle != nil && s.Idle(c) }

func (s Setting[T]) Bad(v string, why string) error {
	if s.Kind == Choice && len(s.Options) > 0 {
		names := make([]string, 0, len(s.Options))
		for _, o := range s.Options {
			names = append(names, o.Value)
		}
		return fmt.Errorf(
			"%s: %s takes %s, not %q",
			s.domain,
			s.Name,
			strings.Join(names, " or "),
			v,
		)
	}
	if why == "" {
		return fmt.Errorf("%s: %q is not a %s for %s", s.domain, v, s.Kind, s.Name)
	}
	return fmt.Errorf("%s: %q is not %s for %s", s.domain, v, why, s.Name)
}

func (s Setting[T]) Range() string {
	if s.Kind != Number {
		return ""
	}
	out := fmt.Sprintf("%d to %d", s.Min, s.Max)
	if s.Unit != "" {
		out += " " + s.Unit
	}
	if s.OneX != 0 {
		out += fmt.Sprintf(", %d is 1x", s.OneX)
	}
	return out
}

func (s Setting[T]) Percent(raw int) int {
	if s.Kind != Number || s.Max <= s.Min {
		return 0
	}
	raw = min(max(raw, s.Min), s.Max)

	if s.Scale == Doubling {
		lo, hi, at := math.Log2(float64(s.Min)), math.Log2(float64(s.Max)), math.Log2(float64(raw))
		return int(math.Round((at - lo) / (hi - lo) * 100))
	}
	return int(math.Round(float64(raw-s.Min) / float64(s.Max-s.Min) * 100))
}

func (s Setting[T]) Raw(pct int) int {
	if s.Kind != Number || s.Max <= s.Min {
		return 0
	}
	pct = min(max(pct, 0), 100)

	if s.Scale == Doubling {
		lo, hi := math.Log2(float64(s.Min)), math.Log2(float64(s.Max))
		return int(math.Round(math.Pow(2, lo+(hi-lo)*float64(pct)/100)))
	}
	return s.Min + int(math.Round(float64(s.Max-s.Min)*float64(pct)/100))
}

func (s Setting[T]) Neutral(defaults T) int {
	if s.Kind != Number {
		return 0
	}
	return s.Percent(s.Level(&defaults))
}

func (s Setting[T]) Number(v string) (int, error) {
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, s.Bad(v, "a number")
	}
	if s.Min != 0 || s.Max != 0 {
		if n < s.Min || n > s.Max {
			return 0, s.Bad(v, "within "+s.Range())
		}
	}
	return n, nil
}

func (s Setting[T]) Put(c *T, v string) error {
	if v == "" && s.Kind != Toggle {
		return nil
	}
	return s.Write(c, v)
}

type Messages struct {
	Text    func(string) string
	Missing func(string) bool
}

func (m Messages) text(key string) string {
	if m.Text == nil {
		return key
	}
	return m.Text(key)
}
func (m Messages) missing(key string) bool { return m.Missing == nil || m.Missing(key) }
