package homeassistant

import (
	"strings"
	"testing"
)

func TestEachSlotFindsItsOwnSelect(t *testing.T) {
	const mac = "AA:BB:CC:DD:EE:FF"
	for _, tc := range []struct {
		base string
		slot int
		want string
	}{
		{"_wake_word", 0, "_wake_word$"},
		{"_wake_word", 1, "_wake_word_2$"},
		{"_assistant", 0, "_assistant$"},
		{"_assistant", 1, "_assistant_2$"},
	} {
		got := slotSelect(mac, tc.base, tc.slot)
		if !strings.Contains(got, tc.want) {
			t.Errorf("%s slot %d template %q does not match %q", tc.base, tc.slot+1, got, tc.want)
		}
		if !strings.Contains(got, "'aa:bb:cc:dd:ee:ff'") {
			t.Errorf(
				"%s slot %d template does not match the device by its lower-case MAC",
				tc.base,
				tc.slot+1,
			)
		}
	}
}
