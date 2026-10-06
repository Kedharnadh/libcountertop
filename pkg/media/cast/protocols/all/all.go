// Package all pulls in every cast protocol, each defining itself on import.
package all

import (
	_ "github.com/ygelfand/libcountertop/pkg/media/cast/protocols/primevideo"
	_ "github.com/ygelfand/libcountertop/pkg/media/cast/protocols/unsupported"
	_ "github.com/ygelfand/libcountertop/pkg/media/cast/protocols/youtube"
)
