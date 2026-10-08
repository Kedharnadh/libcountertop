package live

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ygelfand/libcountertop/pkg/say"
	setting "github.com/ygelfand/libcountertop/pkg/settings"
)

type Knobs struct {
	EV                                               int
	Scene, Banding, ISO, WhiteBalance, Effect        string
	Brightness, Contrast, Saturation, Sharpness, Hue string
	Noise                                            string
	RateAuto                                         bool
	Rate                                             int
	MainOn, SubOn                                    bool
	MainSize                                         string
	SubWidth, Keyframe                               int
	Quality                                          string
}

type Knob = setting.Setting[Knobs]

const (
	PictureGroup  setting.Group = "Picture"
	ExposureGroup setting.Group = "Exposure"
	BalanceGroup  setting.Group = "White Balance"
)

type Vendor struct {
	EVMin, EVMax int
	Scenes       []string
	Balances     []string
	Effects      []string
	ISOs         []string
	Levels       bool
	Sliders      bool
	Noises       []string
	Rates        []int
}

const (
	sliderMax    = 10
	sliderMiddle = "5"
	sharpSteps   = 6
	sharpStep    = 6
	sharpDefault = "2"
)

var bandings = []string{"auto", "off", "50hz", "60hz"}
var levels = []string{"low", "middle", "high"}

func (c *Camera) Defaults() Knobs {
	k := Knobs{
		Scene:        "auto",
		Banding:      "auto",
		ISO:          "auto",
		WhiteBalance: "auto",
		Effect:       "none",
		Brightness:   "middle",
		Contrast:     "middle",
		Saturation:   "middle",
		Sharpness:    "middle",
		Hue:          "middle",
		RateAuto:     true,
		Rate:         FPS,
		MainOn:       true,
		SubOn:        c.options.Streams.SubDefault > 0,
		MainSize:     c.options.Streams.MainDefault,
		Keyframe:     Keyframe,
		Quality:      "standard",
	}
	if subs := c.options.Streams.Sub; len(subs) > 0 {
		k.SubWidth = max(c.options.Streams.SubDefault, subs[len(subs)-1])
	}
	if c.options.Vendor.Sliders {
		k.Brightness, k.Contrast, k.Saturation, k.Sharpness = sliderMiddle, sliderMiddle, sliderMiddle, sharpDefault
		k.Noise = "fast"
	}
	return k
}

func nearest(rates []int, n int) int {
	best := rates[0]
	for _, r := range rates {
		if abs(r-n) < abs(best-n) {
			best = r
		}
	}
	return best
}

func abs(n int) int { return max(n, -n) }

func rateRows(rates []int) []Knob {
	auto := Knob{
		Name:  "framerate_auto",
		Kind:  setting.Toggle,
		Group: ExposureGroup,
		Icon:  "mdi:speedometer",
		Read:  func(k *Knobs) string { return setting.OnOff(k.RateAuto) },
	}
	auto.Write = func(k *Knobs, v string) error {
		on, ok := setting.Boolean(v)
		if !ok {
			return auto.Bad(v, "on or off")
		}
		k.RateAuto = on
		return nil
	}
	fixed := Knob{
		Name:   "framerate",
		Kind:   setting.Number,
		Group:  ExposureGroup,
		Icon:   "mdi:filmstrip",
		Slider: true,
		Min:    rates[0],
		Max:    rates[len(rates)-1],
		Unit:   "fps",
		IdleAs: "Auto",
		Idle:   func(k *Knobs) bool { return k.RateAuto },
		Read:   func(k *Knobs) string { return strconv.Itoa(k.Rate) },
	}
	fixed.Write = func(k *Knobs, s string) error {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fixed.Bad(s, fmt.Sprintf("between %d and %d", rates[0], rates[len(rates)-1]))
		}
		k.Rate = nearest(rates, n)
		return nil
	}
	return []Knob{auto, fixed}
}

func number(name string, g setting.Group, icon string, most int, field func(*Knobs) *string) Knob {
	k := Knob{
		Name:   name,
		Kind:   setting.Number,
		Group:  g,
		Icon:   icon,
		Slider: true,
		Min:    0,
		Max:    most,
		Read:   func(k *Knobs) string { return *field(k) },
	}
	k.Write = func(at *Knobs, s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < 0 || n > most {
			return k.Bad(s, fmt.Sprintf("between 0 and %d", most))
		}
		*field(at) = strconv.Itoa(n)
		return nil
	}
	return k
}

func words(values []string) []setting.Option {
	out := make([]setting.Option, len(values))
	for i, v := range values {
		out[i] = setting.Option{Value: v, Key: v}
	}
	return out
}

func labels(values []string) []setting.Option {
	out := make([]setting.Option, len(values))
	for i, v := range values {
		out[i] = setting.Option{Value: v, Label: v}
	}
	return out
}

func choice(
	name string,
	g setting.Group,
	icon string,
	opts []setting.Option,
	field func(*Knobs) *string,
) Knob {
	k := Knob{Name: name, Kind: setting.Choice, Group: g, Icon: icon, Options: opts,
		Read: func(k *Knobs) string { return *field(k) }}
	k.Write = func(at *Knobs, v string) error {
		for _, o := range opts {
			if o.Value == v {
				*field(at) = v
				return nil
			}
		}
		return k.Bad(v, "not one of its options")
	}
	return k
}

func (c *Camera) rows() []Knob {
	v := c.options.Vendor
	ev := Knob{
		Name:   "ev",
		Kind:   setting.Number,
		Group:  ExposureGroup,
		Icon:   "mdi:plus-minus-variant",
		Slider: true,
		Min:    v.EVMin,
		Max:    v.EVMax,
		Read:   func(k *Knobs) string { return strconv.Itoa(k.EV) },
	}
	ev.Write = func(k *Knobs, s string) error {
		n, err := strconv.Atoi(s)
		if err != nil || n < v.EVMin || n > v.EVMax {
			return ev.Bad(s, fmt.Sprintf("between %d and %d", v.EVMin, v.EVMax))
		}
		k.EV = n
		return nil
	}

	out := []Knob{
		ev,
		choice(
			"scene",
			ExposureGroup,
			"mdi:image-filter-hdr",
			words(v.Scenes),
			func(k *Knobs) *string { return &k.Scene },
		),
		choice(
			"banding",
			ExposureGroup,
			"mdi:sine-wave",
			words(bandings),
			func(k *Knobs) *string { return &k.Banding },
		),
	}
	if len(v.ISOs) > 0 {
		out = append(
			out,
			choice(
				"iso",
				ExposureGroup,
				"mdi:film",
				labels(v.ISOs),
				func(k *Knobs) *string { return &k.ISO },
			),
		)
	}
	out = append(out,
		choice("whitebalance", BalanceGroup, "mdi:white-balance-auto", words(v.Balances),
			func(k *Knobs) *string { return &k.WhiteBalance }),
		choice("effect", PictureGroup, "mdi:image-filter-black-white", words(v.Effects),
			func(k *Knobs) *string { return &k.Effect }),
	)
	if v.Levels {
		out = append(
			out,
			choice(
				"brightness",
				PictureGroup,
				"mdi:brightness-6",
				words(levels),
				func(k *Knobs) *string { return &k.Brightness },
			),
			choice(
				"contrast",
				PictureGroup,
				"mdi:contrast-box",
				words(levels),
				func(k *Knobs) *string { return &k.Contrast },
			),
			choice(
				"saturation",
				PictureGroup,
				"mdi:palette",
				words(levels),
				func(k *Knobs) *string { return &k.Saturation },
			),
			choice(
				"sharpness",
				PictureGroup,
				"mdi:blur-off",
				words(levels),
				func(k *Knobs) *string { return &k.Sharpness },
			),
			choice(
				"hue",
				PictureGroup,
				"mdi:looks",
				words(levels),
				func(k *Knobs) *string { return &k.Hue },
			),
		)
	}
	out = append(out, rateRows(v.Rates)...)
	out = append(out, c.streamRows()...)
	if v.Sliders {
		out = append(
			out,
			number(
				"brightness",
				PictureGroup,
				"mdi:brightness-6",
				sliderMax,
				func(k *Knobs) *string { return &k.Brightness },
			),
			number(
				"contrast",
				PictureGroup,
				"mdi:contrast-box",
				sliderMax,
				func(k *Knobs) *string { return &k.Contrast },
			),
			number(
				"saturation",
				PictureGroup,
				"mdi:palette",
				sliderMax,
				func(k *Knobs) *string { return &k.Saturation },
			),
			number(
				"sharpness",
				PictureGroup,
				"mdi:blur-off",
				sharpSteps,
				func(k *Knobs) *string { return &k.Sharpness },
			),
			choice(
				"noise",
				PictureGroup,
				"mdi:grain",
				words(v.Noises),
				func(k *Knobs) *string { return &k.Noise },
			),
		)
	}
	return out
}

func (c *Camera) Table() *setting.Table[Knobs] {
	c.built.Do(func() {
		c.table = setting.NewTable(
			"camera",
			[]setting.Group{PictureGroup, ExposureGroup, BalanceGroup, StreamGroup},
			c.rows(),
			setting.Messages{Text: say.T, Missing: say.Missing},
		)
	})
	return c.table
}

func (c *Camera) Params(k Knobs) string {
	v := c.options.Vendor
	p := []string{
		"exposure-compensation=" + strconv.Itoa(k.EV),
		"scene-mode=" + k.Scene,
		"antibanding=" + k.Banding,
		"whitebalance=" + k.WhiteBalance,
		"effect=" + k.Effect,
	}
	if v.Levels {
		p = append(p, "iso-speed="+k.ISO, "brightness="+k.Brightness, "contrast="+k.Contrast,
			"saturation="+k.Saturation, "edge="+k.Sharpness, "hue="+k.Hue)
	}
	if k.RateAuto {
		p = append(p, "fps-range=auto")
	} else {
		r := nearest(v.Rates, k.Rate)
		p = append(p, fmt.Sprintf("fps-range=%d,%d", r, r))
	}
	if v.Sliders {
		sharp, _ := strconv.Atoi(k.Sharpness)
		p = append(
			p,
			"brightness="+k.Brightness,
			"contrast="+k.Contrast,
			"saturation="+k.Saturation,
			"sharpness="+strconv.Itoa(sharp*sharpStep),
			"noise-reduction="+strconv.Itoa(max(0, slices.Index(v.Noises, k.Noise))),
		)
	}
	return strings.Join(p, ";")
}
