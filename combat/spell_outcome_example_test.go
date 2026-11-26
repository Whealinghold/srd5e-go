package combat_test

import (
	"context"
	"fmt"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A Blink caster on the other plane rolls a 5 at the end of its turn and vanishes to the Ethereal Plane.
func ExampleResolveSpellOutcomeTable() {
	use, _ := combat.NewResolveSpellOutcomeTable(dicetest.NewFixedRoller(5))
	res, _ := use.Resolve(context.Background(), combat.RollBlinkTurnEndCommand{Location: combat.BlinkOnOtherPlane})
	switch o := res.(combat.SpellTableBlinkRolled).Outcome().(type) {
	case combat.BlinkVanishesToEtherealPlane:
		fmt.Println("vanishes to", o.Location())
	}
	// Output: vanishes to ethereal_plane
}
