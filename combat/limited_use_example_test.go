package combat_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A monster with a Recharge 5-6 action uses it, fails to recharge it at the start of its next turn
// and gets it back with a rest.
func ExampleApplyMonsterTurnStart() {
	monster := luMonster(new(testing.T), func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{luPart("Breath", luRecharge(5, 6))}
	})
	breath := luMust(combat.NewLimitedUsePartRef(combat.MonsterPartActions, 0))
	state := luMust(combat.StartLimitedUse(monster, combat.MustCombatantID("monster"), combat.OutsideLair))
	state = state.Spend(breath).(combat.LimitedUseSpent).Transition.State()

	turn := luMust(combat.NewApplyMonsterTurnStart(dicetest.NewFixedRoller(4)))
	res := luMust(turn.Run(context.Background(), combat.ApplyMonsterTurnStartCommand{State: state}))
	started := res.(combat.MonsterTurnStartApplied)
	fmt.Println("rolled", started.Rolls()[0].Face(), started.Rolls()[0].Result(), started.State().AvailabilityOf(breath))

	rested := started.State().AfterRest(character.ShortRest).State()
	fmt.Println("after a Short Rest", rested.AvailabilityOf(breath))
	// Output:
	// rolled 4 not_regained exhausted
	// after a Short Rest available
}
