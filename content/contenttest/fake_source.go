package contenttest

import (
	"context"
	"slices"
	"sync"

	"github.com/Whealinghold/srd5e-go/content"
)

var _ content.Source = (*FakeSource)(nil)

// FakeSource is an in-memory [content.Source] double: it answers from a
// [content.Catalogue], records every request, and can be scripted to fail.
// Build it with [NewFakeSource]; configure failure before sharing it. Calls are
// safe for concurrent use; the Fail methods are not.
type FakeSource struct {
	catalogue content.Catalogue
	lookupErr error
	listErr   error

	mu      sync.Mutex
	queries []content.Query
}

// NewFakeSource returns a fake serving entries. It panics with the error of
// [content.MustCatalogue] on a duplicate key, a nil entry or an invalid key.
func NewFakeSource(entries ...content.Entry) *FakeSource {
	return &FakeSource{catalogue: content.MustCatalogue(entries)}
}

// FailLookup scripts every Lookup to fail with err, wrapped in a
// *[content.SourceError] that carries the query. A nil err removes the script.
func (f *FakeSource) FailLookup(err error) *FakeSource {
	f.lookupErr = err
	return f
}

// FailListKind scripts every ListKind to fail with err, as FailLookup does.
func (f *FakeSource) FailListKind(err error) *FakeSource {
	f.listErr = err
	return f
}

// FailAll scripts both Lookup and ListKind to fail with err.
func (f *FakeSource) FailAll(err error) *FakeSource {
	return f.FailLookup(err).FailListKind(err)
}

// Queries returns a copy of the requests seen so far, in call order,
// including those that failed.
func (f *FakeSource) Queries() []content.Query {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.queries)
}

func (f *FakeSource) record(q content.Query) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries = append(f.queries, q)
}

// Lookup records the query, then fails if scripted, else answers from the
// catalogue. It does not consult ctx.
func (f *FakeSource) Lookup(_ context.Context, key content.Key) (content.LookupResult, error) {
	q := content.LookupQuery{Key: key}
	f.record(q)
	if f.lookupErr != nil {
		return nil, &content.SourceError{Query: q, Err: f.lookupErr}
	}
	return f.catalogue.Lookup(key), nil
}

// ListKind records the query, then fails if scripted, else answers from the
// catalogue. It does not consult ctx.
func (f *FakeSource) ListKind(_ context.Context, kind content.Kind) ([]content.Entry, error) {
	q := content.ListKindQuery{Kind: kind}
	f.record(q)
	if f.listErr != nil {
		return nil, &content.SourceError{Query: q, Err: f.listErr}
	}
	return f.catalogue.OfKind(kind), nil
}
