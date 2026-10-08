package metrics

import "strings"

func (r Reader) Uptime() Reading {
	line, err := text(r.path("proc/uptime"))
	if err != nil {
		return Reading{}
	}
	first, _, _ := strings.Cut(line, " ")
	return parse(first)
}
