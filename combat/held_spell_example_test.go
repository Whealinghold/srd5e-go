package combat_test

import (
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// A spell with a casting time of a Magic action can be readied; the host asks the
// spellcasting rules to begin the Concentration the hold needs.
func ExampleReadySpell() {
	caster := combat.MustCombatantID("mage")
	res := combat.ReadySpell(caster, combat.MustReadyTriggerID("door opens"), "fireball", content.NewCastingTimeMagicAction())
	applied := res.(combat.ReadySpellApplied)
	fmt.Println(applied.Hold.Spell, applied.Events[0].Kind())

	_, refused := combat.ReadySpell(caster, "door opens", "fireball", content.NewCastingTimeBonusAction()).(combat.ReadySpellNotAnAction)
	fmt.Println(refused)
	// Output:
	// fireball spell_held
	// true
}
