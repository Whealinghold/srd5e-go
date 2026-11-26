package combat

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// forgeable builds a valid started encounter from inside the package so its fields can be forged.
func forgeable(t *testing.T) Encounter {
	t.Helper()
	ids := []CombatantID{"a", "b"}
	var standings []InitiativeStanding
	var profiles []CombatantProfile
	for i, id := range ids {
		standings = append(standings, InitiativeStanding{Combatant: id, Count: InitiativeCount(10 - i)})
		sp, err := NewMovementSpeeds(30)
		if err != nil {
			t.Fatal(err)
		}
		profiles = append(profiles, CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: sp})
	}
	cmd, err := NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := NewEstablishInitiativeOrder().Run(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	mk := func(id CombatantID, side Side) Combatant {
		c, err := NewCombatant(CombatantRef{ID: id, Kind: MonsterCombatant{}}, side, MustHitPointStatus(10, FallsUnconscious, NoAffinities()), ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	e, _, err := (Encounter{}).Start("enc", res.(InitiativeOrderEstablished), []Combatant{mk("a", SideA), mk("b", SideB)})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Requirement: doc/specs/encounter-aggregate.md FR-082, section 5 (invariants) and plan-phase-4.md section 7 - an Encounter forged from the zero value or with a field changed to break an invariant is invalid (IsZero), and every transition on it is an ErrInvalidInput error: a forged value cannot skip the invariants. The valid one is the control.
// Path: unhappy (invalid input, state forgery).
func TestAForgedEncounterIsInvalidAndEveryCommandRefusesIt(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	if good.IsZero() {
		t.Fatal("the control encounter is invalid")
	}
	forge := map[string]func(e *Encounter){
		"zero value":            func(e *Encounter) { *e = Encounter{} },
		"no id":                 func(e *Encounter) { e.id = "" },
		"nil status":            func(e *Encounter) { e.status = nil },
		"ended with no reason":  func(e *Encounter) { e.status = EncounterEndedStatus{} },
		"zero tracker":          func(e *Encounter) { e.tracker = TurnTracker{} },
		"no combatants":         func(e *Encounter) { e.combatants = nil },
		"one combatant missing": func(e *Encounter) { e.combatants = e.combatants[:1] },
		"order reversed":        func(e *Encounter) { e.combatants = []Combatant{e.combatants[1], e.combatants[0]} },
		"duplicate id":          func(e *Encounter) { e.combatants = []Combatant{e.combatants[0], e.combatants[0]} },
		"zero combatant":        func(e *Encounter) { e.combatants = []Combatant{e.combatants[0], {}} },
		"kindless combatant": func(e *Encounter) {
			c := e.combatants[1]
			c.ref.Kind = nil
			e.combatants = []Combatant{e.combatants[0], c}
		},
		"combatant without a side": func(e *Encounter) {
			c := e.combatants[1]
			c.side = 0
			e.combatants = []Combatant{e.combatants[0], c}
		},
		"combatant without hit points": func(e *Encounter) {
			c := e.combatants[1]
			c.hp = HitPointStatus{}
			e.combatants = []Combatant{e.combatants[0], c}
		},
	}
	deepOnly := map[string]bool{"order reversed": true, "duplicate id": true, "zero combatant": true,
		"kindless combatant": true, "combatant without a side": true, "combatant without hit points": true}
	for name, f := range forge {
		e := good
		e.combatants = slices.Clone(good.combatants)
		f(&e)
		if !e.IsZero() {
			t.Errorf("%s: IsZero = false, want a forged encounter to be invalid", name)
		}
		// The root methods run inside a cascade and make the O(1) check (usable); the part-by-part check of the
		// order, the ids and each Combatant is IsZero, made at entry and exit of the call (plan P14, TestValidationCost...).
		if deepOnly[name] {
			continue
		}
		for cmd, run := range map[string]func() (encounterTransition, error){
			"AdvanceTurn":     func() (encounterTransition, error) { return e.AdvanceTurn(TurnAdvance{}) },
			"ApplyTurnChange": func() (encounterTransition, error) { return e.ApplyTurnChange(good.tracker, nil) },
			"ApplyDamage": func() (encounterTransition, error) {
				return e.ApplyDamage("a", []Damage{MustDamage(1, content.DamageFire)}, nil, OtherSource, NormalHit)
			},
			"ApplyHealing":     func() (encounterTransition, error) { return e.ApplyHealing("a", 1) },
			"ApplyCondition":   func() (encounterTransition, error) { return e.ApplyCondition("a", Blinded{}) },
			"RemoveCondition":  func() (encounterTransition, error) { return e.RemoveCondition("a", content.ConditionBlinded) },
			"ReduceExhaustion": func() (encounterTransition, error) { return e.ReduceExhaustion("a", 1) },
			"DeathSavingThrow": func() (encounterTransition, error) { return e.ApplyDeathSavingThrow("a", 10, 10) },
			"Stabilize":        func() (encounterTransition, error) { return e.Stabilize("a") },
			"RecoverStable":    func() (encounterTransition, error) { return e.RecoverStable("a") },
			"GainTemporary":    func() (encounterTransition, error) { return e.GainTemporaryHitPoints("a", 1, AcceptNew) },
			"ReduceMaximum":    func() (encounterTransition, error) { return e.ReduceHitPointMaximum("a", 1) },
			"CheckGrapple":     func() (encounterTransition, error) { return e.CheckGrapple("a", 1, 1) },
			"ReleaseGrapple":   func() (encounterTransition, error) { return e.ReleaseGrapple("a") },
			"EscapeGrapple":    func() (encounterTransition, error) { return e.EscapeGrapple("a", d20.Success) },
			"end":              func() (encounterTransition, error) { return e.end(EndReasonMutualAgreement{}) },
		} {
			if tr, err := run(); !errors.Is(err, srderr.ErrInvalidInput) || tr != nil {
				t.Errorf("%s on %q: = %v, %v, want a nil transition and ErrInvalidInput", cmd, name, tr, err)
			}
		}
		if r := e.PrecheckDeathSavingThrow("a"); r != (UnknownCombatantRejection{ID: "a"}) {
			t.Errorf("%s: precheck = %#v, want UnknownCombatant", name, r)
		}
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-064 - a combatant that is not Dead yet holds Exhaustion level 6 (a state the root never produces) gets CombatantIsDead for another Exhaustion, since ExhaustionAtMaximum has no rejection of its own.
// Path: unhappy (boundary).
func TestExhaustionAtMaximumHasNoRejectionOfItsOwn(t *testing.T) {
	t.Parallel()
	e := forgeable(t)
	set, err := NewConditionSet([]Condition{ExhaustionCondition{Level: MustExhaustionLevel(6)}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	e.combatants = slices.Clone(e.combatants)
	e.combatants[0].conditions = set
	tr, err := e.ApplyCondition("a", ExhaustionCondition{Level: MustExhaustionLevel(1)})
	if err != nil || tr.Rejection() != (CombatantIsDeadRejection{ID: "a"}) {
		t.Errorf("ApplyCondition = %v, %v, want CombatantIsDead", tr, err)
	}
}

// Requirement: doc/specs/encounter-aggregate.md FR-049, FR-052 - a transition leaves the receiver's combatant slice untouched: replace works on a copy, so an alias held by the host cannot see a transition.
// Path: unhappy (immutability).
func TestReplaceCopiesTheCombatants(t *testing.T) {
	t.Parallel()
	e := forgeable(t)
	c := e.combatants[0]
	c.knockedOut = true
	n := e.replace(0, c)
	if e.combatants[0].knockedOut || !n.combatants[0].knockedOut {
		t.Errorf("replace changed the receiver's slice")
	}
	if &n.combatants[0] == &e.combatants[0] {
		t.Errorf("replace shares the backing array")
	}
}

// Requirement: doc/specs/encounter-aggregate.md section 5 (invariants) - more combatants than the tracker holds, and a combatant of the right count whose id is below or above its place in the order, make the encounter invalid (without an out-of-range read).
// Path: unhappy (state forgery).
func TestForgedCombatantCountsAndIDsAreInvalid(t *testing.T) {
	t.Parallel()
	good := forgeable(t)
	extra := good
	extra.combatants = append(slices.Clone(good.combatants), good.combatants[0])
	if !extra.IsZero() {
		t.Errorf("an extra combatant must make the encounter invalid")
	}
	for _, id := range []CombatantID{"0", "z"} {
		e := good
		e.combatants = slices.Clone(good.combatants)
		e.combatants[0].ref.ID = id
		if !e.IsZero() {
			t.Errorf("a combatant %q in the place of a in the order must make the encounter invalid", id)
		}
	}
}
