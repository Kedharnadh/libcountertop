package schema

import "testing"

func TestAnUnsetSlotIsTheDefaults(t *testing.T) {
	if got := (Wake{}).Slot(0); got != DefaultWakeWord() {
		t.Errorf("an untouched slot reads %+v", got)
	}
}

func TestTheStopWordIsOffAtTheTop(t *testing.T) {
	if (Stop{Threshold: StopOff}).Listening() {
		t.Error("the stop word listens at the threshold that turns it off")
	}
	if !(Stop{Threshold: DefaultStopThreshold}).Listening() {
		t.Error("the stop word is off by default")
	}
}
