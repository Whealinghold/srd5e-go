package contenttest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// recorder is a reporter that counts failures instead of failing the test.
type recorder struct{ failures []string }

func (r *recorder) Helper() {}
func (r *recorder) Errorf(format string, args ...any) {
	r.failures = append(r.failures, fmt.Sprintf(format, args...))
}

func goodSource(t *testing.T, entries []content.Entry) *content.InMemorySource {
	t.Helper()
	s, err := content.NewInMemorySource(entries)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// Deliberately bad adapters. Each embeds a good source and breaks one rule.

type misordered struct{ content.Source }

func (m misordered) ListKind(ctx context.Context, k content.Kind) ([]content.Entry, error) {
	l, err := m.Source.ListKind(ctx, k)
	for i, j := 0, len(l)-1; i < j; i, j = i+1, j-1 {
		l[i], l[j] = l[j], l[i]
	}
	return l, err
}

type unfiltered struct{ content.Source }

func (u unfiltered) ListKind(ctx context.Context, _ content.Kind) ([]content.Entry, error) {
	var out []content.Entry
	for _, k := range allKinds() {
		l, _ := u.Source.ListKind(ctx, k)
		out = append(out, l...)
	}
	return out, nil
}

type aliased struct {
	content.Source
	cache map[content.Kind][]content.Entry
}

func (a aliased) ListKind(ctx context.Context, k content.Kind) ([]content.Entry, error) {
	if l, ok := a.cache[k]; ok {
		return l, nil
	}
	l, err := a.Source.ListKind(ctx, k)
	a.cache[k] = l
	return l, err
}

// kindBlind finds an entry by id under any kind.
type kindBlind struct{ content.Source }

func (b kindBlind) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	for _, k := range allKinds() {
		if r, err := b.Source.Lookup(ctx, content.NewKey(k, key.ID())); err == nil {
			if f, ok := r.(content.LookupFound); ok {
				return f, nil
			}
		}
	}
	return content.LookupNotFound{Key: key}, nil
}

type wrongNotFoundKey struct{ content.Source }

func (w wrongNotFoundKey) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	r, err := w.Source.Lookup(ctx, key)
	if _, ok := r.(content.LookupNotFound); ok {
		return content.LookupNotFound{}, err
	}
	return r, err
}

type nilResult struct{ content.Source }

func (nilResult) Lookup(context.Context, content.Key) (content.LookupResult, error) {
	return nil, nil //nolint:nilnil // the deliberate contract violation under test
}

type spuriousError struct{ content.Source }

func (spuriousError) ListKind(context.Context, content.Kind) ([]content.Entry, error) {
	return nil, errors.New("spurious")
}

// flaky answers each lookup found, then not found.
type flaky struct {
	content.Source
	calls *int
}

func (f flaky) Lookup(ctx context.Context, key content.Key) (content.LookupResult, error) {
	*f.calls++
	if *f.calls%2 == 0 {
		return content.LookupNotFound{Key: key}, nil
	}
	return f.Source.Lookup(ctx, key)
}

// failing adapters: backendErr is the error the suite scripted.

type failing struct {
	lookup func(key content.Key) (content.LookupResult, error)
	list   func(kind content.Kind) ([]content.Entry, error)
}

func (f failing) Lookup(_ context.Context, key content.Key) (content.LookupResult, error) {
	return f.lookup(key)
}

func (f failing) ListKind(_ context.Context, kind content.Kind) ([]content.Entry, error) {
	return f.list(kind)
}

func failingWith(lookup func(content.Key) (content.LookupResult, error), list func(content.Kind) ([]content.Entry, error)) failing {
	return failing{lookup, list}
}

func wrap(q content.Query, err error) error { return &content.SourceError{Query: q, Err: err} }

// Requirement: doc/specs/content-source-port.md FR-028, FR-042, FR-058 - the
// check groups pass conforming sources and report each deliberately bad one
// through the reporter: wrong order, other kinds leaking into a list, an
// aliased slice, an entry of another kind, a wrong key in LookupNotFound, a nil
// result, a spurious error, and unrepeatable results.
// Path: per case.
func TestChecksReportBadSources(t *testing.T) {
	t.Parallel()
	fixture := fixtureEntries()
	calls := 0
	cases := []struct {
		name  string
		src   content.Source
		check func(reporter, content.Source, []content.Entry)
		bad   bool
	}{
		{"conforming lookups", goodSource(t, fixture), checkLookups, false},
		{"conforming lists", goodSource(t, fixture), checkLists, false},
		{"conforming repeatable", goodSource(t, fixture), checkRepeatable, false},
		{"fake source", NewFakeSource(fixture...), checkLists, false},
		{"wrong order", misordered{goodSource(t, fixture)}, checkLists, true},
		{"other kinds leak into a list", unfiltered{goodSource(t, fixture)}, checkLists, true},
		{"aliased slice", aliased{goodSource(t, fixture), map[content.Kind][]content.Entry{}}, checkRepeatable, true},
		{"entry of another kind", kindBlind{goodSource(t, fixture)}, checkLookups, true},
		{"not found carries the wrong key", wrongNotFoundKey{goodSource(t, fixture)}, checkLookups, true},
		{"nil result with nil error", nilResult{goodSource(t, fixture)}, checkLookups, true},
		{"spurious error on a list", spuriousError{goodSource(t, fixture)}, checkLists, true},
		{"unrepeatable lookup", flaky{goodSource(t, fixture), &calls}, checkRepeatable, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			c.check(rec, c.src, fixture)
			if got := len(rec.failures) > 0; got != c.bad {
				t.Errorf("reported failures = %v (%v), want failing: %v", rec.failures, got, c.bad)
			}
		})
	}
}

// Requirement: doc/specs/content-source-port.md FR-017, FR-018, FR-042 - the
// failing-backend group fails an adapter that leaks a raw error, returns
// LookupNotFound or an empty slice, panics, drops the backend error, names the
// wrong query or leaves Query nil; and passes the conforming forms.
// Path: unhappy (collaborator failure).
func TestFailingGroupReportsBadAdapters(t *testing.T) {
	t.Parallel()
	fixture := fixtureEntries()
	backendErr := errors.New("backend down")
	key := fixture[0].Key()
	kind := key.Kind()
	lq, kq := content.LookupQuery{Key: key}, content.ListKindQuery{Kind: kind}
	notFound := func(k content.Key) (content.LookupResult, error) { return content.LookupNotFound{Key: k}, nil }
	empty := func(content.Kind) ([]content.Entry, error) { return []content.Entry{}, nil }
	okLookup := func(k content.Key) (content.LookupResult, error) {
		return nil, wrap(content.LookupQuery{Key: k}, backendErr)
	}
	okList := func(k content.Kind) ([]content.Entry, error) {
		return nil, wrap(content.ListKindQuery{Kind: k}, backendErr)
	}

	cases := []struct {
		name string
		src  content.Source
		bad  bool
	}{
		{"conforming", failingWith(okLookup, okList), false},
		{"fake source", NewFakeSource(fixture...).FailAll(backendErr), false},
		{"raw error leaked from lookup", failingWith(func(content.Key) (content.LookupResult, error) { return nil, backendErr }, okList), true},
		{"raw error leaked from list", failingWith(okLookup, func(content.Kind) ([]content.Entry, error) { return nil, backendErr }), true},
		{"lookup not found on failure", failingWith(notFound, okList), true},
		{"empty slice on failure", failingWith(okLookup, empty), true},
		{"empty slice with an error", failingWith(okLookup, func(k content.Kind) ([]content.Entry, error) {
			return []content.Entry{}, wrap(kq, backendErr)
		}), true},
		{"result returned with the error", failingWith(func(k content.Key) (content.LookupResult, error) {
			return content.LookupNotFound{Key: k}, wrap(lq, backendErr)
		}, okList), true},
		{"lookup panics", failingWith(func(content.Key) (content.LookupResult, error) { panic(backendErr) }, okList), true},
		{"list panics", failingWith(okLookup, func(content.Kind) ([]content.Entry, error) { panic(backendErr) }), true},
		{"backend error dropped", failingWith(func(k content.Key) (content.LookupResult, error) {
			return nil, wrap(content.LookupQuery{Key: k}, errors.New("other"))
		}, okList), true},
		{"wrong query", failingWith(func(content.Key) (content.LookupResult, error) {
			return nil, wrap(kq, backendErr)
		}, okList), true},
		{"nil query", failingWith(func(content.Key) (content.LookupResult, error) {
			return nil, wrap(nil, backendErr)
		}, okList), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := &recorder{}
			checkFailing(rec, c.src, fixture, backendErr)
			if got := len(rec.failures) > 0; got != c.bad {
				t.Errorf("reported failures = %v, want failing: %v", rec.failures, c.bad)
			}
		})
	}
}

// Requirement: doc/specs/content-source-port.md FR-011 - the ordering table is
// the spec's (byte order: digit before underscore before letter, a prefix
// first, other kinds ignored).
// Path: unhappy (boundary).
func TestOrderingTableMatchesSpec(t *testing.T) {
	t.Parallel()
	ids := func(es []content.Entry) (out []string) {
		for _, e := range es {
			out = append(out, e.Key().ID().String())
		}
		return out
	}
	want := map[string][]string{
		"already sorted":                        {"a", "b", "c"},
		"reverse insertion":                     {"a", "b", "c"},
		"digit before underscore before letter": {"a1", "a_b", "ab"},
		"prefix first":                          {"a", "ab"},
		"other kind ignored":                    {"a"},
		"single entry":                          {"a"},
	}
	cases := orderingCases()
	if len(cases) != len(want) {
		t.Fatalf("%d ordering cases, want %d", len(cases), len(want))
	}
	for _, c := range cases {
		got := ids(ordered(c.entries, content.KindSpell))
		if fmt.Sprint(got) != fmt.Sprint(want[c.name]) {
			t.Errorf("%s: ordered = %v, want %v", c.name, got, want[c.name])
		}
	}
}
