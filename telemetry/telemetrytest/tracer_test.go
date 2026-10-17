package telemetrytest_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

const spanName = "srd5e.test.run"

func TestTracerRecordsSpan(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	boom := errors.New("boom")
	_, span := tr.Start(context.Background(), spanName, telemetry.String("a", "1"), telemetry.Int("n", 1))
	span.SetAttributes(telemetry.String("a", "2"), telemetry.Bool("b", true))
	span.RecordError(boom)
	span.RecordError(nil)
	span.SetStatus(telemetry.StatusOK)
	span.SetStatus(telemetry.StatusError) // last call wins
	span.End()
	span.End() // idempotent

	spans := tr.Spans()
	if len(spans) != 1 {
		t.Fatalf("spans = %d", len(spans))
	}
	s := spans[0]
	if s.Name != spanName || s.ID != 1 || s.ParentID != 0 || !s.Ended || s.Status != telemetry.StatusError {
		t.Fatalf("span = %+v", s)
	}
	if len(s.Errors) != 1 || s.Errors[0] != boom { //nolint:errorlint // identity is the point
		t.Fatalf("errors = %v", s.Errors)
	}
	if v, ok := s.AttrValue("a"); !ok || v != "2" {
		t.Fatalf("attribute a = %v (later value must replace)", v)
	}
	if len(s.Attrs) != 3 || s.Attrs[0].Key() != "a" {
		t.Fatalf("attrs = %v (replace keeps first-set order)", s.Attrs)
	}
	if _, ok := s.AttrValue("missing"); ok {
		t.Fatal("missing attribute reported present")
	}
	telemetrytest.RequireNoViolations(t, tr)
}

func TestTracerDefaultStatusIsUnset(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	_, span := tr.Start(context.Background(), spanName)
	span.End()
	if s := tr.Spans()[0]; s.Status != telemetry.StatusUnset || len(s.Errors) != 0 || len(s.Attrs) != 0 {
		t.Fatalf("span = %+v", s)
	}
}

func TestTracerParentLinkage(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	other := new(telemetrytest.Tracer)
	ctx, parent := tr.Start(context.Background(), "srd5e.test.parent")
	cctx, child := tr.Start(ctx, "srd5e.test.child")
	_, grand := tr.Start(cctx, "srd5e.test.grand")
	_, sibling := tr.Start(ctx, "srd5e.test.sibling")
	_, foreign := other.Start(ctx, "srd5e.test.foreign") // another tracer's span is not a parent
	for _, s := range []telemetry.Span{grand, child, sibling, foreign, parent} {
		s.End()
	}
	got := map[string]int{}
	for _, s := range tr.Spans() {
		got[s.Name] = s.ParentID
	}
	want := map[string]int{"srd5e.test.parent": 0, "srd5e.test.child": 1, "srd5e.test.grand": 2, "srd5e.test.sibling": 1}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parent of %s = %d, want %d", k, got[k], v)
		}
	}
	if other.Spans()[0].ParentID != 0 {
		t.Error("foreign span must be top-level")
	}
	if tr.Spans()[0].Name != "srd5e.test.parent" {
		t.Error("a parent must precede its children")
	}
}

func TestTracerViolations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		do   func(*telemetrytest.Tracer)
		want error
		recs int
	}{
		{"invalid name", func(tr *telemetrytest.Tracer) { _, s := tr.Start(context.Background(), "bad"); s.End() }, telemetrytest.ErrInvalidName, 0},
		{"invalid start attr", func(tr *telemetrytest.Tracer) {
			_, s := tr.Start(context.Background(), spanName, telemetry.Attr{})
			s.End()
		}, telemetrytest.ErrInvalidAttr, 1},
		{"invalid attr later", func(tr *telemetrytest.Tracer) {
			_, s := tr.Start(context.Background(), spanName)
			s.SetAttributes(telemetry.String("", "x"))
			s.End()
		}, telemetrytest.ErrInvalidAttr, 1},
		{"SetAttributes after End", func(tr *telemetrytest.Tracer) {
			_, s := tr.Start(context.Background(), spanName)
			s.End()
			s.SetAttributes(telemetry.String("k", "v"))
		}, telemetrytest.ErrSpanEnded, 1},
		{"SetStatus after End", func(tr *telemetrytest.Tracer) {
			_, s := tr.Start(context.Background(), spanName)
			s.End()
			s.SetStatus(telemetry.StatusError)
		}, telemetrytest.ErrSpanEnded, 1},
		{"RecordError after End", func(tr *telemetrytest.Tracer) {
			_, s := tr.Start(context.Background(), spanName)
			s.End()
			s.RecordError(errors.New("late"))
		}, telemetrytest.ErrSpanEnded, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tr := new(telemetrytest.Tracer)
			tc.do(tr)
			if v := tr.Violations(); len(v) != 1 || !errors.Is(v[0], tc.want) {
				t.Fatalf("violations = %v, want one %v", v, tc.want)
			}
			if len(tr.Spans()) != tc.recs {
				t.Fatalf("spans = %d, want %d", len(tr.Spans()), tc.recs)
			}
		})
	}
}

func TestTracerWriteAfterEndIsIgnored(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	_, s := tr.Start(context.Background(), spanName)
	s.End()
	s.SetAttributes(telemetry.String("k", "v"))
	s.SetStatus(telemetry.StatusError)
	s.RecordError(errors.New("late"))
	if got := tr.Spans()[0]; len(got.Attrs) != 0 || got.Status != telemetry.StatusUnset || len(got.Errors) != 0 {
		t.Fatalf("ended span changed: %+v", got)
	}
}

func TestTracerSpansNamedAndClear(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	for _, n := range []string{"srd5e.test.a", "srd5e.test.b", "srd5e.test.a"} {
		_, s := tr.Start(context.Background(), n)
		s.End()
	}
	if len(tr.SpansNamed("srd5e.test.a")) != 2 || len(tr.SpansNamed("srd5e.test.zzz")) != 0 {
		t.Fatal("SpansNamed filter wrong")
	}
	_, held := tr.Start(context.Background(), "srd5e.test.c")
	tr.Clear()
	held.End() // still works after Clear
	if len(tr.Spans()) != 0 || len(tr.Violations()) != 0 {
		t.Fatal("Clear must forget spans and violations")
	}
}

func TestTracerSnapshotsAreCopies(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	_, s := tr.Start(context.Background(), spanName, telemetry.String("k", "v"))
	s.RecordError(errors.New("e"))
	snap := tr.Spans()
	snap[0].Attrs[0] = telemetry.String("changed", "x")
	snap[0].Errors[0] = nil
	again := tr.Spans()[0]
	if again.Attrs[0].Key() != "k" || again.Errors[0] == nil {
		t.Fatal("snapshot mutation leaked into the tracer")
	}
}

func TestTracerConcurrent(t *testing.T) {
	t.Parallel()
	tr := new(telemetrytest.Tracer)
	ctx, root := tr.Start(context.Background(), "srd5e.test.root")
	var wg sync.WaitGroup
	for range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, s := tr.Start(ctx, spanName)
			s.SetAttributes(telemetry.Int("n", 1))
			s.End()
			_ = tr.Spans()
		}()
	}
	wg.Wait()
	root.End()
	for _, s := range tr.SpansNamed(spanName) {
		if s.ParentID != 1 {
			t.Fatalf("parent = %d, want 1", s.ParentID)
		}
	}
}
