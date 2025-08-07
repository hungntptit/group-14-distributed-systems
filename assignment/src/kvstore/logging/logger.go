package logging

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

var Logger *slog.Logger

func InitLogger(debug bool) {
	var level = slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}
	handler := NewPrettyHandler(level)
	Logger = slog.New(handler)
}

func Infof(format string, args ...interface{}) {
	logWithCaller(slog.LevelInfo, format, args...)
}

func Errorf(format string, args ...interface{}) {
	logWithCaller(slog.LevelError, format, args...)
}

func Debugf(format string, args ...interface{}) {
	logWithCaller(slog.LevelDebug, format, args...)
}

func logWithCaller(level slog.Level, format string, args ...interface{}) {
	if !Logger.Enabled(context.Background(), level) {
		return
	}

	pc, _, _, ok := runtime.Caller(2) // skip 2 levels: logWithCaller -> Infof/Errorf/Debugf
	if !ok {
		pc = 0
	}

	msg := fmt.Sprintf(format, args...)
	record := slog.NewRecord(time.Now(), level, msg, pc)

	_ = Logger.Handler().Handle(context.Background(), record)
}
