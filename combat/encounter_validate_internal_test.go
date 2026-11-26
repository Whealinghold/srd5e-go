package combat

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// forgedTracker returns a deep copy of the tracker of a valid encounter after f changed it.
func forgedTracker(t *testing.T, f func(*TurnTracker)) TurnTracker {
	t.Helper()
	tr := forgeable(t).tracker.clone()
	f(&tr)
	return tr
}

// Requirement: doc/specs/encounter-aggregate.md FR-009, plan-phase-4.md P14 - the root validates the tracker: the zero tracker and a forged one (round, active index, nil collections, a duplicated or unnamed order entry, a profile that is missing, wrong or extra, state naming a combatant that is not in the order) are invalid, as an encounter that holds one (IsZero), as the tracker of a start and as the tracker of ApplyTurnChange; the valid tracker is the control.
// Path: per case.
func TestAForgedTrackerIsRejectedByTheRoot(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	if !good.tracker.wellFormed() || good.IsZero() {
		t.Fatal("the control tracker is invalid")
	}
	cases := []struct {
		name, requirement, path string
		forge                   func(*TurnTracker)
	}{
		{"zero tracker", "FR-009 zero tracker", "unhappy (invalid input)", func(tr *TurnTracker) { *tr = TurnTracker{} }},
		{"round 0", "P14 round", "unhappy (state forgery)", func(tr *TurnTracker) { tr.round = 0 }},
		{"negative round", "P14 round", "unhappy (state forgery)", func(tr *TurnTracker) { tr.round = -3 }},
		{"active index past the order", "P14 active", "unhappy (state forgery)", func(tr *TurnTracker) { tr.active = len(tr.order) }},
		{"negative active index", "P14 active", "unhappy (state forgery)", func(tr *TurnTracker) { tr.active = -1 }},
		{"nil profiles", "P14 collections", "unhappy (state forgery)", func(tr *TurnTracker) { tr.profiles = nil }},
		{"nil spent", "P14 collections", "unhappy (state forgery)", func(tr *TurnTracker) { tr.spent = nil }},
		{"nil dodging", "P14 collections", "unhappy (state forgery)", func(tr *TurnTracker) { tr.dodging = nil }},
		{"nil readied", "P14 collections", "unhappy (state forgery)", func(tr *TurnTracker) { tr.readied = nil }},
		{"nil feats", "P14 collections", "unhappy (state forgery)", func(tr *TurnTracker) { tr.feats = nil }},
		{"duplicate order entry", "P14 order", "unhappy (state forgery)", func(tr *TurnTracker) { tr.order[1].Combatant = tr.order[0].Combatant }},
		{"unnamed order entry", "P14 order", "unhappy (state forgery)", func(tr *TurnTracker) { tr.order[1].Combatant = "" }},
		{"missing profile", "P14 profiles", "unhappy (state forgery)", func(tr *TurnTracker) { delete(tr.profiles, "b") }},
		{"profile under the wrong id", "P14 profiles", "unhappy (state forgery)", func(tr *TurnTracker) {
			p := tr.profiles["b"]
			p.ID = "z"
			tr.profiles["b"] = p
		}},
		{"extra profile", "P14 profiles", "unhappy (state forgery)", func(tr *TurnTracker) { tr.profiles["z"] = CombatantProfile{ID: "z"} }},
		{"Reaction spent by a stranger", "P14 state", "unhappy (state forgery)", func(tr *TurnTracker) { tr.spent["z"] = true }},
		{"Dodge of a stranger", "P14 state", "unhappy (state forgery)", func(tr *TurnTracker) { tr.dodging["z"] = true }},
		{"readied action of a stranger", "P14 state", "unhappy (state forgery)", func(tr *TurnTracker) { tr.readied["z"] = ReadiedAction{} }},
		{"feat use of a stranger", "P14 state", "unhappy (state forgery)", func(tr *TurnTracker) { tr.feats[featUseKey{id: "z"}] = true }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			tr := forgedTracker(t, c.forge)
			if tr.wellFormed() {
				t.Fatalf("wellFormed = true [%s]", c.requirement)
			}
			held := good
			held.tracker = tr
			if !held.IsZero() {
				t.Errorf("an encounter holding the forged tracker is valid [%s]", c.requirement)
			}
			if tr, err := good.ApplyTurnChange(tr, nil); tr != nil || !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("ApplyTurnChange = %v, %v, want a nil transition and ErrInvalidInput [%s]", tr, err, c.requirement)
			}
			// A forged tracker passed to Start is refused too, unless its fault is that it is zero (then Start says so first).
			est := InitiativeOrderEstablished{}
			est.tracker = tr
			if _, _, err := (Encounter{}).Start("enc", est, good.combatants); !errors.Is(err, srderr.ErrInvalidInput) {
				t.Errorf("Start = %v, want ErrInvalidInput [%s]", err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-083, plan-phase-4.md P14 - the root validates every Combatant it holds, part by part: Hit Points outside their bounds, a life state that disagrees with the Hit Points, a condition stored under another kind or invalid, and an immunity bit for the unset kind 0 make the combatant, an encounter holding it and a start with it invalid; consistency between life state, markers and conditions is still the host's (the control is a Dying combatant without Unconscious).
// Path: per case.
func TestAForgedCombatantIsRejectedByTheRoot(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	dying := MustHitPointStatus(10, FallsUnconscious, NoAffinities())
	dying.hp.current, dying.life = 0, Dying{}
	cases := []struct {
		name, requirement string
		path              string
		forge             func(*Combatant)
		wantValid         bool
	}{
		{"control: a Dying combatant without Unconscious", "FR-083 not checked", "happy", func(c *Combatant) { c.hp = dying }, true},
		{"current above the maximum", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.hp.current = 11 }, false},
		{"negative current", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.hp.current = -1 }, false},
		{"negative temporary", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.hp.temporary = -1 }, false},
		{"Dying with Hit Points", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.life = Dying{} }, false},
		{"Alive at 0 Hit Points", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.hp.current = 0 }, false},
		{"no zero Hit Points rule", "P14 hit points", "unhappy (state forgery)", func(c *Combatant) { c.hp.rule = 0 }, false},
		{"a condition under another kind", "P14 conditions", "unhappy (state forgery)", func(c *Combatant) { c.conditions.slots[content.ConditionDeafened] = Blinded{} }, false},
		{"an invalid condition", "P14 conditions", "unhappy (state forgery)", func(c *Combatant) { c.conditions.slots[content.ConditionCharmed] = Charmed{} }, false},
		{"an immunity bit for the unset kind 0", "P14 conditions", "unhappy (state forgery)", func(c *Combatant) { c.conditions.immune = 1 << 0 }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			cb := good.combatants[1]
			c.forge(&cb)
			if got := cb.wellFormed(); got != c.wantValid {
				t.Fatalf("wellFormed = %v, want %v [%s]", got, c.wantValid, c.requirement)
			}
			held := good
			held.combatants = []Combatant{good.combatants[0], cb}
			if held.IsZero() == c.wantValid {
				t.Errorf("an encounter holding it: IsZero = %v [%s]", held.IsZero(), c.requirement)
			}
			enc, _, err := (Encounter{}).Start("enc", InitiativeOrderEstablished{appliedTurnStep: appliedTurnStep{}, ties: nil}.withTracker(good.tracker), held.combatants)
			if c.wantValid != (err == nil) || (err != nil && (!errors.Is(err, srderr.ErrInvalidInput) || !enc.Equal(Encounter{}))) {
				t.Errorf("Start = %v, want valid %v and otherwise ErrInvalidInput with the zero encounter [%s]", err, c.wantValid, c.requirement)
			}
		})
	}
}

// withTracker returns the established order holding tr, for a test that starts from a tracker it built.
func (o InitiativeOrderEstablished) withTracker(tr TurnTracker) InitiativeOrderEstablished {
	o.tracker = tr
	return o
}

// Requirement: doc/specs/encounter-aggregate.md FR-012, plan-phase-4.md P14 - a duplicate CombatantID is reported by Start as DuplicateCombatantIDProblem alone (the tracker and the other combatant are well formed), the encounter is the zero Encounter, and the error matches ErrInvalidInput; a registry-forged duplicate in a held encounter is invalid.
// Path: unhappy (invalid input).
func TestADuplicateCombatantIDIsRejected(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	a := good.combatants[0]
	est := InitiativeOrderEstablished{}.withTracker(good.tracker)
	enc, _, err := (Encounter{}).Start("enc", est, []Combatant{a, good.combatants[1], a})
	want := []EncounterProblem{DuplicateCombatantIDProblem{ID: "a"}}
	if got := ProblemsOf(err); !errors.Is(err, srderr.ErrInvalidInput) || !slices.Equal(got, want) || !enc.Equal(Encounter{}) {
		t.Errorf("Start = %v with problems %v, want %v and the zero encounter", err, got, want)
	}
	held := good
	held.combatants = []Combatant{a, a}
	if !held.IsZero() {
		t.Error("a held encounter with a duplicated id is valid")
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-009, FR-083, plan-phase-4.md P14 - the root validates at the exit of a call: a transition that produces an encounter that is not valid is an ErrInvalidInput error with no signal counted and no result, not an applied result; a valid one and a refusal pass.
// Path: per case.
func TestRunTransitionValidatesTheEncounterAtExit(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	forged := good
	forged.combatants = []Combatant{good.combatants[0], {}}
	cases := []struct {
		name, requirement, path string
		work                    func() (encounterTransition, error)
		wantErr, wantApplied    bool
	}{
		{"a valid result", "P14 exit", "happy", func() (encounterTransition, error) { return good.accept(nil), nil }, false, true},
		{"a refusal of the encounter", "P14 exit", "unhappy (state error)", func() (encounterTransition, error) { return good.reject(UnknownCombatantRejection{ID: "z"}), nil }, false, false},
		{"an invalid encounter produced", "P14 exit", "unhappy (state forgery)", func() (encounterTransition, error) { return forged.accept(nil), nil }, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec := telemetrytest.NewRecorder()
			use := newEncounterUse([]Option{WithTelemetry(rec.Telemetry())})
			res, err := runTransition(t.Context(), use, transitionOp("end_encounter", "span"), good, c.work,
				func(encounterApplied, []DamageResolution) bool { return true }, func(encounterRejected) bool { return false })
			if c.wantErr != errors.Is(err, srderr.ErrInvalidInput) || (err == nil && res != c.wantApplied) {
				t.Errorf("= %v, %v, want error %v applied %v [%s]", res, err, c.wantErr, c.wantApplied, c.requirement)
			}
			if got := len(rec.Meter.MeasurementsNamed(metricOperations)); c.wantErr && got != 0 {
				t.Errorf("%d operations counted for an invalid exit, want none [%s]", got, c.requirement)
			}
		})
	}
}
