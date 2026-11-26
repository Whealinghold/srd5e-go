package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/combat/combattest"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func sbWindow(tb testing.TB, target, source string, part combat.LimitedUsePartRef, remaining time.Duration) combat.SaveImmunityWindow {
	tb.Helper()
	return luMust(combat.NewSaveImmunityWindow(combat.MustCombatantID(target), combat.MustCombatantID(source), part, remaining))
}

func sbGranted(tb testing.TB, s combat.SaveImmunities, w combat.SaveImmunityWindow) combat.SaveImmunities {
	tb.Helper()
	return s.Grant(w).Immunities()
}

func sbElapsed(tb testing.TB, s combat.SaveImmunities, d time.Duration) combat.SaveImmunityTransition {
	tb.Helper()
	return luMust(s.Elapse(d))
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-039, FR-040 - a granted window makes exactly its target immune to exactly its source's part while time remains;
// another target, source or part is not covered (SRD 5.2.1, "Ghost", "Mummy", pp. 288, 309).
// Path: per case.
func TestSaveImmunitiesIsImmune(t *testing.T) {
	t.Parallel()
	part, otherPart := luRef(t, combat.MonsterPartActions, 0), luRef(t, combat.MonsterPartActions, 1)
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, 24*time.Hour))
	cases := []struct {
		name, requirement, path string
		target, source          string
		part                    combat.LimitedUsePartRef
		want                    bool
	}{
		{"the window", `FR-040`, "happy", "hero", "ghost", part, true},
		{"other source", `FR-039; "this devil's Unnerving Gaze"`, "unhappy (rule interaction)", "hero", "other", part, false},
		{"other part", `FR-039`, "unhappy (rule interaction)", "hero", "ghost", otherPart, false},
		{"other target", `FR-039`, "unhappy (rule interaction)", "villain", "ghost", part, false},
		{"swapped target and source", `FR-039`, "unhappy (rule interaction)", "ghost", "hero", part, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if got := held.IsImmune(combat.MustCombatantID(c.target), combat.MustCombatantID(c.source), c.part); got != c.want {
				t.Errorf("IsImmune = %v, want %v [%s]", got, c.want, c.requirement)
			}
		})
	}
	if (combat.SaveImmunities{}).IsImmune(combat.MustCombatantID("hero"), combat.MustCombatantID("ghost"), part) {
		t.Error("the zero SaveImmunities holds a window")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-041, table SV-1 "23h59", "24h", "negative time" - Elapse reduces every window, drops each that reaches zero or
// below and emits one SaveImmunityExpired per dropped window in granted order; 23 hours 59 minutes after a 24-hour grant it is still immune, 24 hours it is not; zero
// changes nothing; a negative duration is an error (SRD 5.2.1, "Ghost", "Mummy", pp. 288, 309).
// Path: per case.
func TestSaveImmunitiesElapse(t *testing.T) {
	t.Parallel()
	part := luRef(t, combat.MonsterPartActions, 0)
	hero, ghost := combat.MustCombatantID("hero"), combat.MustCombatantID("ghost")
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, 24*time.Hour))
	cases := []struct {
		name, requirement, path string
		elapsed                 time.Duration
		immune                  bool
		expired                 int
	}{
		{"23h59", `FR-041`, "unhappy (boundary)", 23*time.Hour + 59*time.Minute, true, 0},
		{"24h", `FR-041`, "unhappy (boundary)", 24 * time.Hour, false, 1},
		{"24h and a nanosecond", `FR-041`, "unhappy (boundary)", 24*time.Hour + 1, false, 1},
		{"zero", `FR-041`, "unhappy (empty and minimal)", 0, true, 0},
		{"the longest duration", `FR-041; plan section 5 (overflow)`, "unhappy (boundary)", time.Duration(math.MaxInt64), false, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := sbElapsed(t, held, c.elapsed)
			if got := tr.Immunities().IsImmune(hero, ghost, part); got != c.immune || len(tr.Events()) != c.expired {
				t.Errorf("immune %v with %d events, want %v and %d [%s]", got, len(tr.Events()), c.immune, c.expired, c.requirement)
			}
			if c.expired == 1 {
				e, ok := tr.Events()[0].(combat.SaveImmunityExpired)
				if !ok || e.Part() != part {
					t.Fatalf("event = %#v, want SaveImmunityExpired for the part", tr.Events()[0])
				}
				combattest.CheckEvent(t, e, "save_immunity_expired", hero, ghost)
			}
		})
	}
	if tr, err := held.Elapse(-time.Nanosecond); !errors.Is(err, srderr.ErrInvalidInput) || len(tr.Events()) != 0 {
		t.Errorf("Elapse(-1ns) = %v, %v, want an invalid-input error", tr, err)
	}
	// Windows expire in the order they were granted, and the others keep their reduced time.
	s := sbGranted(t, sbGranted(t, sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "a", "ghost", part, 3*time.Hour)), sbWindow(t, "b", "ghost", part, 9*time.Hour)), sbWindow(t, "c", "ghost", part, 3*time.Hour))
	tr := sbElapsed(t, s, 3*time.Hour)
	var order []combat.CombatantID
	for _, e := range tr.Events() {
		order = append(order, e.Combatants()[0])
	}
	kept := tr.Immunities().Windows()
	if !slices.Equal(order, []combat.CombatantID{"a", "c"}) || len(kept) != 1 || kept[0].Combatants()[0] != "b" || kept[0].Remaining() != 6*time.Hour {
		t.Errorf("expired %v, kept %+v, want a then c expired and b with 6 hours", order, kept)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-042, table SV-1 "second grant" - a grant for a triple that already has a window keeps one window with the larger remaining
// time; windows never add; the grant emits one SaveImmunityGranted with the time now held.
// Path: per case.
func TestSaveImmunitiesGrantKeepsTheLargerWindow(t *testing.T) {
	t.Parallel()
	part := luRef(t, combat.MonsterPartActions, 0)
	hero, ghost := combat.MustCombatantID("hero"), combat.MustCombatantID("ghost")
	cases := []struct {
		name, requirement, path string
		first, second, want     time.Duration
	}{
		{"a longer regrant replaces", `FR-042; SV-1 "second grant"`, "unhappy (rule interaction)", 5 * time.Hour, 24 * time.Hour, 24 * time.Hour},
		{"a shorter regrant is kept out", `FR-042`, "unhappy (rule interaction)", 24 * time.Hour, 5 * time.Hour, 24 * time.Hour},
		{"an equal regrant", `FR-042`, "unhappy (boundary)", 5 * time.Hour, 5 * time.Hour, 5 * time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, c.first))
			tr := held.Grant(sbWindow(t, "hero", "ghost", part, c.second))
			windows := tr.Immunities().Windows()
			if len(windows) != 1 || windows[0].Remaining() != c.want {
				t.Errorf("windows = %+v, want one of %v [%s]", windows, c.want, c.requirement)
			}
			e, ok := tr.Events()[0].(combat.SaveImmunityGranted)
			if !ok || len(tr.Events()) != 1 || e.Remaining() != c.want || e.Part() != part {
				t.Fatalf("events = %#v, want one SaveImmunityGranted of %v", tr.Events(), c.want)
			}
			combattest.CheckEvent(t, e, "save_immunity_granted", hero, ghost)
		})
	}
	// A different triple is a second window, and the input is not changed.
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, time.Hour))
	both := held.Grant(sbWindow(t, "hero", "ghost", luRef(t, combat.MonsterPartActions, 1), time.Hour)).Immunities()
	if len(both.Windows()) != 2 || len(held.Windows()) != 1 {
		t.Errorf("windows %d then %d, want a second window beside the first and the first value unchanged", len(both.Windows()), len(held.Windows()))
	}
	// A zero window changes nothing and emits nothing.
	if tr := held.Grant(combat.SaveImmunityWindow{}); len(tr.Events()) != 0 || !tr.Immunities().Equal(held) {
		t.Error("granting the zero window changed the immunities or emitted an event")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-043, table SV-1 "bad window" - a window needs distinct, non-zero target and source, a part and a positive remaining
// time; otherwise the constructor returns an error matching srderr.ErrInvalidInput.
// Path: per case.
func TestNewSaveImmunityWindowRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	part := luRef(t, combat.MonsterPartActions, 0)
	cases := []struct {
		name           string
		target, source combat.CombatantID
		part           combat.LimitedUsePartRef
		remaining      time.Duration
	}{
		{"remaining zero", "hero", "ghost", part, 0},
		{"remaining negative", "hero", "ghost", part, -time.Hour},
		{"target equals source", "ghost", "ghost", part, time.Hour},
		{"zero target", "", "ghost", part, time.Hour},
		{"zero source", "hero", "", part, time.Hour},
		{"zero part", "hero", "ghost", combat.LimitedUsePartRef{}, time.Hour},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			w, err := combat.NewSaveImmunityWindow(c.target, c.source, c.part, c.remaining)
			if !errors.Is(err, srderr.ErrInvalidInput) || !w.IsZero() {
				t.Errorf("NewSaveImmunityWindow = %+v, %v, want the zero window and an invalid-input error", w, err)
			}
		})
	}
	w := sbWindow(t, "hero", "ghost", part, time.Nanosecond)
	if !slices.Equal(w.Combatants(), []combat.CombatantID{"hero", "ghost"}) || w.Part() != part || w.Remaining() != time.Nanosecond || w.IsZero() {
		t.Errorf("window = %+v, want the smallest positive window", w)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-049 - SaveImmunities is an immutable value: Windows returns a copy, an operation leaves the receiver unchanged and the
// events and the immunities of a transition are copies.
// Path: unhappy (immutability).
func TestSaveImmunitiesAreImmutable(t *testing.T) {
	t.Parallel()
	part := luRef(t, combat.MonsterPartActions, 0)
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, time.Hour))
	windows := held.Windows()
	windows[0] = combat.SaveImmunityWindow{}
	if held.Windows()[0].IsZero() {
		t.Error("Windows returned the value's own slice")
	}
	tr := sbElapsed(t, held, 2*time.Hour)
	if len(held.Windows()) != 1 || !held.IsImmune("hero", "ghost", part) {
		t.Error("Elapse changed its receiver")
	}
	events := tr.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	events[0] = nil
	if tr.Events()[0] == nil {
		t.Error("Events returned the transition's own slice")
	}
	grant := held.Grant(sbWindow(t, "villain", "ghost", part, time.Hour))
	inner := grant.Immunities().Windows()
	inner[0] = combat.SaveImmunityWindow{}
	if grant.Immunities().Windows()[0].IsZero() {
		t.Error("Immunities returned the transition's own slice")
	}
	if !held.Equal(sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, time.Hour))) || held.Equal(combat.SaveImmunities{}) || !(combat.SaveImmunities{}).Equal(combat.SaveImmunities{}) {
		t.Error("Equal does not compare the windows")
	}
}

// FuzzNewSaveImmunityWindow checks that no combination of ids, part and duration makes the constructor panic, and that a window it
// returns holds distinct non-zero creatures, a valid part and a positive remaining time.
func FuzzNewSaveImmunityWindow(f *testing.F) {
	f.Add("hero", "ghost", 2, 0, int64(86400000000000))
	f.Add("", "", 0, -1, int64(0))
	f.Add("a", "a", 1, 3, int64(-5))
	f.Fuzz(func(t *testing.T, target, source string, section, index int, remaining int64) {
		part, _ := combat.NewLimitedUsePartRef(combat.MonsterPartSection(section), index)
		w, err := combat.NewSaveImmunityWindow(combat.CombatantID(target), combat.CombatantID(source), part, time.Duration(remaining))
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		if len(w.Combatants()) != 2 || w.Combatants()[0] == "" || w.Combatants()[1] == "" || w.Combatants()[0] == w.Combatants()[1] || w.Part().IsZero() || w.Remaining() <= 0 {
			t.Fatalf("window %+v breaks the invariants", w)
		}
		if after := luMust(combat.SaveImmunities{}.Grant(w).Immunities().Elapse(w.Remaining())); len(after.Immunities().Windows()) != 0 || len(after.Events()) != 1 {
			t.Fatalf("a window does not expire when its remaining time elapses")
		}
	})
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-042 - a window is the same only when target, source and part are all the same: a grant that differs in the target, in the source or in the part
// is a window of its own and nothing is merged.
// Path: per case.
func TestSaveImmunitiesGrantMergesOnlyTheSameTriple(t *testing.T) {
	t.Parallel()
	part, other := luRef(t, combat.MonsterPartActions, 0), luRef(t, combat.MonsterPartActions, 1)
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "hero", "ghost", part, time.Hour))
	for name, w := range map[string]combat.SaveImmunityWindow{
		"another target": sbWindow(t, "villain", "ghost", part, 2*time.Hour), "another source": sbWindow(t, "hero", "wraith", part, 2*time.Hour),
		"another part": sbWindow(t, "hero", "ghost", other, 2*time.Hour)} {
		windows := held.Grant(w).Immunities().Windows()
		if len(windows) != 2 || windows[0].Remaining() != time.Hour || windows[1].Remaining() != 2*time.Hour {
			t.Errorf("%s: windows = %+v, want the hour and the two hours side by side", name, windows)
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-039, FR-040, FR-042 - a window answers only for its own target, source and part whichever way another id compares with it, and a grant merges with
// the window of the same triple wherever it stands among the windows held.
// Path: per case.
func TestSaveImmunitiesIdsCompareByEqualityOnly(t *testing.T) {
	t.Parallel()
	part, after := luRef(t, combat.MonsterPartActions, 1), luRef(t, combat.MonsterPartActions, 2)
	held := sbGranted(t, combat.SaveImmunities{}, sbWindow(t, "mmm", "nnn", part, time.Hour))
	for name, query := range map[string][3]any{"a target before": {"aaa", "nnn", part}, "a source before": {"mmm", "aaa", part}, "a part before": {"mmm", "nnn", luRef(t, combat.MonsterPartActions, 0)},
		"a target after": {"zzz", "nnn", part}, "a source after": {"mmm", "zzz", part}, "a part after": {"mmm", "nnn", after}} {
		if held.IsImmune(combat.MustCombatantID(query[0].(string)), combat.MustCombatantID(query[1].(string)), query[2].(combat.LimitedUsePartRef)) {
			t.Errorf("%s: the window answered for another triple", name)
		}
	}
	for name, w := range map[string]combat.SaveImmunityWindow{"a target before": sbWindow(t, "aaa", "nnn", part, time.Hour), "a source before": sbWindow(t, "mmm", "aaa", part, time.Hour),
		"a part before": sbWindow(t, "mmm", "nnn", luRef(t, combat.MonsterPartActions, 0), time.Hour)} {
		if windows := held.Grant(w).Immunities().Windows(); len(windows) != 2 {
			t.Errorf("%s: %d windows, want 2", name, len(windows))
		}
	}
	two := sbGranted(t, held, sbWindow(t, "aaa", "nnn", part, time.Hour))
	if windows := two.Grant(sbWindow(t, "aaa", "nnn", part, 5*time.Hour)).Immunities().Windows(); len(windows) != 2 || windows[1].Remaining() != 5*time.Hour || windows[0].Remaining() != time.Hour {
		t.Errorf("regranting the second window gave %+v, want it extended in place", windows)
	}
}
