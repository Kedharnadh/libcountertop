package homeassistant

import (
	"strings"
	"testing"
)

func TestEachSlotFindsItsOwnWakeWordSelect(t *testing.T) {
	w := WakeWords{MAC: func() string { return "AA:BB:CC:DD:EE:FF" }}
	for _, tc := range []struct {
		slot int
		want string
	}{
		{0, "_wake_word$"},
		{1, "_wake_word_2$"},
	} {
		got := w.selector(tc.slot)
		if !strings.Contains(got, tc.want) {
			t.Errorf("slot %d template %q does not match %q", tc.slot+1, got, tc.want)
		}
		if !strings.Contains(got, "'aa:bb:cc:dd:ee:ff'") {
			t.Errorf("slot %d template does not match the device by its lower-case MAC", tc.slot+1)
		}
	}
}
