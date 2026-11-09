package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// A Wizard 5 with Intelligence 18 has spell save DC 15 and a spell attack bonus of +7, and the slots of
// the Wizard Features table.
func ExampleStats() {
	stats := character.Stats(character.CasterWizard, ability.MustScore(18).Modifier(), character.ProficiencyBonusFor(character.MustLevel(5)))
	pools := character.SlotsFor(character.MustClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 5}}))
	fmt.Println(stats.SaveDC().Value(), stats.AttackBonus().Value(), pools.Spellcasting().Maximum(character.MustSlotLevel(3)))
	// Output: 15 7 2
}

// Expending the only slot of a Wizard 1 leaves none, and a third request is rejected.
func ExampleExpendSpellSlot_Do() {
	levels := character.MustClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 1}})
	created, _ := character.NewCreateSpellcasting().Do(context.Background(), character.CreateSpellcastingCommand{Character: character.MustIdentifier("hero"), Levels: levels})
	state := created.(character.CreateSpellcastingCreated).State()
	uc := character.NewExpendSpellSlot()
	for range 3 {
		res, _ := uc.Do(context.Background(), character.ExpendSpellSlotCommand{State: state, Pool: character.PoolSpellcasting, Level: character.MustSlotLevel(1)})
		switch r := res.(type) {
		case character.ExpendSpellSlotExpended:
			state = r.State()
			fmt.Println("expended, remaining", state.Slots(character.PoolSpellcasting).Remaining(character.MustSlotLevel(1)))
		case character.ExpendSpellSlotRejected:
			fmt.Println("rejected:", r.Reason())
		}
	}
	// Output:
	// expended, remaining 1
	// expended, remaining 0
	// rejected: no_slot_available
}
