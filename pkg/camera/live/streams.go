package live

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

const StreamGroup setting.Group = "Streams"

type Streams struct {
	Main        []string
	MainDefault string
	Sub         []int
	SubDefault  int
}

var (
	qualities    = []string{"low", "standard", "high"}
	qualityScale = map[string]float64{"low": 0.5, "standard": 1, "high": 2}
)

const (
	keyframeMin = 1
	keyframeMax = 10
	bitrateMin  = 250_000
	bitrateMax  = 10_000_000
)

func parseSize(s string) (int, int) {
	w, h, _ := strings.Cut(s, "x")
	wi, _ := strconv.Atoi(w)
	hi, _ := strconv.Atoi(h)
	return wi, hi
}

func round16(n int) int { return (n + 8) / 16 * 16 }

func subFor(k Knobs) (int, int) {
	mw, mh := parseSize(k.MainSize)
	w := min(k.SubWidth, mw)
	return w, round16(w * mh / mw)
}

func bitrateFor(k Knobs, w, h int) int {
	scale, ok := qualityScale[k.Quality]
	if !ok {
		scale = 1
	}
	return max(bitrateMin, min(int(float64(w*h*FPS/10)*scale), bitrateMax))
}

func toggle(name, icon string, field func(*Knobs) *bool) Knob {
	k := Knob{Name: name, Kind: setting.Toggle, Group: StreamGroup, Icon: icon,
		Read: func(k *Knobs) string { return setting.OnOff(*field(k)) }}
	k.Write = func(at *Knobs, v string) error {
		on, ok := setting.Boolean(v)
		if !ok {
			return k.Bad(v, "on or off")
		}
		*field(at) = on
		return nil
	}
	return k
}

func sizeLabels(values []string) []setting.Option {
	out := make([]setting.Option, len(values))
	for i, v := range values {
		out[i] = setting.Option{Value: v, Label: strings.Replace(v, "x", "×", 1)}
	}
	return out
}

func (c *Camera) streamRows() []Knob {
	mains := c.options.Streams.Main
	mainSize := Knob{
		Name:    "main_size",
		Kind:    setting.Choice,
		Group:   StreamGroup,
		Icon:    "mdi:aspect-ratio",
		Options: sizeLabels(mains),
		Read:    func(k *Knobs) string { return k.MainSize },
	}
	mainSize.Write = func(k *Knobs, v string) error {
		if !slices.Contains(mains, v) {
			return mainSize.Bad(v, "")
		}
		k.MainSize = v
		return nil
	}

	keyframe := Knob{
		Name:   "keyframe",
		Kind:   setting.Number,
		Group:  StreamGroup,
		Icon:   "mdi:key-variant",
		Slider: true,
		Min:    keyframeMin,
		Max:    keyframeMax,
		Unit:   "s",
		Read:   func(k *Knobs) string { return strconv.Itoa(k.Keyframe) },
	}
	keyframe.Write = func(k *Knobs, s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < keyframeMin || n > keyframeMax {
			return keyframe.Bad(s, fmt.Sprintf("between %d and %d", keyframeMin, keyframeMax))
		}
		k.Keyframe = n
		return nil
	}

	out := []Knob{
		toggle("main_on", "mdi:video", func(k *Knobs) *bool { return &k.MainOn }),
		mainSize,
	}
	out = append(out, c.subRows()...)
	return append(out,
		keyframe,
		choice(
			"quality",
			StreamGroup,
			"mdi:high-definition",
			words(qualities),
			func(k *Knobs) *string { return &k.Quality },
		),
	)
}

func (c *Camera) subRows() []Knob {
	widths := c.options.Streams.Sub
	if len(widths) == 0 {
		return nil
	}
	subOpts := make([]setting.Option, len(widths))
	for i, w := range widths {
		subOpts[i] = setting.Option{Value: strconv.Itoa(w), Label: fmt.Sprintf("%d px wide", w)}
	}
	subSize := Knob{
		Name:    "sub_size",
		Kind:    setting.Choice,
		Group:   StreamGroup,
		Icon:    "mdi:aspect-ratio",
		Options: subOpts,
		Idle:    func(k *Knobs) bool { return !k.SubOn },
		Read:    func(k *Knobs) string { return strconv.Itoa(k.SubWidth) },
	}
	subSize.Write = func(k *Knobs, v string) error {
		n, err := strconv.Atoi(v)
		if err != nil || !slices.Contains(widths, n) {
			return subSize.Bad(v, "")
		}
		k.SubWidth = n
		return nil
	}
	return []Knob{
		toggle("sub_on", "mdi:video-outline", func(k *Knobs) *bool { return &k.SubOn }),
		subSize,
	}
}

func (c *Camera) Served() []int {
	k := c.Saved()
	var out []int
	if k.MainOn {
		out = append(out, 0)
	}
	if k.SubOn {
		out = append(out, 1)
	}
	return out
}

func shapeOf(k Knobs) string {
	return fmt.Sprintf("%s/%v/%d/%d/%s", k.MainSize, k.SubOn, k.SubWidth, k.Keyframe, k.Quality)
}
