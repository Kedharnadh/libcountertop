package volume

import "testing"

type hardwareVolume struct {
	step    int
	changed func(int)
}

func (v *hardwareVolume) Step() int { return v.step }
func (v *hardwareVolume) Set(step int) {
	v.step = step
	if v.changed != nil {
		v.changed(step)
	}
}
func (v *hardwareVolume) WatchVolume(changed func(int)) func() {
	v.changed = changed
	return func() { v.changed = nil }
}

func TestSteppedVolumeRoundsClampsAndUnsubscribes(t *testing.T) {
	hardware := &hardwareVolume{}
	v := Steps(hardware, 30)
	changed := -1
	stop := v.Watch(func(level int) { changed = level })
	v.Set(35)
	if hardware.step != 11 || v.Level() != 36 || changed != 36 {
		t.Fatal("step/percentage disagree", hardware.step, v.Level(), changed)
	}
	v.Set(500)
	if hardware.step != 30 || changed != 100 {
		t.Fatal("high volume was not clamped")
	}
	stop()
	v.Set(-1)
	if hardware.step != 0 || changed != 100 {
		t.Fatal("negative volume or cancelled subscription was not handled")
	}
}
