package safe

import (
	"log/slog"
	"runtime/debug"
)

func Go(what string, f func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("recovered from a panic",
					"in", what, "panic", r, "stack", string(debug.Stack()))
			}
		}()
		f()
	}()
}
