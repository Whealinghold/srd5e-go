package contenttest

import (
	"cmp"
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
)

// reporter is the part of testing.TB the checks use, so the checks can be
// tested against a recording double.
type reporter interface {
	Helper()
	Errorf(format string, args ...any)
}

// entry is the synthetic Entry the suites build their fixtures from.
type entry struct{ key content.Key }

func (e entry) Key() content.Key { return e.key }

func ent(kind content.Kind, id string) content.Entry {
	return entry{key: content.NewKey(kind, content.MustID(id))}
}

// fixtureEntries returns the entries the main groups run against: spells, the same id under two kinds, and entries of other kinds.
func fixtureEntries() []content.Entry {
	return []content.Entry{
		ent(content.KindSpell, "b"), ent(content.KindSpell, "a"), ent(content.KindFeat, "b"),
		ent(content.KindSpecies, "c"), ent(content.KindMonster, "z_1"),
	}
}

// allKinds returns every valid kind in order.
func allKinds() []content.Kind {
	var out []content.Kind
	for k := content.KindSpecies; k.Valid(); k++ {
		out = append(out, k)
	}
	return out
}

func sameEntries(a, b []content.Entry) bool {
	return slices.EqualFunc(a, b, func(x, y content.Entry) bool { return reflect.DeepEqual(x, y) })
}

// ordered returns the entries of kind sorted by ID in byte order.
func ordered(entries []content.Entry, kind content.Kind) []content.Entry {
	out := []content.Entry{}
	for _, e := range entries {
		if e.Key().Kind() == kind {
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b content.Entry) int {
		return cmp.Compare(a.Key().ID().String(), b.Key().ID().String())
	})
	return out
}

// absentKey returns a spell key whose id no entry uses, under any kind.
func absentKey(entries []content.Entry) content.Key {
	id := "absent"
	for slices.ContainsFunc(entries, func(e content.Entry) bool { return e.Key().ID().String() == id }) {
		id += "x"
	}
	return content.NewKey(content.KindSpell, content.MustID(id))
}

// RunSourceContract runs the behavioural contract of [content.Source] against
// sources built by create from arbitrary entries, one fresh source per
// subtest. create must serve exactly the entries it is given, so a source
// whose content is fixed does not qualify (see [RunReadOnlyContract]).
//
// It checks that Lookup finds each entry and returns [content.LookupNotFound]
// with a nil error for an absent key, for an id that exists only under another
// kind, and on an empty source; that the same id under two kinds is two
// entries; that ListKind returns the entries of one kind ordered by ID in byte
// order, empty with a nil error when there are none, as a copy, and the same
// on every call.
//
// createFailing may be nil. When given, it must return a source whose backend
// fails with backendErr on every call; the suite then checks that neither
// method panics and that each returns a nil result and a *[content.SourceError]
// carrying the right [content.Query] that wraps backendErr, never
// LookupNotFound or an empty slice.
func RunSourceContract(t *testing.T,
	create func(t *testing.T, entries []content.Entry) content.Source,
	createFailing func(t *testing.T, entries []content.Entry, backendErr error) content.Source,
) {
	t.Helper()
	fixture := fixtureEntries()
	t.Run("lookup and list", func(t *testing.T) {
		src := create(t, fixture)
		checkLookups(t, src, fixture)
		checkLists(t, src, fixture)
	})
	t.Run("repeat calls and copies", func(t *testing.T) {
		checkRepeatable(t, create(t, fixture), fixture)
	})
	for _, c := range orderingCases() {
		t.Run("ordering "+c.name, func(t *testing.T) {
			checkLists(t, create(t, c.entries), c.entries)
		})
	}
	t.Run("empty source", func(t *testing.T) {
		src := create(t, nil)
		checkLookups(t, src, nil)
		checkLists(t, src, nil)
	})
	if createFailing != nil {
		t.Run("failing backend", func(t *testing.T) {
			backendErr := errors.New("backend down")
			checkFailing(t, createFailing(t, fixture, backendErr), fixture, backendErr)
		})
	}
}

// RunReadOnlyContract runs the read-only part of the [content.Source] contract
// against source, a source with fixed content that is exactly expectedEntries
// (in any order). It checks lookup (found, absent key, an id present only under
// another kind), ListKind ordering and kind filtering, copies and repeatable
// results, and does not build sources, so it has no empty-source, same-id-under-
// two-kinds or failing-backend group.
func RunReadOnlyContract(t *testing.T, source content.Source, expectedEntries []content.Entry) {
	t.Helper()
	t.Run("lookup", func(t *testing.T) { checkLookups(t, source, expectedEntries) })
	t.Run("list kind", func(t *testing.T) { checkLists(t, source, expectedEntries) })
	t.Run("repeat calls and copies", func(t *testing.T) { checkRepeatable(t, source, expectedEntries) })
}

type orderingCase struct {
	name    string
	entries []content.Entry
}

// orderingCases is the ListKind ordering table of the source-port spec.
func orderingCases() []orderingCase {
	spells := func(ids ...string) []content.Entry {
		var out []content.Entry
		for _, id := range ids {
			out = append(out, ent(content.KindSpell, id))
		}
		return out
	}
	return []orderingCase{
		{"already sorted", spells("a", "b", "c")},
		{"reverse insertion", spells("c", "b", "a")},
		{"digit before underscore before letter", spells("ab", "a_b", "a1")},
		{"prefix first", spells("ab", "a")},
		{"other kind ignored", []content.Entry{ent(content.KindSpell, "a"), ent(content.KindFeat, "b")}},
		{"single entry", spells("a")},
	}
}

func checkLookups(r reporter, src content.Source, entries []content.Entry) {
	r.Helper()
	ctx := context.Background()
	for _, e := range entries {
		res, err := src.Lookup(ctx, e.Key())
		found, ok := res.(content.LookupFound)
		switch {
		case err != nil:
			r.Errorf("Lookup(%v) error = %v, want nil", e.Key(), err)
		case !ok:
			r.Errorf("Lookup(%v) = %T, want LookupFound", e.Key(), res)
		case found.Entry == nil || found.Entry.Key() != e.Key() || !reflect.DeepEqual(found.Entry, e):
			r.Errorf("Lookup(%v) = %v, want the entry with that key", e.Key(), found.Entry)
		}
		// The same id under every other kind is a different key: not found
		// unless the fixture holds it.
		for _, k := range allKinds() {
			other := content.NewKey(k, e.Key().ID())
			if !slices.ContainsFunc(entries, func(x content.Entry) bool { return x.Key() == other }) {
				checkNotFound(r, src, other)
			}
		}
	}
	checkNotFound(r, src, absentKey(entries))
}

func checkNotFound(r reporter, src content.Source, key content.Key) {
	r.Helper()
	res, err := src.Lookup(context.Background(), key)
	nf, ok := res.(content.LookupNotFound)
	switch {
	case err != nil:
		r.Errorf("Lookup(%v) of an absent key: error = %v, want nil", key, err)
	case !ok:
		r.Errorf("Lookup(%v) of an absent key = %T, want LookupNotFound", key, res)
	case nf.Key != key || nf.Missing() != content.NewMissingContent(key.Kind(), key.ID()):
		r.Errorf("Lookup(%v) = LookupNotFound for key %v, want the requested key", key, nf.Key)
	}
}

func checkLists(r reporter, src content.Source, entries []content.Entry) {
	r.Helper()
	for _, k := range allKinds() {
		got, err := src.ListKind(context.Background(), k)
		want := ordered(entries, k)
		switch {
		case err != nil:
			r.Errorf("ListKind(%v) error = %v, want nil", k, err)
		case !sameEntries(got, want):
			r.Errorf("ListKind(%v) = %v, want %v (ordered by ID)", k, got, want)
		}
	}
}

// checkRepeatable checks that results are equal on repeated calls and that a
// returned slice is a copy.
func checkRepeatable(r reporter, src content.Source, entries []content.Entry) {
	r.Helper()
	ctx := context.Background()
	for _, e := range entries {
		a, errA := src.Lookup(ctx, e.Key())
		b, errB := src.Lookup(ctx, e.Key())
		if errA != nil || errB != nil || !reflect.DeepEqual(a, b) {
			r.Errorf("Lookup(%v) twice = (%v, %v) and (%v, %v), want equal results", e.Key(), a, errA, b, errB)
		}
	}
	for _, k := range allKinds() {
		first, err := src.ListKind(ctx, k)
		if err != nil {
			r.Errorf("ListKind(%v) error = %v, want nil", k, err)
			continue
		}
		snapshot := slices.Clone(first)
		clear(first) // a caller scribbling on the result must not reach the source
		again, err := src.ListKind(ctx, k)
		if err != nil || !sameEntries(again, snapshot) {
			r.Errorf("ListKind(%v) after mutating an earlier result = (%v, %v), want %v", k, again, err, snapshot)
		}
	}
}

func checkFailing(r reporter, src content.Source, entries []content.Entry, backendErr error) {
	r.Helper()
	ctx := context.Background()
	key := entries[0].Key()
	var (
		res  content.LookupResult
		err  error
		list []content.Entry
	)
	if guard(r, "Lookup", func() { res, err = src.Lookup(ctx, key) }) {
		checkSourceError(r, "Lookup", err, content.LookupQuery{Key: key}, backendErr)
		if res != nil {
			r.Errorf("Lookup on a failing backend returned %T with its error, want a nil result", res)
		}
	}
	kind := key.Kind()
	if guard(r, "ListKind", func() { list, err = src.ListKind(ctx, kind) }) {
		checkSourceError(r, "ListKind", err, content.ListKindQuery{Kind: kind}, backendErr)
		if list != nil {
			r.Errorf("ListKind on a failing backend returned %v with its error, want a nil slice", list)
		}
	}
}

func checkSourceError(r reporter, op string, err error, want content.Query, backendErr error) {
	r.Helper()
	var se *content.SourceError
	switch {
	case err == nil:
		r.Errorf("%s on a failing backend returned a nil error", op)
	case !errors.As(err, &se):
		r.Errorf("%s on a failing backend returned %T, want *content.SourceError", op, err)
	case se.Query != want:
		r.Errorf("%s: SourceError.Query = %v, want %v", op, se.Query, want)
	case !errors.Is(err, backendErr):
		r.Errorf("%s: the returned error does not wrap the backend error", op)
	}
}

// guard runs fn and reports a panic as a contract failure; it returns false
// when fn panicked.
func guard(r reporter, op string, fn func()) (ok bool) {
	r.Helper()
	defer func() {
		if p := recover(); p != nil {
			r.Errorf("%s panicked: %v", op, p)
			ok = false
		}
	}()
	fn()
	return true
}
