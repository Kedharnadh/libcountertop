package nativeapi

import (
	"reflect"

	esphome "github.com/ygelfand/go-esphome-device"
)

func SubDevices(all []esphome.Device, ents []esphome.Entity) []esphome.Device {
	used := map[uint32]bool{}
	for _, e := range ents {
		used[deviceOf(e)] = true
	}

	var out []esphome.Device
	for _, d := range all {
		if used[d.ID] {
			out = append(out, d)
		}
	}
	return out
}

func deviceOf(e esphome.Entity) uint32 {
	v := reflect.ValueOf(e)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0
	}
	if f := v.FieldByName("DeviceID"); f.IsValid() && f.Kind() == reflect.Uint32 {
		return uint32(f.Uint())
	}
	return 0
}
