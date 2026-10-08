package nativeapi

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	proto "github.com/ygelfand/go-esphome-device/api"

	"github.com/ygelfand/libcountertop/pkg/android/logd"
)

const waitForReader = time.Second

type LogSource interface{ Lines() <-chan logd.Line }

func (a *Server) pipeLogs(ctx context.Context) {
	lines := a.deps.Logs.Lines()

	for {
		if !a.srv.LogsSubscribed() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(waitForReader):
			}
			continue
		}

		select {
		case <-ctx.Done():
			return
		case l := <-lines:
			if l.Dropped > 0 {
				a.srv.Log(proto.LogLevel_LOG_LEVEL_WARN,
					fmt.Sprintf("%d log lines dropped", l.Dropped))
			}
			a.srv.Log(logLevel(l.Level), l.Text)
		}
	}
}

func logLevel(l slog.Level) proto.LogLevel {
	switch {
	case l >= slog.LevelError:
		return proto.LogLevel_LOG_LEVEL_ERROR
	case l >= slog.LevelWarn:
		return proto.LogLevel_LOG_LEVEL_WARN
	case l >= slog.LevelInfo:
		return proto.LogLevel_LOG_LEVEL_INFO
	}
	return proto.LogLevel_LOG_LEVEL_DEBUG
}
