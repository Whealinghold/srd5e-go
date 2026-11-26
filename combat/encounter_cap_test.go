package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/encounter-aggregate.md plan P10 (combatants per encounter 256) - Start accepts 256 combatants and rejects 257 with the typed CombatantLimitProblem alone, the zero Encounter and an ErrInvalidInput error; the other problems are not reported then.
// Path: per case.
func TestStartCombatantCap(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		n                       int
		ok                      bool
	}{
		{"one below the cap", "P10 below the cap", "unhappy (boundary)", 255, true},
		{"at the cap", "P10 at the cap", "unhappy (boundary)", 256, true},
		{"cap plus one", "P10 cap plus one", "unhappy (boundary)", 257, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			est, cs := rosBigParts(t, c.n, func(int) (string, bool) { return "", false })
			e, _, err := (combat.Encounter{}).Start(eid(), est, cs)
			if c.ok {
				if err != nil || len(e.Combatants()) != c.n {
					t.Fatalf("Start = %d combatants, %v, want %d [%s]", len(e.Combatants()), err, c.n, c.requirement)
				}
				return
			}
			if !errors.Is(err, srderr.ErrInvalidInput) || !e.Equal(combat.Encounter{}) || !slices.Equal(combat.ProblemsOf(err), []combat.EncounterProblem{combat.CombatantLimitProblem{}}) {
				t.Errorf("Start = %v, %v, want the zero encounter and CombatantLimitProblem [%s]", e, err, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/encounter-aggregate.md plan P10, section 7 - StartEncounter reports 257 combatants as class invalid with the shared result value limit_exceeded and counts no encounter size; 256 is applied.
// Path: per case.
func TestStartEncounterCapSignal(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name, requirement string
		n                 int
		class, result     string
	}{{"at the cap", "P10 at the cap", 256, "applied", "applied"}, {"cap plus one", "P10 cap plus one", 257, "invalid", "limit_exceeded"}} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			est, cs := rosBigParts(t, c.n, func(int) (string, bool) { return "", false })
			rec, opts := recording()
			res, err := combat.NewStartEncounter(opts...).Run(t.Context(), combat.NewStartEncounterCommand(eid(), est, cs))
			if err != nil {
				t.Fatal(err)
			}
			_, applied := res.(combat.StartEncounterApplied)
			ops := rec.Meter.MeasurementsNamed("srd5e.combat.operations")
			if applied != (c.class == "applied") || len(ops) != 1 {
				t.Fatalf("result %#v, %d operations [%s]", res, len(ops), c.requirement)
			}
			got := attrMap(ops[0].Attrs)
			if got["srd5e.combat.result_class"] != c.class || got["srd5e.combat.result"] != c.result {
				t.Errorf("attributes = %v, want class %s result %s [%s]", got, c.class, c.result, c.requirement)
			}
		})
	}
}
