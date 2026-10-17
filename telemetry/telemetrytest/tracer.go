package telemetrytest

import (
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/Whealinghold/srd5e-go/telemetry"
)

// SpanRecord is a snapshot of one span stored by a Tracer.
type SpanRecord struct {
	// ID is the 1-based start order of the span within its Tracer.
	ID int
	// ParentID is the ID of the span whose context this one was started from,
	// or 0 for a top-level span.
	ParentID int
	Name     string
	// Attrs holds the attributes the span started with plus any set later, in
	// first-set order; a later value for a key replaces the earlier one.
	Attrs  []telemetry.Attr
	Status telemetry.Status
	// Errors are the errors recorded on the span, in call order.
	Errors []error
	Ended  bool
}

// AttrValue returns the value of the attribute key and whether it is set.
func (r SpanRecord) AttrValue(key string) (any, bool) {
	for _, a := range r.Attrs {
		if a.Key() == key {
			return a.Value(), true
		}
	}
	return nil, false
}

// Tracer is a telemetry.Tracer that stores one SpanRecord per span.
//
// A span is writable from Start until End; later writes are collected as
// ErrSpanEnded violations and ignored. A name that is not srd5e.<context>.<noun>
// is an ErrInvalidName violation and yields a telemetry.NoopSpan without a
// record. Parent linkage follows the context returned by Start, so nested and
// concurrent use are both correct. No duration is recorded: that needs a clock.
// It is safe for concurrent use; the zero value is ready to use.
type Tracer struct {
	mu    sync.Mutex
	spans []*spanState
	viol  violations
}

var _ telemetry.Tracer = (*Tracer)(nil)

type spanState struct {
	rec SpanRecord
}

type spanRef struct {
	tracer *Tracer
	id     int
}

type spanKey struct{}

// Start records a span and returns a context carrying it as the parent of
// spans started from that context.
func (t *Tracer) Start(ctx context.Context, name string, attrs ...telemetry.Attr) (context.Context, telemetry.Span) {
	if !telemetry.ValidName(name) {
		t.viol.add(fmt.Errorf("%w: span %q", ErrInvalidName, name))
		return ctx, telemetry.NoopSpan{}
	}
	valid := t.viol.validAttrs(attrs)
	parent := 0
	if ref, ok := ctx.Value(spanKey{}).(spanRef); ok && ref.tracer == t {
		parent = ref.id
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	s := &spanState{rec: SpanRecord{ID: len(t.spans) + 1, ParentID: parent, Name: name}}
	s.rec.Attrs = setAttrs(nil, valid)
	t.spans = append(t.spans, s)
	return context.WithValue(ctx, spanKey{}, spanRef{t, s.rec.ID}), &recordingSpan{t: t, s: s}
}

// setAttrs adds attrs to dst, replacing the value of an existing key.
func setAttrs(dst, attrs []telemetry.Attr) []telemetry.Attr {
	for _, a := range attrs {
		if i := slices.IndexFunc(dst, func(d telemetry.Attr) bool { return d.Key() == a.Key() }); i >= 0 {
			dst[i] = a
		} else {
			dst = append(dst, a)
		}
	}
	return dst
}

// Spans returns snapshots of the spans started so far, in start order (a parent
// precedes its children).
func (t *Tracer) Spans() []SpanRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]SpanRecord, len(t.spans))
	for i, s := range t.spans {
		out[i] = s.rec
		out[i].Attrs = slices.Clone(s.rec.Attrs)
		out[i].Errors = slices.Clone(s.rec.Errors)
	}
	return out
}

// SpansNamed returns the spans called name, in start order.
func (t *Tracer) SpansNamed(name string) []SpanRecord {
	var out []SpanRecord
	for _, s := range t.Spans() {
		if s.Name == name {
			out = append(out, s)
		}
	}
	return out
}

// Clear forgets the recorded spans and violations. Spans started earlier keep
// working but are no longer listed.
func (t *Tracer) Clear() {
	t.mu.Lock()
	t.spans = nil
	t.mu.Unlock()
	t.viol.reset()
}

// Violations returns the misuse collected so far.
func (t *Tracer) Violations() []error { return t.viol.snapshot() }

type recordingSpan struct {
	t *Tracer
	s *spanState
}

// write runs f under the tracer lock if the span is open, else reports a
// violation.
func (r *recordingSpan) write(op string, f func(*SpanRecord)) {
	r.t.mu.Lock()
	ended := r.s.rec.Ended
	if !ended {
		f(&r.s.rec)
	}
	r.t.mu.Unlock()
	if ended {
		r.t.viol.add(fmt.Errorf("%w: %s on %s", ErrSpanEnded, op, r.s.rec.Name))
	}
}

func (r *recordingSpan) SetAttributes(attrs ...telemetry.Attr) {
	valid := r.t.viol.validAttrs(attrs)
	r.write("SetAttributes", func(rec *SpanRecord) { rec.Attrs = setAttrs(rec.Attrs, valid) })
}

func (r *recordingSpan) SetStatus(status telemetry.Status) {
	r.write("SetStatus", func(rec *SpanRecord) { rec.Status = status })
}

func (r *recordingSpan) RecordError(err error) {
	if err == nil {
		return
	}
	r.write("RecordError", func(rec *SpanRecord) { rec.Errors = append(rec.Errors, err) })
}

func (r *recordingSpan) End() {
	r.t.mu.Lock()
	defer r.t.mu.Unlock()
	r.s.rec.Ended = true
}
