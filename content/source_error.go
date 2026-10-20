package content

// SourceError is the typed cause of an infrastructure failure in a [Source].
// A source returns a *SourceError; the caller wraps it in a *srderr.PortError,
// so errors.As finds both. Build it with keyed fields:
// &SourceError{Query: q, Err: err}.
//
// Query describes the request and must not be nil; Error and Unwrap tolerate a
// nil Query or Err. Unwrap returns Err unchanged. Callers branch with errors.Is
// and errors.As, never on Error.
type SourceError struct {
	Query Query
	Err   error
}

// Error describes the failure for logs.
func (e *SourceError) Error() string {
	op := "invalid query"
	switch e.Query.(type) {
	case LookupQuery:
		op = "lookup"
	case ListKindQuery:
		op = "list_kind"
	}
	if e.Err == nil {
		return "content: source " + op + " failed"
	}
	return "content: source " + op + " failed: " + e.Err.Error()
}

// Unwrap returns the backend error.
func (e *SourceError) Unwrap() error { return e.Err }
