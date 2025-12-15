package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"

	internalcfg "github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-bruteforce/internal/ctxmeta"
)

// Logger - обёртка над slog.Logger с контекстными методами логирования.
// Формат сообщений: (msg, key1, value1, key2, value2 ...)
type Logger struct {
	*slog.Logger
}

// Fatal не делаем в данной реализации
// // Non-context wrappers for fatal level logging.
// func (l *Logger) Fatal(msg string, v ...any) {
// 	l.Error(msg, v...)
// 	os.Exit(1)
// }

// Обогащаем логгер полями из контекста.
func (l *Logger) withCtx(ctx context.Context) *slog.Logger {
	if l == nil || l.Logger == nil {
		return slog.Default()
	}
	if ctx == nil {
		return l.Logger
	}

	// Собираем поля из контекста
	attrs := make([]any, 0, 6)

	if rid, ok := ctxmeta.RequestID(ctx); ok {
		attrs = append(attrs, "request_id", rid)
	}
	if uid, ok := ctxmeta.UserID(ctx); ok {
		attrs = append(attrs, "user_id", uid)
	}
	if tid, ok := ctxmeta.TraceID(ctx); ok {
		attrs = append(attrs, "trace_id", tid)
	}

	if len(attrs) == 0 {
		return l.Logger
	}
	return l.With(attrs...)
}

// Context-aware wrappers.
func (l *Logger) InfoContext(ctx context.Context, msg string, v ...any) {
	l.withCtx(ctx).InfoContext(ctx, msg, v...)
}

func (l *Logger) ErrorContext(ctx context.Context, msg string, v ...any) {
	l.withCtx(ctx).ErrorContext(ctx, msg, v...)
}

func (l *Logger) DebugContext(ctx context.Context, msg string, v ...any) {
	l.withCtx(ctx).DebugContext(ctx, msg, v...)
}

func (l *Logger) WarnContext(ctx context.Context, msg string, v ...any) {
	l.withCtx(ctx).WarnContext(ctx, msg, v...)
}

// New - создаём логгер на базе slog
// По умолчанию level = info
// Вывод лога - в stdout.
func New(logCfg *internalcfg.Logger) *Logger {
	lvl := parseLevel(logCfg.Level)
	hdlr := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return &Logger{slog.New(hdlr)}
}

// NewWithWriter позволяет писать в io.Writer. Для тестирования
// Если writer не задан - пишем в os.Stdout.
func NewWithWriter(w io.Writer, logCfg *internalcfg.Logger) *Logger {
	if w == nil {
		w = os.Stdout
	}
	lvl := parseLevel(logCfg.Level)
	hdlr := slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl})
	return &Logger{slog.New(hdlr)}
}

func parseLevel(s string) slog.Level {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info", "":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
