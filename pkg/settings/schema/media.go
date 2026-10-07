package schema

import "github.com/ygelfand/libcountertop/pkg/audio/ducking"

// Media stores the attenuation applied to background music during a voice turn.
type Media struct {
	DuckDB float64 `json:"duck_db"`
}

func DefaultMedia() Media { return Media{DuckDB: ducking.DefaultDB} }
