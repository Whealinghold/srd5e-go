package contenttest_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/contenttest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

type testEntry struct{ key content.Key }

func (e testEntry) Key() content.Key { return e.key }

func entryOf(kind content.Kind, id string) content.Entry {
	return testEntry{content.NewKey(kind, content.MustID(id))}
}

// Requirement: doc/specs/content-source-port.md FR-030 - a scripted failure is
// returned from the scripted operation only, as the same error instance inside
// a *SourceError carrying the query, and never panics.
// Path: unhappy (collaborator failure).
func TestFakeSourceScriptedFailure(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	key := content.NewKey(content.KindSpell, content.MustID("a"))
	entries := []content.Entry{entryOf(content.KindSpell, "a")}
	ctx := context.Background()

	cases := []struct {
		name               string
		script             func(*contenttest.FakeSource) *contenttest.FakeSource
		lookupFails, lFail bool
	}{
		{"lookup only", func(f *contenttest.FakeSource) *contenttest.FakeSource { return f.FailLookup(boom) }, true, false},
		{"list only", func(f *contenttest.FakeSource) *contenttest.FakeSource { return f.FailListKind(boom) }, false, true},
		{"both", func(f *contenttest.FakeSource) *contenttest.FakeSource { return f.FailAll(boom) }, true, true},
		{"cleared", func(f *contenttest.FakeSource) *contenttest.FakeSource { return f.FailAll(boom).FailAll(nil) }, false, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f := c.script(contenttest.NewFakeSource(entries...))

			res, err := f.Lookup(ctx, key)
			var se *content.SourceError
			if c.lookupFails {
				if !errors.As(err, &se) || !errors.Is(se.Err, boom) || se.Query != (content.LookupQuery{Key: key}) || res != nil {
					t.Errorf("Lookup = (%v, %v), want nil and *SourceError{LookupQuery, boom}", res, err)
				}
			} else if err != nil || res != (content.LookupFound{Entry: entries[0]}) {
				t.Errorf("Lookup = (%v, %v), want the entry", res, err)
			}

			list, err := f.ListKind(ctx, content.KindSpell)
			if c.lFail {
				if !errors.As(err, &se) || !errors.Is(se.Err, boom) || se.Query != (content.ListKindQuery{Kind: content.KindSpell}) || list != nil {
					t.Errorf("ListKind = (%v, %v), want nil and *SourceError{ListKindQuery, boom}", list, err)
				}
			} else if err != nil || len(list) != 1 {
				t.Errorf("ListKind = (%v, %v), want one entry", list, err)
			}
		})
	}
}

// Requirement: doc/specs/content-source-port.md FR-031 - FakeSource records
// every request as a Query in call order, failed ones included, and returns the
// record as a copy.
// Path: happy.
func TestFakeSourceRecordsQueries(t *testing.T) {
	t.Parallel()
	f := contenttest.NewFakeSource(entryOf(content.KindFeat, "a")).FailListKind(errors.New("x"))
	ctx := context.Background()
	k1 := content.NewKey(content.KindFeat, content.MustID("a"))
	k2 := content.NewKey(content.KindSpell, content.MustID("zz"))

	_, _ = f.Lookup(ctx, k1)
	_, _ = f.ListKind(ctx, content.KindFeat)
	_, _ = f.Lookup(ctx, k2)

	want := []content.Query{
		content.LookupQuery{Key: k1}, content.ListKindQuery{Kind: content.KindFeat}, content.LookupQuery{Key: k2},
	}
	got := f.Queries()
	if !slices.Equal(got, want) {
		t.Fatalf("Queries() = %v, want %v", got, want)
	}
	clear(got)
	if again := f.Queries(); !slices.Equal(again, want) {
		t.Errorf("Queries() after mutating an earlier copy = %v, want %v", again, want)
	}
	if n := len(contenttest.NewFakeSource().Queries()); n != 0 {
		t.Errorf("fresh FakeSource recorded %d queries", n)
	}
}

// Requirement: doc/specs/content-source-port.md FR-021 - the fake rejects
// duplicate keys through content.MustCatalogue, which panics with the error.
// Path: unhappy (invalid input).
func TestFakeSourcePanicsOnDuplicateKey(t *testing.T) {
	t.Parallel()
	defer func() {
		err, _ := recover().(error)
		var dk *content.DuplicateKeyError
		if !errors.As(err, &dk) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("panic value = %v, want *DuplicateKeyError wrapping ErrInvalidInput", err)
		}
	}()
	contenttest.NewFakeSource(entryOf(content.KindFeat, "a"), entryOf(content.KindFeat, "a"))
	t.Error("NewFakeSource with a duplicate key did not panic")
}

// Requirement: none - pins that FakeSource is safe for concurrent calls and
// records every query.
// Path: happy.
func TestFakeSourceConcurrentCalls(t *testing.T) {
	t.Parallel()
	f := contenttest.NewFakeSource(entryOf(content.KindFeat, "a"))
	const n = 50
	done := make(chan struct{})
	for range n {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = f.ListKind(context.Background(), content.KindFeat)
			_ = f.Queries()
		}()
	}
	for range n {
		<-done
	}
	if got := len(f.Queries()); got != n {
		t.Errorf("recorded %d queries, want %d", got, n)
	}
}
