package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: doc/specs/dispel-and-counterspell.md FR-016, FR-021, FR-024 - a Dispel Magic whose ability check misses the DC (10 plus the spell's level) leaves the level 5 buff in the registry; the host carries that registry into a second cast whose check reaches the DC, and only then does the buff end (SRD 5.2.1, "Dispel Magic", p. 124).
// Path: unhappy (failure), then happy.
func TestDispelMagicFailsThenSucceedsOnTheCarriedRegistry(t *testing.T) {
	t.Parallel()
	goblin := must(combat.NewDispelCreature(combat.MustCombatantID("goblin")))
	reg := must(combat.NewOngoingSpellRegistry([]combat.HeldSpell{
		{Target: goblin, Spell: dcSpell(t, "buff-5", "haste", 5, combat.DispelUnrestricted{}, combat.CombatantID(""))},
	}))
	cast := func(reg combat.OngoingSpellRegistry, face int) combat.OngoingSpellsDispelled {
		t.Helper()
		use := must(combat.NewDispelOngoingSpells(dicetest.NewFixedRoller(face)))
		res, err := use.Run(context.Background(), combat.DispelOngoingSpellsCommand{Registry: reg, Target: goblin, CastLevel: character.MustSlotLevel(3),
			AbilityModifier: ability.MustScore(14).Modifier(), Modifiers: d20.NewRollModifiers(modifier.New(0))})
		got, ok := res.(combat.OngoingSpellsDispelled)
		if err != nil || !ok {
			t.Fatalf("Run = %#v, %v, want OngoingSpellsDispelled", res, err)
		}
		return got
	}
	missed := cast(reg, 5) // 5 + 2 = 7, below DC 15
	if len(missed.Checks()) != 1 || len(missed.Conclusion().Ended()) != 0 || len(missed.Conclusion().Registry().SpellsOn(goblin)) != 1 || len(missed.Conclusion().Events()) != 0 {
		t.Fatalf("a missed check = %d ended, %d left, %d events, want the buff left and no event", len(missed.Conclusion().Ended()), len(missed.Conclusion().Registry().SpellsOn(goblin)), len(missed.Conclusion().Events()))
	}
	hit := cast(missed.Conclusion().Registry(), 13) // 13 + 2 = 15, equal to the DC
	if len(hit.Conclusion().Ended()) != 1 || len(hit.Conclusion().Registry().SpellsOn(goblin)) != 0 {
		t.Errorf("a check equal to the DC = %d ended, %d left, want the buff ended", len(hit.Conclusion().Ended()), len(hit.Conclusion().Registry().SpellsOn(goblin)))
	}
}

// Requirement: doc/specs/dispel-and-counterspell.md FR-031, FR-034, FR-036 - a wizard whose Constitution save reaches the Counterspell DC resists it: the result is CounterspellResistedResult, not a counter, so the cast goes on (SRD 5.2.1, "Counterspell", p. 120).
// Path: unhappy (boundary, the save equals the DC).
func TestCounterspellResistedByTheCasterSave(t *testing.T) {
	t.Parallel()
	wizard := combat.MustCombatantID("wizard")
	cast := must(combat.NewCastObservation("haste", wizard, character.CastSourceSlot, combat.NewUsedComponents(true, true, true), true, content.NewCastingTimeMagicAction()))
	dc := character.Stats(character.CasterWizard, ability.MustScore(16).Modifier(), character.ProficiencyBonusFor(must(character.NewLevel(3)))).SaveDC()
	run := func(face int) combat.CounterspellResult {
		t.Helper()
		use := must(combat.NewCounterCastSpell(dicetest.NewFixedRoller(face)))
		res, err := use.Run(context.Background(), combat.CounterCastSpellCommand{Cast: cast, Counterspeller: combat.MustCombatantID("goblin"), DistanceFeet: 30,
			SeesCaster: true, SaveDC: dc, CasterSave: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(2))}})
		concluded, ok := res.(combat.CounterCastSpellConcluded)
		if err != nil || !ok {
			t.Fatalf("Run = %#v, %v, want CounterCastSpellConcluded", res, err)
		}
		return concluded.Result()
	}
	if _, ok := run(19).(combat.CounterspellResistedResult); !ok {
		t.Errorf("a high save = %#v, want CounterspellResistedResult", run(19))
	}
	if _, ok := run(1).(combat.CounterspellCountered); !ok {
		t.Errorf("a natural 1 save = %#v, want CounterspellCountered", run(1))
	}
}
