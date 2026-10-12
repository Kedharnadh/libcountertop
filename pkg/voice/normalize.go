package voice

import (
	"strconv"
	"strings"
	"unicode"
)

var units = map[string]int{
	"zero":      0,
	"one":       1,
	"two":       2,
	"three":     3,
	"four":      4,
	"five":      5,
	"six":       6,
	"seven":     7,
	"eight":     8,
	"nine":      9,
	"ten":       10,
	"eleven":    11,
	"twelve":    12,
	"thirteen":  13,
	"fourteen":  14,
	"fifteen":   15,
	"sixteen":   16,
	"seventeen": 17,
	"eighteen":  18,
	"nineteen":  19,
}

var tens = map[string]int{
	"twenty":  20,
	"thirty":  30,
	"forty":   40,
	"fifty":   50,
	"sixty":   60,
	"seventy": 70,
	"eighty":  80,
	"ninety":  90,
}

func Normalize(text string) string {
	clean := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'':
			return unicode.ToLower(r)
		default:
			return ' '
		}
	}, text)
	return strings.Join(numbers(strings.Fields(clean)), " ")
}

func numbers(words []string) []string {
	out := make([]string, 0, len(words))
	for i := 0; i < len(words); {
		n, used := number(words[i:])
		if used == 0 {
			out = append(out, words[i])
			i++
			continue
		}
		out = append(out, strconv.Itoa(n))
		i += used
	}
	return out
}

func number(words []string) (value, used int) {
	i := 0
	if len(words) > 1 && words[0] == "a" && words[1] == "hundred" {
		value, i = 1, 1
	} else if v, ok := tens[words[0]]; ok {
		value, i = v, 1
		if len(words) > 1 {
			if u, ok := units[words[1]]; ok && u > 0 && u < 10 {
				value, i = v+u, 2
			}
		}
	} else if u, ok := units[words[0]]; ok {
		value, i = u, 1
	} else {
		return 0, 0
	}
	if i < len(words) && words[i] == "hundred" && value > 0 && value < 10 {
		value *= 100
		i++
		j := i
		if j < len(words) && words[j] == "and" {
			j++
		}
		if j < len(words) {
			if rest, n := number(words[j:]); n > 0 && rest < 100 {
				value += rest
				i = j + n
			}
		}
	}
	return value, i
}
