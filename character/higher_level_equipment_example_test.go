package character_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A character started at level 5 gets 500 GP plus 1d10 x 25 GP; a roll of 7 totals 675 GP.
func ExampleRollHigherLevelStartingMoney() {
	uc, err := character.NewRollHigherLevelStartingMoney(dicetest.NewFixedRoller(7))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := uc.Roll(context.Background(), character.RollHigherLevelStartingMoneyCommand{StartingLevel: character.MustLevel(5)})
	if err != nil {
		fmt.Println(err)
		return
	}
	if rolled, ok := res.(character.RollHigherLevelStartingMoneyRolled); ok {
		roll, _ := rolled.Roll()
		fmt.Println(roll, rolled.Total().Amount(), rolled.Total().Coin())
	}
	// Output: 7 675 GP
}

// The table entry for a level range is a closed value; level 1 has none.
func ExampleHigherLevelEquipmentRules_ForStartingLevel() {
	rules := character.HigherLevelEquipmentRules{}
	fmt.Println(rules.ForStartingLevel(character.MustLevel(1)) == character.NoHigherLevelEquipment{})
	allowance := rules.ForStartingLevel(character.MustLevel(11)).(character.HigherLevelEquipmentAllowance)
	roll := 10
	total, _ := allowance.TotalGold(&roll)
	fmt.Println(allowance.FixedGold().Amount(), total)
	// Output:
	// true
	// 5000 7500
}

// A Trinket is rolled once on a table the host supplies; here every entry is a placeholder.
func ExampleRollTrinket() {
	entries := make([]character.Trinket, character.TrinketCount)
	for i := range entries {
		entries[i], _ = character.NewTrinket(fmt.Sprintf("Entry %d", i+1))
	}
	uc, _ := character.NewRollTrinket(dicetest.NewFixedRoller(100))
	res, _ := uc.Roll(context.Background(), character.RollTrinketCommand{Table: character.MustTrinketTable(entries)})
	rolled := res.(character.RollTrinketRolled)
	fmt.Println(rolled.Roll(), rolled.Trinket().Description())
	// Output: 100 Entry 100
}
