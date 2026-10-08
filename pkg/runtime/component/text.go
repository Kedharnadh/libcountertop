package component

import "unicode/utf8"

// MAX_LENGTH_STATE_STATE in homeassistant/const.py, counted in characters.
const StateLimit = 255

// The ESPHome API transport caps a whole message at 65515 bytes.
const EventLimit = 8000

func Fit(text string) string { return clip(text, StateLimit) }

func FitEvent(text string) string { return clip(text, EventLimit) }

func clip(text string, limit int) string {
	if utf8.RuneCountInString(text) <= limit {
		return text
	}

	kept := 0
	for i := range text {
		if kept == limit-1 {
			return text[:i] + "…"
		}
		kept++
	}
	return text
}
