package panel

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// Writes to the PMIC's wled are undone when the controller brings the panel up.
const Backlight = "/sys/class/leds/lcd-backlight/brightness"

const DefaultBrightness = 70

func SetBacklight(percent int) error {
	full, err := backlightMax()
	if err != nil {
		return err
	}

	level := perceived(percent, full)
	if err := os.WriteFile(Backlight, []byte(strconv.Itoa(level)), 0o644); err != nil {
		return fmt.Errorf("backlight: %w", err)
	}
	return nil
}

// CIE 1976 lightness: L* = 116·Y^(1/3) − 16, linear below L* = 8.
func perceived(percent, full int) int {
	l := float64(min(max(percent, 0), 100))
	if l == 0 {
		return 0
	}
	y := l / 903.3
	if l > 8 {
		f := (l + 16) / 116
		y = f * f * f
	}
	return max(1, int(math.Round(y*float64(full))))
}

func backlightMax() (int, error) {
	b, err := os.ReadFile(strings.TrimSuffix(Backlight, "brightness") + "max_brightness")
	if err != nil {
		return 0, fmt.Errorf("backlight max: %w", err)
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("backlight max: %q", strings.TrimSpace(string(b)))
	}
	return n, nil
}
