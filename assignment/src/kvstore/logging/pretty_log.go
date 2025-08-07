package logging

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
)

type PrettyHandler struct {
	level slog.Level
}

func NewPrettyHandler(level slog.Level) slog.Handler {
	return &PrettyHandler{
		level: level,
	}
}

func (h *PrettyHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *PrettyHandler) Handle(_ context.Context, r slog.Record) error {
	timestamp := r.Time.Format("2006/01/02 15:04:05")
	level := r.Level.String()
	msg := r.Message

	file := "unknown"
	line := 0
	if r.PC != 0 {
		if fn := runtime.FuncForPC(r.PC); fn != nil {
			file, line = fn.FileLine(r.PC)
			if slash := strings.LastIndex(file, "/"); slash != -1 {
				file = file[slash+1:] // Short filename only
			}
		}
	}

	var attrs string
	r.Attrs(func(a slog.Attr) bool {
		attrs += fmt.Sprintf(" %s=%v", a.Key, a.Value)
		return true
	})

	fmt.Fprintf(os.Stdout, "[%s][%-5s] %s:%d: %s%s\n", timestamp, level, file, line, msg, attrs)

	return nil
}

func (h *PrettyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &PrettyHandler{level: h.level}
}

func (h *PrettyHandler) WithGroup(name string) slog.Handler {
	return &PrettyHandler{level: h.level}
}
