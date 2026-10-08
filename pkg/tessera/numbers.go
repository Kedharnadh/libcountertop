package tessera

import (
	"strconv"
	"strings"
)

type Numbers struct {
	Style    int
	GroupMin int
	Percent  int
}

var styles = map[string]int{"auto": 0, "point": 1, "comma": 2, "space": 3}

func (n Numbers) from(m message) Numbers {
	if m.Numbers != nil {
		if style, ok := styles[*m.Numbers]; ok {
			n.Style = style
		}
	}
	if m.GroupMin != nil && *m.GroupMin >= 1 && *m.GroupMin <= 2 {
		n.GroupMin = *m.GroupMin
	}
	if m.Percent != nil {
		n.Percent = 1
		if *m.Percent {
			n.Percent = 2
		}
	}
	return n
}

func (n Numbers) decimal() string {
	if n.Style == 2 || n.Style == 3 {
		return ","
	}
	return "."
}

func (n Numbers) group() string {
	switch n.Style {
	case 2:
		return "."
	case 3:
		return " "
	}
	return ","
}

func (n Numbers) write(whole, fraction string) string {
	from := 4
	if n.GroupMin >= 2 {
		from = 5
	}
	var b strings.Builder
	if len(whole) >= from {
		lead := len(whole) % 3
		if lead == 0 {
			lead = 3
		}
		b.WriteString(whole[:lead])
		for i := lead; i < len(whole); i += 3 {
			b.WriteString(n.group())
			b.WriteString(whole[i : i+3])
		}
	} else {
		b.WriteString(whole)
	}
	if fraction != "" {
		b.WriteString(n.decimal())
		b.WriteString(fraction)
	}
	return b.String()
}

func (n Numbers) Localize(state string) string {
	start := 0
	if strings.HasPrefix(state, "-") {
		start = 1
	}
	if start == len(state) {
		return state
	}
	dot := -1
	for i := start; i < len(state); i++ {
		switch c := state[i]; {
		case c == '.' && dot < 0 && i > start && i+1 < len(state):
			dot = i
		case c < '0' || c > '9':
			return state
		}
	}
	if dot < 0 {
		return state[:start] + n.write(state[start:], "")
	}
	return state[:start] + n.write(state[start:dot], state[dot+1:])
}

func (n Numbers) percentSign() string {
	if n.Percent == 2 {
		return " %"
	}
	return "%"
}

func (n Numbers) WithUnit(value, unit string) string {
	switch unit {
	case "", "°":
		return value + unit
	case "%":
		return value + n.percentSign()
	}
	return value + " " + unit
}

func (n Numbers) Percentage(value int) string { return strconv.Itoa(value) + n.percentSign() }
