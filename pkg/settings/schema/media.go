package schema

import "github.com/ygelfand/libcountertop/pkg/audio/ducking"

type Media struct {
	DuckDB float64 `json:"duck_db"`
}

func DefaultMedia() Media { return Media{DuckDB: ducking.DefaultDB} }
