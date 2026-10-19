package theme

import (
	"fmt"
	"strconv"
	"strings"
)

const CustomName = "Custom"

type Custom struct {
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	Muted      string `json:"muted"`
	Accent     string `json:"accent"`
	Accent2    string `json:"accent2"`
}

var CustomFields = []string{"background", "surface", "text", "muted", "accent", "accent2"}

func (c *Custom) Field(name string) (*string, bool) {
	switch name {
	case "background":
		return &c.Background, true
	case "surface":
		return &c.Surface, true
	case "text":
		return &c.Text, true
	case "muted":
		return &c.Muted, true
	case "accent":
		return &c.Accent, true
	case "accent2":
		return &c.Accent2, true
	}
	return nil, false
}

func (c Custom) Theme() Theme {
	t := Default()
	for _, name := range CustomFields {
		at, _ := c.Field(name)
		v, ok := ParseColor(*at)
		if !ok {
			continue
		}
		*t.color(name) = v
	}
	t.Name = CustomName
	t.Dark = Dark(t.Background)
	return t
}

func (t *Theme) color(name string) *Color {
	switch name {
	case "background":
		return &t.Background
	case "surface":
		return &t.Surface
	case "text":
		return &t.Text
	case "muted":
		return &t.Muted
	case "accent":
		return &t.Accent
	}
	return &t.Accent2
}

func (t Theme) Color(name string) Color { return *t.color(name) }

func Resolve(name string, custom Custom) (Theme, bool) {
	if name == CustomName {
		return custom.Theme(), true
	}
	return ByName(name)
}

func ParseColor(s string) (Color, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return Color{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return Color{}, false
	}
	return rgb(uint32(v)), true
}

func CheckColor(s string) error {
	if s == "" {
		return nil
	}
	if _, ok := ParseColor(s); !ok {
		return fmt.Errorf("%q is not a #rrggbb color", s)
	}
	return nil
}
