package content

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Decoder is the port that turns the bytes of a content document into a
// [Set]: synchronous and in memory, so it takes no context (loading bytes is
// the caller's job and cancellation is not honoured while it runs). Content
// problems are returned inside its error, as an errors.Join of [Problem]
// values read with [Problems], and nothing else is; it never panics for
// content. A nil error means a valid document, an empty Set included.
// [JSONDecoder] is the shipped implementation; contenttest.FakeDecoder and
// contenttest.RunDecoderContract are its test support.
type Decoder interface {
	Decode(document []byte) (Set, error)
}

// LoadSetCommand is the input of [Loader.LoadSet]: the bytes of one document.
// Construct it with a keyed literal. The Loader neither retains nor modifies
// Document.
type LoadSetCommand struct {
	Document []byte
}

// LoadSetResult is the closed set of outcomes of [Loader.LoadSet]. Variants:
// LoadSetLoaded, LoadSetRejected.
type LoadSetResult interface{ isLoadSetResult() }

// LoadSetLoaded is the outcome for a document the decoder accepted. The zero
// value holds the empty Set. Safe for concurrent use.
type LoadSetLoaded struct{ set Set }

func (LoadSetLoaded) isLoadSetResult() {}

// Set returns the decoded entries.
func (r LoadSetLoaded) Set() Set { return r.set }

// Equal reports whether r and o hold equal sets.
func (r LoadSetLoaded) Equal(o LoadSetLoaded) bool { return r.set.Equal(o.set) }

// LoadSetRejected is the outcome for a document the decoder rejected. Only
// [Loader.LoadSet] builds one, from a non-empty list of problems; the zero
// value holds none. Safe for concurrent use.
type LoadSetRejected struct{ problems []Problem }

func (LoadSetRejected) isLoadSetResult() {}

// Problems returns every problem found, in the decoder's order, as a new slice
// the caller may modify.
func (r LoadSetRejected) Problems() []Problem { return slices.Clone(r.problems) }

// Equal reports whether r and o hold the same problems in the same order.
func (r LoadSetRejected) Equal(o LoadSetRejected) bool { return slices.Equal(r.problems, o.problems) }

// LoaderProblem says why a [Loader] cannot work. The zero value is not a
// problem.
type LoaderProblem int

// The loader problems.
const (
	// LoaderMissingDecoder: [NewLoader] got a nil Decoder, or the Loader is the
	// zero value.
	LoaderMissingDecoder LoaderProblem = 1
)

// String returns the lower camel case name of p, or "unknown" for an
// undefined value.
func (p LoaderProblem) String() string {
	if p == LoaderMissingDecoder {
		return "missingDecoder"
	}
	return unknownName
}

// LoaderError reports a misconfigured [Loader]. It matches
// srderr.ErrInvalidInput with errors.Is.
type LoaderError struct {
	Problem LoaderProblem
}

// Error renders the problem. The text is not a contract.
func (e *LoaderError) Error() string { return "content: invalid loader: " + e.Problem.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *LoaderError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *LoaderError with the same Problem.
func (e *LoaderError) Is(target error) bool {
	t, ok := target.(*LoaderError)
	return ok && t != nil && t.Problem == e.Problem
}

// Loader is the use case that loads one content document through a [Decoder]:
// it calls the port once, returns the [LoadSetResult] and emits the six
// srd5e.content.* signals. It holds no rules logic and no state beyond its
// injected ports, and calls no other use case; [ValidateSet] is a separate
// pure function the caller runs on the loaded set or the merged sets.
//
// Create it with [NewLoader]; the zero value is not usable. A Loader is safe
// for concurrent use only if its Decoder, telemetry and clock are.
type Loader struct {
	decoder      Decoder
	tel          telemetry.Telemetry
	clock        func() time.Time
	loads        telemetry.Counter
	duration     telemetry.Histogram
	entries      telemetry.Counter
	decodeErrors telemetry.Counter
}

// Option configures a Loader.
type Option func(*Loader)

// WithTelemetry sets the telemetry the Loader emits to. Nil members of t
// default to no-ops. Without this option the Loader is silent. The members
// MUST be safe for concurrent use if the Loader is shared across goroutines.
func WithTelemetry(t telemetry.Telemetry) Option {
	return func(l *Loader) { l.tel = t }
}

// WithClock sets the clock that times calls for the duration histogram; the
// default is time.Now. A nil clock is ignored. It MUST be safe for concurrent
// use if the Loader is shared across goroutines.
func WithClock(now func() time.Time) Option {
	return func(l *Loader) {
		if now != nil {
			l.clock = now
		}
	}
}

// NewLoader returns a Loader that decodes through d. The telemetry
// instruments are created once, here.
//
// It returns a *[LoaderError] with LoaderMissingDecoder (matching
// srderr.ErrInvalidInput) when d is nil.
func NewLoader(d Decoder, opts ...Option) (Loader, error) {
	if d == nil {
		return Loader{}, &LoaderError{Problem: LoaderMissingDecoder}
	}
	l := Loader{decoder: d, clock: time.Now}
	for _, opt := range opts {
		opt(&l)
	}
	l.tel = l.tel.WithDefaults()
	m := l.tel.Meter
	l.loads = m.Counter(telemetry.Descriptor{Name: MetricLoads, Unit: "{load}", Description: "Content documents loaded, by outcome."})
	l.duration = m.Histogram(telemetry.Descriptor{Name: MetricLoadDuration, Unit: "s", Description: "Duration of loading one content document."})
	l.entries = m.Counter(telemetry.Descriptor{Name: MetricEntriesLoaded, Unit: "{entry}", Description: "Entries of loaded content documents, by kind."})
	l.decodeErrors = m.Counter(telemetry.Descriptor{Name: MetricDecodeErrors, Unit: "{problem}", Description: "Problems of rejected content documents, by problem kind."})
	return l, nil
}

// decoded is what one call of the Decoder port yielded: a set, or problems.
type decoded struct {
	set      Set
	problems []Problem
}

// LoadSet decodes cmd.Document through the Decoder once and returns its
// result: a [LoadSetLoaded] holding the decoded Set for a nil error, or a
// [LoadSetRejected] holding the problems (never empty) for an error that holds
// problems. A rejected document is a result, not an error.
//
// It returns ctx.Err() unwrapped, before any signal, when ctx is already
// cancelled; cancellation after that point is not observed, as the Decoder
// takes no context. It returns a *[LoaderError] before any signal for the zero
// Loader. It returns a *srderr.PortError (srderr.ErrorType "content_error"),
// with a nil result, when the Decoder fails as a port: it panics (the cause
// matches srderr.ErrPortPanic), or returns an error that holds no problem (the
// cause also matches srderr.ErrContractViolation); the failure is wrapped
// once. A failure is recorded once on both spans and on the loads counter,
// never logged; the panic value stays out of telemetry.
func (l Loader) LoadSet(ctx context.Context, cmd LoadSetCommand) (LoadSetResult, error) {
	if l.decoder == nil {
		return nil, &LoaderError{Problem: LoaderMissingDecoder}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	ctx, span := l.tel.Tracer.Start(ctx, SpanLoad)
	defer span.End()
	timer := telemetry.StartTimer(l.clock, l.duration)
	defer timer.Stop(ctx)

	out, err := l.decode(ctx, cmd.Document)
	failure := telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure)
	switch {
	case err != nil:
		markFailed(span, err)
		l.loads.Add(ctx, 1, failure, telemetry.String(telemetry.AttrErrorType, srderr.ErrorType(err)))
		return nil, err
	case len(out.problems) > 0: // equivalent mutant: a length is never negative, so > and != agree
		markRejected(span)
		l.loads.Add(ctx, 1, failure)
		l.countProblems(ctx, out.problems)
		return LoadSetRejected{problems: out.problems}, nil
	}
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess))
	l.loads.Add(ctx, 1, telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess))
	l.countEntries(ctx, out.set)
	return LoadSetLoaded{set: out.set}, nil
}

// decode calls the Decoder port inside the decode span, converting a port
// failure to a *srderr.PortError (FR-086).
func (l Loader) decode(ctx context.Context, document []byte) (decoded, error) {
	_, span := l.tel.Tracer.Start(ctx, SpanDecode)
	defer span.End()
	out, err := srderr.GuardValue(Port, "decode", func() (decoded, error) {
		set, err := l.decoder.Decode(document)
		if err == nil {
			return decoded{set: set}, nil
		}
		if problems := Problems(err); len(problems) > 0 { // equivalent mutant: a length is never negative, so > and != agree
			return decoded{problems: problems}, nil
		}
		return decoded{}, errors.Join(&srderr.ContractError{Detail: "decoder returned an error that holds no problem"}, err)
	})
	switch {
	case err != nil:
		markFailed(span, err)
	case len(out.problems) > 0: // equivalent mutant: a length is never negative, so > and != agree
		markRejected(span)
	default:
		span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeSuccess))
	}
	return out, err
}

// recordedError is the error a span records for a port failure: only the
// error type, never the message, so a recovered panic value (which may hold
// document bytes) does not reach telemetry (schema FR-085).
type recordedError string

func (e recordedError) Error() string { return string(e) }

func markFailed(span telemetry.Span, err error) {
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
	span.RecordError(recordedError(srderr.ErrorType(err)))
	span.SetStatus(telemetry.StatusError)
}

func markRejected(span telemetry.Span) {
	span.SetAttributes(telemetry.String(telemetry.AttrOutcome, telemetry.OutcomeFailure))
	span.SetStatus(telemetry.StatusError)
}

// countProblems adds the problems of a rejected document to decode_errors,
// once per problem kind with the count of that kind (not once per problem),
// in the kind order of the schema.
func (l Loader) countProblems(ctx context.Context, problems []Problem) {
	counts := map[ProblemKind]int64{}
	for _, p := range problems {
		counts[p.kind]++
	}
	for k := ProblemSyntax; k <= ProblemConstraintViolation; k++ {
		if n := counts[k]; n > 0 { // equivalent mutant: a count is never negative, so > and != agree
			l.decodeErrors.Add(ctx, n, telemetry.String(AttrProblemKind, problemKindValue(k)))
		}
	}
}

// countEntries adds the entries of a loaded set to entries_loaded, once per
// kind that has any, in kind order.
func (l Loader) countEntries(ctx context.Context, set Set) {
	counts := map[Kind]int64{}
	for _, e := range set.entries {
		counts[e.Key().Kind()]++
	}
	for k := KindSpecies; k.Valid(); k++ {
		if n := counts[k]; n > 0 { // equivalent mutant: a count is never negative, so > and != agree
			l.entries.Add(ctx, n, telemetry.String(AttrKind, kindValue(k)))
		}
	}
}
