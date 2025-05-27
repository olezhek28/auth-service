package logger

import (
	"context"
	"log/slog"
	"os"
)

// Logger интерфейс для логирования
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// slogLogger обертка над slog.Logger
type slogLogger struct {
	logger *slog.Logger
}

// New создает новый структурированный логгер
func New(level slog.Level) Logger {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &slogLogger{
		logger: logger,
	}
}

// NewDevelopment создает логгер для разработки с текстовым форматом
func NewDevelopment() Logger {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	logger := slog.New(handler)

	return &slogLogger{
		logger: logger,
	}
}

func (l *slogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *slogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *slogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *slogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

func (l *slogLogger) With(args ...any) Logger {
	return &slogLogger{
		logger: l.logger.With(args...),
	}
}

func (l *slogLogger) WithContext(ctx context.Context) Logger {
	return &slogLogger{
		logger: l.logger.With(slog.Any("trace_id", ctx.Value("trace_id"))),
	}
}
