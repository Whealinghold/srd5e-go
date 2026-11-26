package combattest_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
)

// fakeTB records failures instead of failing the test, so the checks can be
// seen to fail.
type fakeTB struct {
	testing.TB
	failures []string
}

func (f *fakeTB) Helper() {}
func (f *fakeTB) Errorf(format string, args ...any) {
	f.failures = append(f.failures, fmt.Sprintf(format, args...))
}

// fakeEvent is a fake family member; Event is embedded only to satisfy the
// unexported marker.
type fakeEvent struct {
	combat.Event
	kind string
	ids  []combat.CombatantID
	// alias makes Combatants return the stored slice instead of a copy.
	alias bool
}

func (e fakeEvent) Kind() string { return e.kind }
func (e fakeEvent) Combatants() []combat.CombatantID {
	if e.alias {
		return e.ids
	}
	return append([]combat.CombatantID(nil), e.ids...)
}

func failed(f func(t testing.TB)) bool {
	tb := &fakeTB{}
	f(tb)
	return len(tb.failures) > 0
}

// Requirement: doc/specs/combat-foundations.md FR-017, FR-018 - the helper passes
// a good event and fails on a wrong or malformed kind, an order that differs from
// the documented one, a duplicate, and a returned slice that aliases stored state.
// Path: per case.
func TestCheckEvent(t *testing.T) {
	t.Parallel()
	a, b := combat.CombatantID("a"), combat.CombatantID("b")
	cases := []struct {
		name, path string
		e          combat.Event
		kind       string
		want       []combat.CombatantID
		fails      bool
	}{
		{"good", "happy", fakeEvent{kind: "damage_taken", ids: []combat.CombatantID{a, b}}, "damage_taken", []combat.CombatantID{a, b}, false},
		{"no creature", "unhappy (boundary)", fakeEvent{kind: "round_started"}, "round_started", nil, false},
		{"nil event", "unhappy (invalid input)", nil, "x", nil, true},
		{"wrong kind", "unhappy (invalid input)", fakeEvent{kind: "other", ids: nil}, "damage_taken", nil, true},
		{"malformed kind", "unhappy (invalid input)", fakeEvent{kind: "DamageTaken"}, "DamageTaken", nil, true},
		{"empty kind", "unhappy (invalid input)", fakeEvent{kind: ""}, "", nil, true},
		{"unordered list", "unhappy (boundary)", fakeEvent{kind: "k", ids: []combat.CombatantID{b, a}}, "k", []combat.CombatantID{a, b}, true},
		{"duplicate", "unhappy (boundary)", fakeEvent{kind: "k", ids: []combat.CombatantID{a, a}}, "k", []combat.CombatantID{a, a}, true},
		{"aliased slice", "unhappy (immutability)", fakeEvent{kind: "k", ids: []combat.CombatantID{a}, alias: true}, "k", []combat.CombatantID{a}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := failed(func(tb testing.TB) { tb.Helper(); combattest.CheckEvent(tb, c.e, c.kind, c.want...) }); got != c.fails {
				t.Errorf("CheckEvent failed = %v, want %v", got, c.fails)
			}
		})
	}
}

// Requirement: doc/specs/combat-foundations.md FR-017 - kinds match the pattern
// and are unique across the family.
// Path: per case.
func TestCheckEventKinds(t *testing.T) {
	t.Parallel()
	ev := func(k string) combat.Event { return fakeEvent{kind: k} }
	cases := []struct {
		name, path string
		events     []combat.Event
		fails      bool
	}{
		{"good", "happy", []combat.Event{ev("a_1"), ev("b")}, false},
		{"empty family", "unhappy (boundary, minimal)", nil, false},
		{"duplicate", "unhappy (name collision)", []combat.Event{ev("a"), ev("a")}, true},
		{"upper case", "unhappy (invalid input)", []combat.Event{ev("A")}, true},
		{"leading digit", "unhappy (invalid input)", []combat.Event{ev("1a")}, true},
		{"space", "unhappy (invalid input)", []combat.Event{ev("a b")}, true},
		{"nil event", "unhappy (invalid input)", []combat.Event{nil}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := failed(func(tb testing.TB) { tb.Helper(); combattest.CheckEventKinds(tb, c.events) }); got != c.fails {
				t.Errorf("CheckEventKinds failed = %v, want %v", got, c.fails)
			}
		})
	}
}

// reordering returns a getter that hands out first for its first two calls (the check reads the events twice before
// it looks again) and then the events in the other order, as a result that sorts on read would.
func reordering(first, later []combat.Event) func() []combat.Event {
	calls := 0
	return func() []combat.Event {
		calls++
		if calls <= 2 {
			return append([]combat.Event(nil), first...)
		}
		return append([]combat.Event(nil), later...)
	}
}

// Requirement: doc/specs/combat-foundations.md FR-021 - a result's events
// accessor returns a copy that keeps the order.
// Path: per case.
func TestCheckEventsCopy(t *testing.T) {
	t.Parallel()
	stored := []combat.Event{fakeEvent{kind: "first"}, fakeEvent{kind: "second"}}
	cases := []struct {
		name, path string
		get        func() []combat.Event
		fails      bool
	}{
		{"copy", "happy", func() []combat.Event { return append([]combat.Event(nil), stored...) }, false},
		{"empty", "unhappy (boundary, minimal)", func() []combat.Event { return nil }, false},
		{"aliased", "unhappy (immutability)", func() []combat.Event { return stored }, true},
		{"reordered after the first call", "unhappy (order)", reordering(stored, []combat.Event{fakeEvent{kind: "second"}, fakeEvent{kind: "first"}}), true},
		{"reordered events of one kind", "unhappy (order)", reordering([]combat.Event{fakeEvent{kind: "same", ids: []combat.CombatantID{"a"}}, fakeEvent{kind: "same", ids: []combat.CombatantID{"b"}}},
			[]combat.Event{fakeEvent{kind: "same", ids: []combat.CombatantID{"b"}}, fakeEvent{kind: "same", ids: []combat.CombatantID{"a"}}}), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			s := append([]combat.Event(nil), stored...) // the aliased case clobbers its own copy
			if c.name == "aliased" {
				c.get = func() []combat.Event { return s }
			}
			if got := failed(func(tb testing.TB) { tb.Helper(); combattest.CheckEventsCopy(tb, c.get) }); got != c.fails {
				t.Errorf("CheckEventsCopy failed = %v, want %v", got, c.fails)
			}
		})
	}
}
