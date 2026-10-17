package slogtelemetry_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/slogtelemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func newText(buf *bytes.Buffer, min slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{
		Level: min,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}

func TestContract(t *testing.T) {
	t.Parallel()
	for name, f := range map[string]func(*testing.T) telemetry.Logger{
		"enabled": func(*testing.T) telemetry.Logger {
			return slogtelemetry.New(newText(&bytes.Buffer{}, slog.Level(-100)))
		},
		"info only": func(*testing.T) telemetry.Logger {
			return slogtelemetry.New(newText(&bytes.Buffer{}, slog.LevelInfo))
		},
		"nil logger": func(*testing.T) telemetry.Logger { return slogtelemetry.New(nil) },
		"zero":       func(*testing.T) telemetry.Logger { return slogtelemetry.Logger{} },
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			telemetrytest.RunLoggerContract(t, f)
		})
	}
}

func TestLog(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		level telemetry.Level
		min   slog.Level
		attrs func() []telemetry.Attr
		want  string
	}{
		{"info with every attr kind", telemetry.LevelInfo, slog.LevelDebug,
			func() []telemetry.Attr {
				return []telemetry.Attr{telemetry.String("s", "x"), telemetry.Int("i", 3), telemetry.Float("f", 1.5), telemetry.Bool("b", true)}
			}, "level=INFO msg=hello s=x i=3 f=1.5 b=true\n"},
		{"trace maps below debug", telemetry.LevelTrace, slog.LevelDebug - 4, nil, "level=DEBUG-4 msg=hello\n"},
		{"debug", telemetry.LevelDebug, slog.LevelDebug, nil, "level=DEBUG msg=hello\n"},
		{"warn", telemetry.LevelWarn, slog.LevelDebug, nil, "level=WARN msg=hello\n"},
		{"error with Err attr", telemetry.LevelError, slog.LevelDebug,
			func() []telemetry.Attr { return []telemetry.Attr{telemetry.Err(context.Canceled)} },
			"level=ERROR msg=hello exception.message=\"context canceled\"\n"},
		{"invalid attrs skipped", telemetry.LevelInfo, slog.LevelDebug,
			func() []telemetry.Attr { return []telemetry.Attr{{}, telemetry.Err(nil), telemetry.String("ok", "1")} },
			"level=INFO msg=hello ok=1\n"},
		{"above error clamps to error", telemetry.Level(99), slog.LevelDebug, nil, "level=ERROR msg=hello\n"},
		{"below trace clamps to trace", telemetry.Level(-5), slog.LevelDebug - 4, nil, "level=DEBUG-4 msg=hello\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			slogtelemetry.New(newText(&buf, tc.min)).Log(context.Background(), tc.level, "hello", tc.attrs)
			if got := buf.String(); got != tc.want {
				t.Fatalf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDisabledSkipsAttrs(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	l := slogtelemetry.New(newText(&buf, slog.LevelWarn))
	calls := 0
	l.Log(context.Background(), telemetry.LevelInfo, "hello", func() []telemetry.Attr { calls++; return nil })
	if calls != 0 || buf.Len() != 0 || l.Enabled(context.Background(), telemetry.LevelInfo) {
		t.Fatalf("calls=%d out=%q", calls, buf.String())
	}
	if !l.Enabled(context.Background(), telemetry.LevelError) {
		t.Fatal("error level must be enabled")
	}
}

func TestNilLoggerIsSilent(t *testing.T) {
	t.Parallel()
	l := slogtelemetry.New(nil)
	if l.Enabled(context.Background(), telemetry.LevelError) {
		t.Fatal("nil logger must enable nothing")
	}
	l.Log(context.Background(), telemetry.LevelError, "x", nil)
}

func ExampleNew() {
	var buf bytes.Buffer
	log := slogtelemetry.New(newText(&buf, slog.LevelInfo))
	log.Log(context.Background(), telemetry.LevelInfo, "content loaded", func() []telemetry.Attr {
		return []telemetry.Attr{telemetry.Int("count", 3)}
	})
	fmt.Println(strings.TrimSpace(buf.String()))
	// Output: level=INFO msg="content loaded" count=3
}
