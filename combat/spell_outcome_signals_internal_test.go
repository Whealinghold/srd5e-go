package combat

import (
	"slices"
	"testing"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-100 to FR-102 - the span, the operation, the counter, the two attribute keys, the two result values and the six table names are pinned to their catalogue spelling and each result carries the class of section 7 (a rolled table applied, a refused Reincarnate refused).
// Path: per case.
func TestSpellOutcomeSignalValuesArePinned(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		got, want               string
	}{
		{"operation", "FR-100", "happy", valueOperationResolveSpellOutcomeTable, "resolve_spell_outcome_table"},
		{"span", "FR-100", "happy", spanResolveSpellOutcomeTable, "srd5e.combat.resolve_spell_outcome_table"},
		{"rolls counter", "FR-102", "happy", metricSpellTableRolls, "srd5e.combat.spell_table_rolls"},
		{"table attribute", "FR-101", "happy", attrSpellTable, "srd5e.combat.spell_table"},
		{"outcome attribute", "FR-102", "happy", attrTableOutcome, "srd5e.combat.table_outcome"},
		{"rolled result", "FR-100", "happy", valueResultSpellTableRolled, "spell_table_rolled"},
		{"refused result", "FR-100", "unhappy (rule interaction)", valueResultSpellTableRefused, "spell_table_refused"},
		{"teleport", "FR-101", "happy", valueSpellTableTeleport, "teleport"},
		{"confusion", "FR-101", "happy", valueSpellTableConfusion, "confusion"},
		{"prismatic spray", "FR-101", "happy", valueSpellTablePrismaticSpray, "prismatic_spray"},
		{"prismatic wall", "FR-101", "happy", valueSpellTablePrismaticWall, "prismatic_wall"},
		{"reincarnate", "FR-101", "happy", valueSpellTableReincarnate, "reincarnate"},
		{"blink", "FR-101", "happy", valueSpellTableBlink, "blink"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			if c.got != c.want {
				t.Errorf("%s = %q, want %q [%s%s]", c.name, c.got, c.want, outcomeSpec, c.requirement)
			}
		})
	}
	if d := describeSpellOutcomeTable(SpellTableRefused{}); d.Class != resultRefused || d.Name != valueResultSpellTableRefused {
		t.Errorf("a refusal is described as %+v", d)
	}
	if d := describeSpellOutcomeTable(SpellTableBlinkRolled{}); d.Class != resultApplied || d.Name != valueResultSpellTableRolled {
		t.Errorf("a rolled table is described as %+v", d)
	}
}

// Requirement: doc/specs/spell-outcome-tables.md section 7.1 and observability.md section 5 - the srd5e.combat.table_outcome values are the closed sets of section 7.1 (4, 4, 8, 10 and 2 values), produced from the enums of the tables.
// Path: happy.
func TestTableOutcomeValuesAreTheClosedSets(t *testing.T) {
	t.Parallel()
	teleport := []string{TeleportMishapOutcome.String(), TeleportSimilarArea.String(), TeleportOffTarget.String(), TeleportOnTarget.String()}
	confusion := []string{ConfusionRowMovesRandomly.String(), ConfusionRowDoesNothing.String(), ConfusionRowAttacksRandomCreature.String(), ConfusionRowChoosesBehavior.String()}
	var spray []string
	for c := PrismaticRed; c <= PrismaticViolet; c++ {
		spray = append(spray, c.String())
	}
	var reincarnate []string
	for face := 1; face <= 10; face++ {
		species, again, _ := ReincarnationSpeciesFor(face)
		if again {
			reincarnate = append(reincarnate, "roll_again")
		} else {
			reincarnate = append(reincarnate, string(species))
		}
	}
	for name, c := range map[string]struct{ got, want []string }{
		"teleport":    {teleport, []string{"mishap", "similar_area", "off_target", "on_target"}},
		"confusion":   {confusion, []string{"moves_randomly", "does_nothing", "attacks_random_creature", "chooses_behavior"}},
		"spray":       {spray, []string{"red", "orange", "yellow", "green", "blue", "indigo", "violet"}},
		"reincarnate": {reincarnate, []string{"roll_again", "dragonborn", "dwarf", "elf", "gnome", "goliath", "halfling", "human", "orc", "tiefling"}},
	} {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("%s outcomes = %v, want %v", name, c.got, c.want)
		}
	}
}
