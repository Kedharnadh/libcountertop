package nativeapi

import (
	"slices"
	"testing"

	esphome "github.com/ygelfand/go-esphome-device"
)

func ids(ds []esphome.Device) []uint32 {
	var out []uint32
	for _, d := range ds {
		out = append(out, d.ID)
	}
	return out
}

func TestASubDeviceNoEntityNamesIsLeftOut(t *testing.T) {
	all := []esphome.Device{{ID: 1, Name: "show screen"}, {ID: 3, Name: "show playback"}}
	ents := []esphome.Entity{
		&esphome.Switch{Base: esphome.Base{ObjectID: "a", DeviceID: 3}},
		&esphome.Switch{Base: esphome.Base{ObjectID: "b"}},
	}
	if got := ids(SubDevices(all, ents)); !slices.Equal(got, []uint32{3}) {
		t.Errorf("announced %v, want only playback", got)
	}
}

func TestSubDevicesKeepTheirOrder(t *testing.T) {
	all := []esphome.Device{{ID: 2}, {ID: 1}, {ID: 5}}
	ents := []esphome.Entity{
		&esphome.Switch{Base: esphome.Base{ObjectID: "a", DeviceID: 1}},
		&esphome.Switch{Base: esphome.Base{ObjectID: "b", DeviceID: 5}},
		&esphome.Switch{Base: esphome.Base{ObjectID: "c", DeviceID: 2}},
	}
	if got := ids(SubDevices(all, ents)); !slices.Equal(got, []uint32{2, 1, 5}) {
		t.Errorf("announced %v, want 2 1 5", got)
	}
}
