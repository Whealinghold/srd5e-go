package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
)

// Requirement: doc/specs/combat-action-resolution.md FR-078, FR-158, FR-132 - ConsequenceRules.Concentration decides, from the combatant after the damage, the input and the DamageTaken total, in this order: no tracking, no Concentration in progress or no damage is no check; a dead combatant ends it (dead); an Incapacitated one ends it (incapacitated, which Unconscious implies); damage that cannot break it is no check; else the save is due; a total of 1, which Temporary Hit Points can absorb, is already due.
// Path: per case.
func TestConsequenceRulesConcentration(t *testing.T) {
	t.Parallel()
	living := encC(t, "b", combat.SideB, pcStatus(t, 20, 20, 0))
	stunned := encC(t, "b", combat.SideB, pcStatus(t, 20, 20, 0), combat.Stunned{})
	unconscious := encC(t, "b", combat.SideB, dyingStatus(t, 20, 0, 0), combat.Unconscious{})
	dead := resDead(t, "b", combat.SideB)
	may, cannot := concTracks(t, true, 0, 0), concTracks(t, true, 0, combat.DamageCannotBreakConcentration)
	cases := []struct {
		name, requirement, path string
		c                       combat.Combatant
		in                      combat.ConcentrationInput
		taken                   int
		step                    combat.ConcentrationStep
		reason                  character.ConcentrationBreakReason
	}{
		{"no tracking", resSpec + "FR-078 step 4", "happy", living, combat.NoConcentrationTracking{}, 9, combat.ConcentrationStepNone, 0},
		{"nil input", resSpec + "FR-078 step 4", "unhappy (empty and minimal)", living, nil, 9, combat.ConcentrationStepNone, 0},
		{"none in progress", resSpec + "FR-078 step 4", "happy", living, concTracks(t, false, 0, 0), 9, combat.ConcentrationStepNone, 0},
		{"no damage taken", resSpec + "FR-081", "unhappy (boundary)", living, may, 0, combat.ConcentrationStepNone, 0},
		{"one damage taken", resSpec + "FR-132", "unhappy (boundary)", living, may, 1, combat.ConcentrationStepSave, 0},
		{"conscious", resSpec + "FR-078 step 3", "happy", living, may, 9, combat.ConcentrationStepSave, 0},
		{"dead", resSpec + "FR-078 step 1", "unhappy (rule interaction)", dead, may, 9, combat.ConcentrationStepBreak, character.BreakDead},
		{"dead beats Incapacitated", resSpec + "FR-078 step 1", "unhappy (rule interaction)", dead, cannot, 9, combat.ConcentrationStepBreak, character.BreakDead},
		{"Incapacitated by a condition", resSpec + "FR-078 step 2", "unhappy (rule interaction)", stunned, may, 9, combat.ConcentrationStepBreak, character.BreakIncapacitated},
		{"Unconscious", resSpec + "FR-078 step 2", "unhappy (rule interaction)", unconscious, may, 9, combat.ConcentrationStepBreak, character.BreakIncapacitated},
		{"damage cannot break, conscious", resSpec + "FR-158", "unhappy (rule interaction)", living, cannot, 9, combat.ConcentrationStepNone, 0},
		{"damage cannot break, Incapacitated", resSpec + "FR-158", "unhappy (rule interaction)", unconscious, cannot, 9, combat.ConcentrationStepBreak, character.BreakIncapacitated},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			d := (combat.ConsequenceRules{}).Concentration(c.c, c.in, c.taken)
			if d.Step() != c.step || d.Reason() != c.reason {
				t.Errorf("Concentration = step %v reason %v, want step %v reason %v [%s]", d.Step(), d.Reason(), c.step, c.reason, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-158, FR-094 - the damage rule is a closed pair with a name, the zero rule reads as the default, and the outcome and event types expose copies.
// Path: unhappy (invalid input).
func TestConcentrationDamageRule(t *testing.T) {
	t.Parallel()
	for r, want := range map[combat.ConcentrationDamageRule]string{combat.DamageMayBreakConcentration: "damage_may_break_concentration",
		combat.DamageCannotBreakConcentration: "damage_cannot_break_concentration", 0: "unknown", 9: "unknown"} {
		if r.String() != want || r.Valid() != (want != "unknown") {
			t.Errorf("rule %d = %q valid %v, want %q", int(r), r.String(), r.Valid(), want)
		}
	}
	if got := concTracks(t, true, 0, 0).DamageRule(); got != combat.DamageMayBreakConcentration {
		t.Errorf("the zero rule reads as %v, want the default DamageMayBreakConcentration", got)
	}
	in := concTracks(t, true, 3, 0)
	if in.Modifiers().Ability().Value() != 3 || len(in.Circumstances()) != 0 {
		t.Errorf("accessors = %v %v, want the modifier of 3 and no circumstance", in.Modifiers(), in.Circumstances())
	}
}
