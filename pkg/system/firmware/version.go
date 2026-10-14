package firmware

import (
	"cmp"
	"strconv"
	"strings"
)

func Newer(offered, running string) bool {
	if offered == "" || offered == running {
		return false
	}
	oc, op, _ := strings.Cut(offered, "-")
	rc, rp, _ := strings.Cut(running, "-")
	o, ok1 := core(oc)
	r, ok2 := core(rc)
	if !ok1 || !ok2 {
		return true
	}
	for i := range o {
		if c := cmp.Compare(o[i], r[i]); c != 0 {
			return c > 0
		}
	}
	if op == "" || rp == "" {
		return op == "" && rp != ""
	}
	return prerelease(op, rp) > 0
}

func core(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) > 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

func prerelease(a, b string) int {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		if c := identifier(as[i], bs[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(as), len(bs))
}

func identifier(a, b string) int {
	an, arest := leading(a)
	bn, brest := leading(b)
	if an >= 0 && bn >= 0 {
		if c := cmp.Compare(an, bn); c != 0 {
			return c
		}
		return cmp.Compare(arest, brest)
	}
	return cmp.Compare(a, b)
}

func leading(s string) (int, string) {
	end := strings.IndexFunc(s, func(r rune) bool { return r < '0' || r > '9' })
	if end == -1 {
		end = len(s)
	}
	n, err := strconv.Atoi(s[:end])
	if err != nil {
		return -1, s
	}
	return n, s[end:]
}
