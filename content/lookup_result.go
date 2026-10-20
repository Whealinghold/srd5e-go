package content

// LookupResult is the outcome of [Source.Lookup] and [Catalogue.Lookup].
// Variants: [LookupFound], [LookupNotFound]. Switch over it with no default.
type LookupResult interface{ lookupResult() }

// LookupFound is the outcome when the requested entry exists. Its Entry has the
// requested key. The zero LookupFound holds a nil Entry and is not returned by
// any source.
type LookupFound struct{ Entry Entry }

// LookupNotFound is the outcome when no entry has the requested Key. It is a
// result, not an error.
type LookupNotFound struct{ Key Key }

func (LookupFound) lookupResult()    {}
func (LookupNotFound) lookupResult() {}

// Missing returns the [MissingContent] for the requested key, for a use case to
// carry in its own content-missing result.
func (n LookupNotFound) Missing() MissingContent {
	return NewMissingContent(n.Key.Kind(), n.Key.ID())
}
