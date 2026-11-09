package character_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// evkSub is the Evoker SubclassDefinition of the bundled Wizard (content/data/class_wizard.json).
func evkSub(tb testing.TB) content.SubclassDefinition {
	tb.Helper()
	found, ok := wzDef(tb).Subclass("evoker").(content.SubclassFound)
	if !ok {
		tb.Fatal("the bundled Wizard has no Evoker")
	}
	return found.Subclass()
}

// evkEvo makes the spell an Evocation spell.
func evkEvo(f *character.WizardSpellFacts) { f.School = content.SchoolEvocation }

// evkDmg makes the spell deal damage.
func evkDmg(f *character.WizardSpellFacts) { f.DealsDamage = true }

// evkAbj makes the spell an Abjuration spell, so it is not an Evocation one.
func evkAbj(f *character.WizardSpellFacts) { f.School = content.SchoolAbjuration }

// evkEvoFact is an Evocation Wizard spell of the level.
func evkEvoFact(id string, level int, mods ...func(*character.WizardSpellFacts)) character.WizardSpellFacts {
	return wzFact(id, level, append([]func(*character.WizardSpellFacts){evkEvo}, mods...)...)
}

// evkMod is the Intelligence modifier of a score.
func evkMod(score int) ability.Modifier { return ability.MustScore(score).Modifier() }

// evkCreatures are n eligible creatures named c1 to cn.
func evkCreatures(n int) []character.SculptCandidate {
	var out []character.SculptCandidate
	for i := 1; i <= n; i++ {
		out = append(out, character.SculptCandidate{ID: character.MustIdentifier(fmt.Sprintf("c%d", i)), Visible: true, Affected: true})
	}
	return out
}

// evkResources is WizardResources with n Overchannel uses.
func evkResources(tb testing.TB, n int) character.WizardResources {
	tb.Helper()
	return must(character.WizardResources{}.WithOverchannelUses(n))
}
