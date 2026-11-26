package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-054 and table 4.5a - the ended-spell event lists the holder then the caster when different (SRD 5.2.1, "Concentration", p. 179).
// Path: per case.
func TestOngoingSpellEndedCombatantsOrder(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	obj, err := combat.NewDispelObject(combat.SpellObjectID("chest"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, requirement, path string
		target                  combat.DispelTarget
		caster                  string
		want                    []string
	}{
		{"creature holder and a different caster", "FR-054", "happy", goblin, "cleric", []string{"goblin", "cleric"}},
		{"creature holder and the same caster", "FR-054", "unhappy (boundary)", goblin, "goblin", []string{"goblin"}},
		{"creature holder and no known caster", "FR-054", "unhappy (boundary)", goblin, "", []string{"goblin"}},
		{"object holder and a caster", "FR-054", "happy", obj, "cleric", []string{"cleric"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			r := dspRegistry(t, dspHold(c.target, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1, caster: c.caster})))
			applied := dspApplied(t, r, dspPlan(t, r, c.target, 3), nil)
			got := applied.Events()[0].Combatants()
			if len(got) != len(c.want) {
				t.Fatalf("Combatants = %v, want %v [%s]", got, c.want, c.requirement)
			}
			for i := range got {
				if got[i] != dspCreature(t, c.want[i]) {
					t.Errorf("Combatants[%d] = %v, want %v [%s]", i, got[i], c.want[i], c.requirement)
				}
			}
		})
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-054 - each event kind is unique and names its past-tense fact; the events are equal by value (combat-foundations FR-015 to FR-019).
// Path: happy.
func TestSpellEndingEventKindsAreUnique(t *testing.T) {
	t.Parallel()
	goblin := dspTarget(t, "goblin")
	r := dspRegistry(t, dspHold(goblin, dspSpell(t, dspSpec{effect: "x", spell: "bless", level: 1, caster: "cleric"})))
	ended := dspApplied(t, r, dspPlan(t, r, goblin, 3), nil).Events()[0].(combat.OngoingSpellEnded)
	again := dspApplied(t, r, dspPlan(t, r, goblin, 3), nil).Events()[0].(combat.OngoingSpellEnded)
	if !ended.Equal(again) {
		t.Error("equal conclusions gave events that are not equal [FR-054]")
	}
	if ended.Kind() == "spell_countered" || ended.Kind() == "counterspell_resisted" {
		t.Errorf("OngoingSpellEnded kind %q collides with another event [FR-054]", ended.Kind())
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-054 and table 4.5a - a counterspell event names the counterspeller, then the casting creature.
// Path: happy.
func TestCounterspellEventsCombatantsOrder(t *testing.T) {
	t.Parallel()
	e := dspEligible(t, character.CastSourceSlot, false, content.NewCastingTimeMagicAction())
	countered := combat.Counterspell{}.Conclude(e, combat.NewCounterspellForcedFailure(dspForcedFailure(t))).(combat.CounterspellCountered)
	got := countered.Event().Combatants()
	if len(got) != 2 || got[0] != dspCreature(t, "goblin") || got[1] != dspCreature(t, "wizard") {
		t.Errorf("SpellCountered combatants = %v, want counterspeller then caster [FR-054]", got)
	}
	resisted := combat.Counterspell{}.Conclude(e, combat.NewCounterspellRolledSave(dspSavingThrow(t, 20, e.Dc().DifficultyClass()))).(combat.CounterspellResistedResult)
	if got := resisted.Event().Combatants(); len(got) != 2 || got[0] != dspCreature(t, "goblin") || got[1] != dspCreature(t, "wizard") {
		t.Errorf("CounterspellResisted combatants = %v [FR-054]", got)
	}
}
