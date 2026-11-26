package combat_test

import (
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
)

// A monster casts a spell from its Spellcasting trait: at the spell's own level, with the printed
// save DC, and with its use counted in the limited-use state. The monster and the spell are synthetic.
func ExampleMonsterSpellcastingRules_Cast() {
	monster := mscMonster(new(testing.T), nil)
	id := combat.MustCombatantID("mage")
	caster := luMust(combat.NewMonsterSpellcaster(monster, id))
	state := luMust(combat.NewMonsterSpellcastingState(id))
	uses := luMust(combat.StartLimitedUse(monster, id, combat.OutsideLair))
	context := luMust(combat.NewMonsterCastContext(true, true, combat.ConditionActionAvailabilityAvailable))
	fireball := luMust(combat.NewSpellUseRef(1, 2)) // the 1/Day Each Fireball of the fixture

	out, err := combat.MonsterSpellcastingRules{}.Cast(caster, state, uses, combat.NewMonsterListCast(fireball, content.CastingTimeMode{}), seFireball(), nil, context)
	performed := out.(combat.MonsterCastPerformed)
	level, _ := performed.Report.CastAtLevel()
	dc, _ := performed.Report.SaveDC()
	fmt.Println(err, level, dc, performed.Transition.LimitedUse().AvailabilityOfSpell(fireball))
	// Output: <nil> 3 14 exhausted
}
