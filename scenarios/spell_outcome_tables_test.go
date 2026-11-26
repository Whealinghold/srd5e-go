package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/content/data"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/spell-outcome-tables.md FR-054 - a Reincarnate rolls the species of table R, and each species the table gives that the bundled content set holds (dragonborn, dwarf, elf and human) is found there by the id the use case returned. The other five species of the table (gnome, goliath, halfling, orc, tiefling) are not bundled yet, so the "every species exists" half of FR-054 waits for that content (SRD 5.2.1, "Reincarnate", p. 158).
// Path: unhappy (missing content).
func TestReincarnatedSpeciesAreFoundInTheBundledContent(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	bundle := must2(data.NewBundledSource())
	bundled := map[int]bool{2: true, 3: true, 4: true, 8: true} // dragonborn, dwarf, elf, human
	for face := 2; face <= 10; face++ {
		use := must2(combat.NewResolveSpellOutcomeTable(dicetest.NewFixedRoller(face)))
		res := must2(use.Resolve(ctx, combat.RollReincarnationSpeciesCommand{CreatureType: content.CreatureTypeHumanoid}))
		species := res.(combat.SpellTableReincarnationRolled).Outcome().(combat.ReincarnationSpeciesRolled).Species()
		if !bundled[face] {
			continue
		}
		got := must2(bundle.Lookup(ctx, content.NewKey(content.KindSpecies, content.MustID(string(species)))))
		if _, ok := got.(content.LookupFound); !ok {
			t.Errorf("face %d gave %q, which the bundled content set does not hold", face, species)
		}
	}
}
