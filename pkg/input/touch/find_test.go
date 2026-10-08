package touch

import "testing"

const procDevices = `I: Bus=0000 Vendor=0000 Product=0000 Version=0000
N: Name="qpnp_pon"
P: Phys=qpnp_pon/input0
S: Sysfs=/devices/virtual/input/input0
U: Uniq=
H: Handlers=event0 cpufreq
B: PROP=0
B: EV=3

I: Bus=0018 Vendor=dead Product=beef Version=28bb
N: Name="goodix-ts"
P: Phys=input/ts
S: Sysfs=/devices/virtual/input/input1
U: Uniq=
H: Handlers=mdss_fb kgsl event1 cpufreq
B: PROP=2
B: EV=b

I: Bus=0000 Vendor=0000 Product=0000 Version=0000
N: Name="msm8953-openq624-snd-card Headset Jack"
P: Phys=ALSA
S: Sysfs=/devices/platform/soc/c051000.sound/sound/card0/input2
U: Uniq=
H: Handlers=event2
B: PROP=0
B: EV=21
`

func TestHandlerFor(t *testing.T) {
	tests := []struct {
		name   string
		device string
		want   string
		wantOK bool
	}{
		{"the touchscreen, past the handlers that are not events", "goodix-ts", "event1", true},
		{"the power key", "qpnp_pon", "event0", true},
		{"a name with spaces", "msm8953-openq624-snd-card Headset Jack", "event2", true},
		{"a device that is not there", "nonesuch", "", false},
		{"a prefix of a real name does not match", "goodix", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := handlerFor(procDevices, tt.device)
			if ok != tt.wantOK {
				t.Fatalf("handlerFor(%q) ok = %v, want %v", tt.device, ok, tt.wantOK)
			}
			if got != tt.want {
				t.Errorf("handlerFor(%q) = %q, want %q", tt.device, got, tt.want)
			}
		})
	}
}
