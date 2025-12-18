package logger

import (
	"bytes"
	"context"
	"strings"
	"testing"

	internalcfg "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/config"
	"github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/ctxmeta"
)

func Test_parseLevel(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"debug", "DEBUG"},
		{"info", "INFO"},
		{"", "INFO"},
		{"warn", "WARN"},
		{"warning", "WARN"},
		{"error", "ERROR"},
		{"unknown", "INFO"},
	}
	for _, tc := range cases {
		lvl := parseLevel(tc.in)
		if lvl.String() != tc.want {
			t.Fatalf("parseLevel(%q) = %s, want %s", tc.in, lvl, tc.want)
		}
	}
}

func Test_Info_Error_WriteAndFiltering(t *testing.T) {
	var buf bytes.Buffer
	cfg := &internalcfg.Logger{Level: "info"}
	l := NewWithWriter(&buf, cfg)

	l.Info("test info")
	if got := buf.String(); !strings.Contains(got, "test info") || !strings.Contains(got, "level=INFO") {
		t.Fatalf("Info not written correctly, got: %s", got)
	}

	buf.Reset()
	l.Error("error test")
	if got := buf.String(); !strings.Contains(got, "error test") || !strings.Contains(got, "level=ERROR") {
		t.Fatalf("Error not written correctly, got: %s", got)
	}

	// Filtering: set level=ERROR, Info should not appear
	buf.Reset()
	cfg.Level = "error"
	l = NewWithWriter(&buf, cfg)
	l.Info("should be filtered")
	if got := buf.String(); got != "" {
		t.Fatalf("Info should be filtered at error level, got: %q", got)
	}

	l.Error("error visible")
	if got := buf.String(); !strings.Contains(got, "error visible") {
		t.Fatalf("Error should be logged at error level, got: %q", got)
	}
}

func Test_ContextMethods_WriteToWriter(t *testing.T) {
	var buf bytes.Buffer
	cfg := &internalcfg.Logger{Level: "debug"}
	l := NewWithWriter(&buf, cfg)

	// prepare context with metadata
	ctx := context.Background()
	ctx = ctxmeta.WithRequestID(ctx, "rid-1")
	ctx = ctxmeta.WithUserID(ctx, 777)

	// InfoContext
	buf.Reset()
	l.InfoContext(ctx, "info ctx msg")
	if got := buf.String(); !strings.Contains(got, "info ctx msg") ||
		!strings.Contains(got, "level=INFO") ||
		!strings.Contains(got, "request_id") ||
		!strings.Contains(got, "rid-1") {
		t.Fatalf("InfoContext not written correctly, got: %s", got)
	}

	// ErrorContext
	buf.Reset()
	l.ErrorContext(ctx, "error ctx msg")
	if got := buf.String(); !strings.Contains(got, "error ctx msg") ||
		!strings.Contains(got, "level=ERROR") ||
		!strings.Contains(got, "user_id") ||
		!strings.Contains(got, "777") {
		t.Fatalf("ErrorContext not written correctly, got: %s", got)
	}

	// DebugContext
	buf.Reset()
	l.DebugContext(ctx, "debug ctx msg")
	if got := buf.String(); !strings.Contains(got, "debug ctx msg") || !strings.Contains(got, "level=DEBUG") {
		t.Fatalf("DebugContext not written correctly, got: %s", got)
	}

	// WarnContext
	buf.Reset()
	l.WarnContext(ctx, "warn ctx msg")
	if got := buf.String(); !strings.Contains(got, "warn ctx msg") || !strings.Contains(got, "level=WARN") {
		t.Fatalf("WarnContext not written correctly, got: %s", got)
	}
}
