package combattest

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
)

// CheckEvent checks one event against the combat.Event contract: Kind is
// wantKind, constant across calls and matches [a-z][a-z0-9_]*; Combatants
// equals want in that order, has no duplicates and is a fresh copy each call
// (changing a returned slice changes nothing stored). Pass no want for an event
// about no creature. Failures are reported with tb.Errorf.
func CheckEvent(tb testing.TB, e combat.Event, wantKind string, want ...combat.CombatantID) {
	tb.Helper()
	if e == nil {
		tb.Errorf("event is nil")
		return
	}
	if !validKind(wantKind) {
		tb.Errorf("kind %q does not match [a-z][a-z0-9_]*", wantKind)
	}
	if got, again := e.Kind(), e.Kind(); got != wantKind || again != got {
		tb.Errorf("Kind() = %q then %q, want constant %q", got, again, wantKind)
	}
	got := e.Combatants()
	if !slices.Equal(got, want) {
		tb.Errorf("Combatants() = %v, want %v in that order", got, want)
	}
	seen := map[combat.CombatantID]bool{}
	for _, id := range got {
		if seen[id] {
			tb.Errorf("Combatants() lists %q twice", id)
		}
		seen[id] = true
	}
	for i := range got {
		got[i] = "clobbered"
	}
	if again := e.Combatants(); !slices.Equal(again, want) {
		tb.Errorf("Combatants() = %v after the first result was changed, want %v: the slice is not a copy", again, want)
	}
}

// CheckEventKinds checks that the Kind of every event matches
// [a-z][a-z0-9_]* and that no two events share one. Pass only non-envelope
// events: an envelope reports its wrapped event's Kind.
func CheckEventKinds(tb testing.TB, events []combat.Event) {
	tb.Helper()
	seen := map[string]bool{}
	for _, e := range events {
		if e == nil {
			tb.Errorf("event is nil")
			continue
		}
		k := e.Kind()
		if !validKind(k) {
			tb.Errorf("kind %q does not match [a-z][a-z0-9_]*", k)
		}
		if seen[k] {
			tb.Errorf("kind %q is used by two events", k)
		}
		seen[k] = true
	}
}

// CheckEventsCopy checks that a transition result hands out its events as a
// copy, in the order they happened: get returns the stored events each call;
// replacing the elements of a returned slice must leave the next call
// unchanged. Every call must also return the events
// in the same order (the elements compared by value, so two events of one kind in another order fail). Failures are reported with tb.Errorf.
func CheckEventsCopy(tb testing.TB, get func() []combat.Event) {
	tb.Helper()
	ref := get()
	var want []string
	for _, e := range get() {
		want = append(want, kindOrNil(e))
	}
	first := get()
	for i := range first {
		first[i] = nil
	}
	var got []string
	for _, e := range get() {
		got = append(got, kindOrNil(e))
	}
	if !slices.Equal(got, want) {
		tb.Errorf("events = %v after a returned slice was changed, want %v: the slice is not a copy", got, want)
	}
	// Order: a call returns the same events in the same order as the first, so a reordered copy fails even when
	// the reordered events share a kind.
	if again := get(); !reflect.DeepEqual(again, ref) {
		tb.Errorf("events = %v, want %v in the order they happened", again, ref)
	}
}

func kindOrNil(e combat.Event) string {
	if e == nil {
		return "<nil>"
	}
	return e.Kind()
}

func validKind(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, c := range s {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return true
}
