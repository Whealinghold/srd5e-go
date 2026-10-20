package content

import "context"

// Port is the srderr port name of every [Source] failure. The error type of a
// wrapped failure is "content_error".
const Port = "content"

// Source is where the engine gets content from. Calls may do I/O, so they take
// a context.
//
// Expected absence is a value: [LookupNotFound] from Lookup, an empty slice
// from ListKind. Infrastructure failure is the returned error, which a source
// builds as a *[SourceError] and the caller wraps once with srderr.WrapPort or
// srderr.GuardValue (Port, "lookup" or "list_kind"). A source returns a failure
// instead of panicking, with the zero result.
//
// Implementations are safe for concurrent use unless their doc says otherwise.
// Adapter authors reuse the contract suites of package contenttest
// (contenttest.RunSourceContract for sources built from arbitrary entries,
// contenttest.RunReadOnlyContract for fixed fixtures).
type Source interface {
	// Lookup returns [LookupFound] for the entry whose key equals key and
	// [LookupNotFound] with a nil error when there is none. It never returns a
	// nil result together with a nil error.
	Lookup(ctx context.Context, key Key) (LookupResult, error)
	// ListKind returns every entry of kind ordered by ID (byte order), as a
	// copy the caller may modify; empty and nil error when there are none.
	ListKind(ctx context.Context, kind Kind) ([]Entry, error)
}
