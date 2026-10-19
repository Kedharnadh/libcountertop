package schema

import (
	"github.com/ygelfand/libcountertop/pkg/audio/ducking"
	"github.com/ygelfand/libcountertop/pkg/say"
)

type Media struct {
	DuckDB  float64       `json:"duck_db"`
	Buttons VolumeButtons `json:"buttons"`
}

func DefaultMedia() Media { return Media{DuckDB: ducking.DefaultDB, Buttons: ButtonsDynamic} }

type VolumeButtons string

const (
	ButtonsStatic  VolumeButtons = "static"
	ButtonsDynamic VolumeButtons = "dynamic"
)

func (b VolumeButtons) Label() string {
	switch b {
	case ButtonsStatic:
		return say.T("buttons.static")
	case ButtonsDynamic:
		return say.T("buttons.dynamic")
	}
	return string(b)
}

func VolumeButtonModes() []VolumeButtons { return []VolumeButtons{ButtonsStatic, ButtonsDynamic} }
