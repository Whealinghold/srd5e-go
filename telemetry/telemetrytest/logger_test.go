package telemetrytest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

func attrs(a ...telemetry.Attr) func() []telemetry.Attr { return func() []telemetry.Attr { return a } }

func TestLoggerMinLevel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	tests := []struct {
		name    string
		min     telemetry.Level
		logged  telemetry.Level
		wantRec bool
	}{
		{"default records trace", telemetry.LevelTrace, telemetry.LevelTrace, true},
		{"at min", telemetry.LevelWarn, telemetry.LevelWarn, true},
		{"above min", telemetry.LevelWarn, telemetry.LevelError, true},
		{"below min", telemetry.LevelWarn, telemetry.LevelInfo, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			l := telemetrytest.NewLogger(tc.min)
			calls := 0
			l.Log(ctx, tc.logged, "m", func() []telemetry.Attr { calls++; return nil })
			if got := len(l.Records()) == 1; got != tc.wantRec {
				t.Fatalf("recorded = %v, want %v", got, tc.wantRec)
			}
			wantCalls := 0
			if tc.wantRec {
				wantCalls = 1
			}
			if calls != wantCalls || l.Enabled(ctx, tc.logged) != tc.wantRec {
				t.Fatalf("attrs calls = %d (want %d), Enabled = %v", calls, wantCalls, l.Enabled(ctx, tc.logged))
			}
		})
	}
}

func TestLoggerRecords(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	l := new(telemetrytest.Logger) // zero value is usable
	l.Log(ctx, telemetry.LevelInfo, "one", attrs(telemetry.String("k", "v")))
	l.Log(ctx, telemetry.LevelDebug, "two", nil)
	l.Log(ctx, telemetry.LevelInfo, "three", attrs(telemetry.Err(errors.New("boom"))))

	recs := l.Records()
	if len(recs) != 3 || recs[0].Message != "one" || recs[1].Message != "two" || recs[2].Message != "three" {
		t.Fatalf("records = %+v", recs)
	}
	if len(recs[1].Attrs) != 0 {
		t.Fatal("no closure must give empty attrs")
	}
	if at := l.RecordsAt(telemetry.LevelInfo); len(at) != 2 || at[1].Attrs[0].Value() != "boom" {
		t.Fatalf("RecordsAt = %+v", at)
	}
	if len(l.RecordsAt(telemetry.LevelError)) != 0 {
		t.Fatal("no error records expected")
	}
	// Mutating a returned copy must not change the double.
	recs[0].Attrs[0] = telemetry.String("changed", "x")
	recs[0].Message = "changed"
	if got := l.Records()[0]; got.Message != "one" || got.Attrs[0].Key() != "k" {
		t.Fatalf("record was mutated through a copy: %+v", got)
	}
}

func TestLoggerClearKeepsMinLevel(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	l := telemetrytest.NewLogger(telemetry.LevelWarn)
	l.Log(ctx, telemetry.LevelError, "m", attrs(telemetry.Attr{}))
	if len(l.Violations()) != 1 {
		t.Fatalf("violations = %v", l.Violations())
	}
	l.Clear()
	if len(l.Records()) != 0 || len(l.Violations()) != 0 || l.Enabled(ctx, telemetry.LevelInfo) {
		t.Fatal("Clear must drop records and violations but keep the min level")
	}
	l.SetMinLevel(telemetry.LevelTrace)
	if !l.Enabled(ctx, telemetry.LevelTrace) {
		t.Fatal("SetMinLevel not applied")
	}
}

func TestLoggerInvalidAttrIsReportedAndDropped(t *testing.T) {
	t.Parallel()
	l := new(telemetrytest.Logger)
	l.Log(context.Background(), telemetry.LevelInfo, "m", attrs(telemetry.String("", "v"), telemetry.String("ok", "v")))
	recs := l.Records()
	if len(recs) != 1 || len(recs[0].Attrs) != 1 || recs[0].Attrs[0].Key() != "ok" {
		t.Fatalf("records = %+v", recs)
	}
	if v := l.Violations(); len(v) != 1 || !errors.Is(v[0], telemetrytest.ErrInvalidAttr) {
		t.Fatalf("violations = %v", v)
	}
}

func TestLoggerConcurrent(t *testing.T) {
	t.Parallel()
	l := new(telemetrytest.Logger)
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Log(context.Background(), telemetry.LevelInfo, "m", nil)
			_ = l.Records()
		}()
	}
	wg.Wait()
	if len(l.Records()) != 20 {
		t.Fatalf("records = %d, want 20", len(l.Records()))
	}
}
