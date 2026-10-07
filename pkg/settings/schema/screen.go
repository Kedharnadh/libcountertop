package schema

import "github.com/ygelfand/libcountertop/pkg/say"

import "github.com/ygelfand/libcountertop/pkg/display/theme"

// Screen is the panel: how bright it is and what it shows when nothing else is asking.
type Screen struct {
	Backlight int        `json:"backlight"`
	Mode      ScreenMode `json:"mode"`
	Theme     string     `json:"theme"`
	Style     string     `json:"style"`
	Size      string     `json:"size"`
	Hours     HourFormat `json:"hours"`

	Keyboard KeyboardSize `json:"keyboard"`

	// Drawer is the edge the control drawer is swiped in from.
	Drawer Edge `json:"drawer"`

	// Volume is the side the volume card comes up on, left or right.
	//
	// Its own setting rather than the drawer's, because the two answer different questions: the
	// drawer's edge is where a hand swipes in from, and this is which side the volume keys are on.
	// They start the same and there is no reason they have to stay that way.
	Volume Edge `json:"volume"`

	// Marks shows a corner mark while the microphone is muted or the camera is blocked. On by
	// default: the sliders are physical, and the screen is the only thing that can say so.
	Marks bool `json:"marks"`

	// Language is the text the panel shows, as a language tag. Empty is English, which is also
	// what a tag with no messages behind it settles on.
	Language string `json:"language"`
}

const (
	// Seventy, because the panel is read across a room rather than at arm's length, and a device
	// nobody has set should be legible before it is comfortable.
	DefaultBacklight = 70

	DefaultScreenMode = ModeAuto

	DefaultHourFormat = TwentyFourHour

	DefaultMarks = true
)

func DefaultScreen() Screen {
	return Screen{
		Backlight: DefaultBacklight,
		Mode:      DefaultScreenMode,
		Theme:     theme.DefaultName,
		Hours:     DefaultHourFormat,
		Keyboard:  KeyboardCompact,
		Drawer:    DefaultEdge,
		Volume:    DefaultEdge,
		Marks:     DefaultMarks,
	}
}

// ScreenWriter changes what the panel is set to.

// All writes the panel's settings together, for a caller holding a whole one it has already
// changed — a settings table writes the field itself and hands back the struct.
const (
	UISizeMini    = "mini"
	UISizeCompact = "compact"
	UISizeMedium  = "medium"
	UISizeLarge   = "large"
	UISizeXLarge  = "xlarge"
)

func ScreenSizes() []string {
	return []string{UISizeMini, UISizeCompact, UISizeMedium, UISizeLarge, UISizeXLarge}
}

// ScreenMode is how the brightness is decided.
type ScreenMode string

const (
	// ModeAuto sets the brightness from the ambient light sensor.
	ModeAuto ScreenMode = "auto"

	// ModeManual holds whatever the backlight is set to.
	ModeManual ScreenMode = "manual"
)

// Label is how the setting is shown.
func (m ScreenMode) Label() string {
	switch m {
	case ModeAuto:
		return say.T("mode.auto")
	case ModeManual:
		return say.T("mode.manual")
	}
	return string(m)
}

// ScreenModes is every value the setting takes.
func ScreenModes() []ScreenMode { return []ScreenMode{ModeAuto, ModeManual} }

// HourFormat is how the time is written.
type HourFormat string

const (
	TwentyFourHour HourFormat = "24"
	TwelveHour     HourFormat = "12"
)

// Label is how the setting is shown.
func (h HourFormat) Label() string {
	switch h {
	case TwentyFourHour:
		return say.T("hours.24")
	case TwelveHour:
		return say.T("hours.12")
	}
	return string(h)
}

// HourFormats is every value the setting takes.
func HourFormats() []HourFormat { return []HourFormat{TwentyFourHour, TwelveHour} }

type KeyboardSize string

const (
	KeyboardCompact  KeyboardSize = "compact"
	KeyboardStandard KeyboardSize = "standard"
	KeyboardLarge    KeyboardSize = "large"
)

func (k KeyboardSize) Label() string {
	switch k {
	case KeyboardCompact:
		return say.T("keyboard.compact")
	case KeyboardStandard:
		return say.T("keyboard.standard")
	case KeyboardLarge:
		return say.T("keyboard.large")
	}
	return string(k)
}

func KeyboardSizes() []KeyboardSize {
	return []KeyboardSize{KeyboardCompact, KeyboardStandard, KeyboardLarge}
}

const DefaultTheme = theme.DefaultName
