package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// rosApplied returns a function that takes the two results of a roster operation and returns the tracker and standing of an applied step, failing the test otherwise.
func rosApplied(tb testing.TB) func(combat.RosterStep, error) (combat.TurnTracker, combat.InitiativeStanding) {
	tb.Helper()
	return func(s combat.RosterStep, err error) (combat.TurnTracker, combat.InitiativeStanding) {
		tb.Helper()
		if err != nil {
			tb.Fatalf("operation error: %v", err)
		}
		a, ok := s.(combat.RosterStepApplied)
		if !ok {
			tb.Fatalf("step = %#v, want RosterStepApplied", s)
		}
		return a.Tracker(), a.Standing()
	}
}

// rosRefused returns a function that takes the two results of a roster operation and returns the reason of a rejected step, failing the test otherwise.
func rosRefused(tb testing.TB) func(combat.RosterStep, error) combat.RosterRejection {
	tb.Helper()
	return func(s combat.RosterStep, err error) combat.RosterRejection {
		tb.Helper()
		if err != nil {
			tb.Fatalf("operation error: %v", err)
		}
		r, ok := s.(combat.RosterStepRejected)
		if !ok {
			tb.Fatalf("step = %#v, want RosterStepRejected", s)
		}
		return r.Reason
	}
}

// rosBusy is the tracker of rosBusyEnc.
func rosBusy(tb testing.TB) combat.TurnTracker {
	tb.Helper()
	return rosBusyEnc(tb).Tracker()
}

// Requirement: doc/specs/combatant-roster-changes.md FR-005 - an insert, a removal and a move change neither the round, the active combatant, its resources, nor any other entry's Reaction, Dodge, readied action, speeds or feat uses.
// Path: unhappy (rule interaction).
func TestRosterOperationsLeaveEveryOtherEntryAlone(t *testing.T) {
	t.Parallel()
	base := rosBusy(t)
	ops := map[string]func() (combat.RosterStep, error){
		"insert after the active combatant": func() (combat.RosterStep, error) { return base.InsertCombatant(rosProfile(t, "x"), rosAfter("d")) },
		"insert before the active combatant": func() (combat.RosterStep, error) {
			return base.InsertCombatant(rosProfile(t, "x"), rosAt(99, rosBefore))
		},
		"remove a bystander":  func() (combat.RosterStep, error) { return base.RemoveCombatant(cid("a")) },
		"move a bystander":    func() (combat.RosterStep, error) { return base.MoveCombatant(cid("a"), rosAt(0, rosBehind)) },
		"move to a tie group": func() (combat.RosterStep, error) { return base.MoveCombatant(cid("b"), rosAt(5, rosBefore)) },
	}
	for name, op := range ops {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got, standing := rosApplied(t)(op())
			if got.Round() != base.Round() || got.Active().Combatant != base.Active().Combatant || got.Resources() != base.Resources() {
				t.Errorf("round, active combatant or resources changed [FR-005]")
			}
			for _, s := range base.Order().Standings() {
				if s.Combatant == standing.Combatant {
					continue
				}
				pb, okb := base.ProfileOf(s.Combatant)
				pa, oka := got.ProfileOf(s.Combatant)
				ra, rok := base.ReadiedActionOf(s.Combatant)
				rb, rok2 := got.ReadiedActionOf(s.Combatant)
				if okb != oka || pa != pb || got.ReactionSpent(s.Combatant) != base.ReactionSpent(s.Combatant) || got.IsDodging(s.Combatant) != base.IsDodging(s.Combatant) ||
					rok != rok2 || (rok && !ra.Equal(rb)) {
					t.Errorf("the state of %v changed [FR-005]", s.Combatant)
				}
			}
		})
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-006, FR-007 - a joiner placed before the active combatant takes no turn until the round wraps, one placed after it takes its turn when the tracker reaches it; a roster operation produces no turn event; a joiner starts with its Reaction available, not Dodging, no readied action and the speeds of its profile.
// Path: unhappy (rule interaction).
func TestJoinerTakesItsTurnByItsPlaceAndStartsFresh(t *testing.T) {
	t.Parallel()
	started := func(tr combat.TurnTracker) []string {
		var ids []string
		for range 4 {
			step, err := tr.Advance(combat.TurnAdvance{})
			var evs []combat.TurnEvent
			tr, evs = trkApplied(t)(step, err)
			for _, e := range evs {
				if s, ok := e.(combat.TrackerTurnStarted); ok {
					ids = append(ids, string(s.Combatants()[0]))
				}
			}
		}
		return ids
	}
	base := rosEnc(t).Tracker() // b is active in round 2; the order is a, b, c, d
	after, _ := rosApplied(t)(base.InsertCombatant(rosProfile(t, "x"), rosAfter("b")))
	if got, want := started(after), []string{"x", "c", "d", "a"}; !slices.Equal(got, want) {
		t.Errorf("after b the turns start %v, want %v: x acts this round [FR-006]", got, want)
	}
	before, _ := rosApplied(t)(base.InsertCombatant(rosProfile(t, "x"), rosAt(20, rosBehind)))
	if got, want := started(before), []string{"c", "d", "x", "a"}; !slices.Equal(got, want) {
		t.Errorf("before the active combatant the turns start %v, want %v: x waits for the wrap [FR-006]", got, want)
	}
	fast, err := combat.NewMovementSpeeds(45)
	if err != nil {
		t.Fatal(err)
	}
	profile := combat.CombatantProfile{ID: cid("x"), Size: content.SizeLarge, Speeds: fast}
	joined, _ := rosApplied(t)(base.InsertCombatant(profile, rosAfter("a")))
	if p, ok := joined.ProfileOf(cid("x")); !ok || p != profile {
		t.Errorf("profile of x = %+v (%v), want the one supplied [FR-007]", p, ok)
	}
	if _, readied := joined.ReadiedActionOf(cid("x")); joined.ReactionSpent(cid("x")) || joined.IsDodging(cid("x")) || readied {
		t.Errorf("a joiner must start with its Reaction available, not Dodging and no readied action [FR-007]")
	}
	if joined.Resources() != base.Resources() {
		t.Errorf("the joiner changed the active turn's resources [FR-007]")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-039, FR-040 - the tracker operations are silent (a step carries no TurnEvent), leave the receiver unchanged, refuse an id that is not there or already is with the two tracker rejections, and do not refuse the active combatant.
// Path: unhappy (state).
func TestTrackerRosterRefusalsAndImmutability(t *testing.T) {
	t.Parallel()
	base := rosEnc(t).Tracker()
	copyOf := rosEnc(t).Tracker()
	if r := rosRefused(t)(base.InsertCombatant(rosProfile(t, "c"), rosAfter("a"))); r != (combat.RosterDuplicateCombatantRejection{ID: "c"}) {
		t.Errorf("duplicate insert = %#v, want RosterDuplicateCombatantRejection{c}", r)
	}
	if r := rosRefused(t)(base.InsertCombatant(rosProfile(t, "x"), rosAfter("z"))); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
		t.Errorf("insert after an unknown anchor = %#v, want RosterUnknownCombatantRejection{z}", r)
	}
	if r := rosRefused(t)(base.InsertCombatant(rosProfile(t, "x"), rosAfter("x"))); r != (combat.RosterUnknownCombatantRejection{ID: "x"}) {
		t.Errorf("insert after itself = %#v, want RosterUnknownCombatantRejection{x}: the joiner is not yet in the tracker", r)
	}
	if r := rosRefused(t)(base.RemoveCombatant(cid("z"))); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
		t.Errorf("remove unknown = %#v", r)
	}
	if r := rosRefused(t)(base.MoveCombatant(cid("z"), rosAfter("a"))); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
		t.Errorf("move unknown subject = %#v", r)
	}
	if r := rosRefused(t)(base.MoveCombatant(cid("c"), rosAfter("z"))); r != (combat.RosterUnknownCombatantRejection{ID: "z"}) {
		t.Errorf("move after an unknown anchor = %#v", r)
	}
	if _, err := base.MoveCombatant(cid("c"), rosAfter("c")); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("move after itself: err = %v, want ErrInvalidInput", err)
	}
	// The active combatant is not refused at tracker level (FR-040): it can be removed and moved.
	if got, _ := rosApplied(t)(base.RemoveCombatant(cid("b"))); got.Active().Combatant != cid("c") {
		t.Errorf("after removing the active entry the active combatant is %v, want the next, c", got.Active().Combatant)
	}
	if got, standing := rosApplied(t)(base.MoveCombatant(cid("b"), rosAt(1, rosBehind))); got.Active().Combatant != cid("b") || standing.Count != 1 {
		t.Errorf("a moved active combatant stays active: %v, standing %+v", got.Active().Combatant, standing)
	}
	if !base.Equal(copyOf) {
		t.Errorf("an operation changed its receiver [FR-039]")
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-039 - an operation on the zero tracker, with a blank id or a profile without a size is invalid input, never a panic.
// Path: unhappy (invalid input).
func TestTrackerRosterInvalidInput(t *testing.T) {
	t.Parallel()
	var zero combat.TurnTracker
	base := rosEnc(t).Tracker()
	noSize := combat.CombatantProfile{ID: cid("x"), Speeds: trkSpeeds(t, 30)}
	checks := map[string]func() error{
		"insert on the zero tracker": func() error { _, err := zero.InsertCombatant(rosProfile(t, "x"), rosAfter("a")); return err },
		"insert a blank id": func() error {
			_, err := base.InsertCombatant(combat.CombatantProfile{Size: content.SizeMedium}, rosAfter("a"))
			return err
		},
		"insert without a size":      func() error { _, err := base.InsertCombatant(noSize, rosAfter("a")); return err },
		"remove on the zero tracker": func() error { _, err := zero.RemoveCombatant(cid("a")); return err },
		"remove a blank id":          func() error { _, err := base.RemoveCombatant(""); return err },
		"move on the zero tracker":   func() error { _, err := zero.MoveCombatant(cid("a"), rosAfter("b")); return err },
		"move a blank id":            func() error { _, err := base.MoveCombatant("", rosAfter("a")); return err },
	}
	for name, f := range checks {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/combatant-roster-changes.md FR-016, FR-017, FR-039 - the index of the active combatant follows a removal (the same combatant stays active after a removal before or after it; the next entry becomes active when the active one is removed, the first when it was last); a removal drops the leaver's profile, spent Reaction, Dodge, readied action and feat uses, so a later insert starts fresh; the last entry leaves the zero tracker.
// Path: unhappy (boundary).
func TestRemovalAdjustsTheActiveIndexAndDropsTheLeaversState(t *testing.T) {
	t.Parallel()
	base := rosEnc(t).Tracker() // a, b, c, d; b is active
	cases := []struct {
		name, remove, active string
	}{
		{"before the active combatant", "a", "b"},
		{"after the active combatant", "d", "b"},
		{"the active combatant", "b", "c"},
	}
	for _, c := range cases {
		got, standing := rosApplied(t)(base.RemoveCombatant(cid(c.remove)))
		if got.Active().Combatant != cid(c.active) || got.Round() != 2 || standing.Combatant != cid(c.remove) {
			t.Errorf("%s: active %v, round %d, standing %+v, want active %s in round 2", c.name, got.Active().Combatant, got.Round(), standing, c.active)
		}
		if _, ok := got.ProfileOf(cid(c.remove)); ok {
			t.Errorf("%s: the profile of the leaver stays", c.name)
		}
	}
	// The last entry active: removing it wraps the active index to the first entry.
	last := rosEnc(t)
	last, _ = advance(t, last)
	last, _ = advance(t, last) // d is active
	if got, _ := rosApplied(t)(last.Tracker().RemoveCombatant(cid("d"))); got.Active().Combatant != cid("a") {
		t.Errorf("removing the last active entry leaves %v active, want the first, a", got.Active().Combatant)
	}
	busy := rosBusy(t)
	gone, _ := rosApplied(t)(busy.RemoveCombatant(cid("c")))
	back, _ := rosApplied(t)(gone.InsertCombatant(rosProfile(t, "c"), rosAt(10, rosBefore)))
	_, readied := back.ReadiedActionOf(cid("c"))
	if !busy.ReactionSpent(cid("c")) || !busy.IsDodging(cid("c")) {
		t.Fatalf("fixture: c holds no spent Reaction or Dodge")
	}
	if back.ReactionSpent(cid("c")) || back.IsDodging(cid("c")) || readied {
		t.Errorf("c came back with its old Reaction, Dodge or readied action [FR-017]")
	}
	if _, readiedD := busy.ReadiedActionOf(cid("d")); !readiedD {
		t.Fatalf("fixture: d holds no readied action")
	}
	if gone2, _ := rosApplied(t)(busy.RemoveCombatant(cid("d"))); func() bool { _, ok := gone2.ReadiedActionOf(cid("d")); return ok }() {
		t.Errorf("the readied action of the leaver stays [FR-016]")
	}
	if r := trkReason(t)(busy.RecordFeatUse(cid("c"), "f", character.OnceUntilStartOfYourNextTurn{})); r != (combat.TurnFeatUseLimitedRejection{Feat: "f"}) {
		t.Fatalf("fixture: the feat use of c is not recorded: %#v", r)
	}
	again, _ := rosApplied(t)(gone.InsertCombatant(rosProfile(t, "c"), rosAt(10, rosBefore)))
	trkApplied(t)(again.RecordFeatUse(cid("c"), "f", character.OnceUntilStartOfYourNextTurn{})) // fails the test if c came back holding its old feat use
	solo := trkTracker(t, trkProfile(t, trkA, content.SizeMedium, trkSpeeds(t, 30)))
	if got, _ := rosApplied(t)(solo.RemoveCombatant(trkA)); !got.IsZero() {
		t.Errorf("removing the only entry leaves a tracker that is not the zero tracker")
	}
}
