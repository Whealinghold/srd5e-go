package combat

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// bigEncounter starts an encounter of n combatants from inside the package.
func bigEncounter(t *testing.T, n int) Encounter {
	t.Helper()
	standings := make([]InitiativeStanding, n)
	profiles := make([]CombatantProfile, n)
	cs := make([]Combatant, n)
	sp, err := NewMovementSpeeds(30)
	if err != nil {
		t.Fatal(err)
	}
	for i := range n {
		id := CombatantID(fmt.Sprintf("m%04d", i))
		standings[i] = InitiativeStanding{Combatant: id, Count: InitiativeCount(n - i)}
		profiles[i] = CombatantProfile{ID: id, Size: content.SizeMedium, Speeds: sp}
		cs[i], err = NewCombatant(CombatantRef{ID: id, Kind: MonsterCombatant{}}, Side(1+i%2), MustHitPointStatus(10, FallsUnconscious, NoAffinities()), ConditionSet{})
		if err != nil {
			t.Fatal(err)
		}
	}
	cmd, err := NewEstablishInitiativeOrderCommand(standings, profiles, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := NewEstablishInitiativeOrder().Run(t.Context(), cmd)
	if err != nil {
		t.Fatal(err)
	}
	e, _, err := (Encounter{}).Start("enc", res.(InitiativeOrderEstablished), cs)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// Requirement: doc/specs/encounter-aggregate.md FR-009, FR-083, plan-phase-4.md P10 and P14 - the cost of validation is tested with the roster at its cap (256) and the cascade at its step budget (1000): the root methods a cascade commits make no per-Combatant pass, so the work is one pass at entry and one at exit of the call (2 x 256), not 1000 x 256. A package variable counting the checks is barred by the architecture tests, so the test pins the cost by construction: one Combatant forged to fail the part-by-part check sits in a roster of 256, 1000 committed root steps on another combatant all succeed (a pass inside the cascade would find it and refuse every step), and the pass that runs at exit of the call (exitInvalid) does find it.
// Path: unhappy (boundary, cost).
func TestValidationCostIsOnePassAtEntryAndOnePassAtExit(t *testing.T) {
	t.Parallel()
	const n = 256
	e := bigEncounter(t, n)
	e.combatants = slices.Clone(e.combatants)
	e.combatants[n-1].ref.Kind = nil // fails Combatant.wellFormed; not the target of any step
	if !e.IsZero() {
		t.Fatal("the forged roster passes the full check")
	}
	r := &resolution{enc: e}
	for i := range maxStepsPerCall {
		var tr encounterTransition
		var err error
		if i%2 == 0 {
			tr, err = r.enc.ApplyCondition("m0000", Blinded{})
		} else {
			tr, err = r.enc.RemoveCondition("m0000", content.ConditionBlinded)
		}
		if err != nil {
			t.Fatalf("step %d: %v; the cascade made a full pass", i, err)
		}
		r.mustCommit(tr)
	}
	if r.steps != maxStepsPerCall {
		t.Fatalf("steps = %d, want %d", r.steps, maxStepsPerCall)
	}
	if r.passes != 0 {
		t.Fatalf("passes after %d committed steps = %d, want 0: a step made a full pass over the %d combatants", maxStepsPerCall, r.passes, n)
	}
	if !r.exitInvalid() {
		t.Error("the exit pass of the call did not find the forged combatant")
	}
	if r.passes != 1 {
		t.Errorf("passes after the exit check = %d, want exactly 1", r.passes)
	}
	cmd, err := NewApplyConditionToCombatantCommand(e, "m0000", Blinded{})
	if err == nil {
		_, err = NewApplyConditionToCombatant().Run(t.Context(), cmd)
	}
	if !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("use case on the forged roster = %v, want ErrInvalidInput at entry", err)
	}
}
