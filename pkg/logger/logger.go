package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
)

type Logger struct {
	slog *slog.Logger
}

func NewLogger(slog *slog.Logger) *Logger {
	return &Logger{slog: slog}
}

func (l *Logger) log(level slog.Level, msg string, args ...any) {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}

	funcName := runtime.FuncForPC(pc).Name()

	args = append(
		[]any{
			"caller", funcName,
			"position", fmt.Sprintf("%s:%d", file, line),
		},
		args...,
	)
	l.slog.Log(context.TODO(), level, msg, args...)
}

func (l *Logger) Debug(msg string, args ...any) {
	l.log(slog.LevelDebug, msg, args...)
}
func (l *Logger) Info(msg string, args ...any) {
	l.log(slog.LevelInfo, msg, args...)
}
func (l *Logger) Warn(msg string, args ...any) {
	l.log(slog.LevelWarn, msg, args...)
}
func (l *Logger) Error(msg string, args ...any) {
	l.log(slog.LevelError, msg, args...)
}
