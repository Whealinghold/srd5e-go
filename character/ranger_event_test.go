package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/class-ranger.md FR-113 - every Ranger event is a past-tense value that implements RangerEvent and Event, carries the Character id it concerns and exposes the data of what happened through its accessors.
// Path: per case (happy).
func TestRangerEventsCarryTheCharacterIDAndData(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	c := rgrRanger(t, 14)
	id := c.ID()
	cast := character.UseFavoredEnemy(id, rgrFacts(t, 14, character.InitialRangerState(), "")).(character.FavoredEnemyUseAccepted).Events()[0].(character.FavoredEnemyCastSpent)
	tireless := must(useTireless(t, dicetest.NewFixedRoller(6)).Run(ctx, character.UseTirelessCommand{Character: c, Wisdom: modifierOf(t, 3)})).(character.UseTirelessAccepted).Events()[0].(character.TirelessUsed)
	veil := character.UseNaturesVeil(id, rgrFacts(t, 14, character.InitialRangerState(), ""), modifierOf(t, 3)).(character.NaturesVeilAccepted).Events()[0].(character.NaturesVeilUsed)
	reduction := must(character.ApplyRangerRest(id, rgrFacts(t, 14, character.InitialRangerState(), ""), character.ShortRest, 2)).Events()[0].(character.ExhaustionReductionGranted)
	made := must(chooseRanger(t).Run(ctx, character.ChooseRangerOptionCommand{Character: rgrRanger(t, 3), Subclass: hunter, Choice: character.RangerChooseHuntersPrey{Option: "horde_breaker"}})).(character.ChooseRangerOptionAccepted).Events()[0].(character.RangerChoiceMade)
	withPrey := rgrWith(t, rgrRanger(t, 3), rgrState(t, character.RangerStateParts{HuntersPrey: "horde_breaker"}))
	replaced := must(chooseRanger(t).Run(ctx, character.ChooseRangerOptionCommand{Character: withPrey, Subclass: hunter, Choice: character.RangerReplaceHuntersPrey{Option: "colossus_slayer", Moment: character.RangerFinishedLongRest}})).(character.ChooseRangerOptionAccepted).Events()[0].(character.RangerChoiceReplaced)
	if cast.Remaining() != 4 || tireless.Rolled() != 6 || tireless.TemporaryHitPoints() != 9 || veil.Remaining() != 2 || reduction.Amount() != 1 || made.Choice() != "hunters_prey" || replaced.Choice() != "hunters_prey" {
		t.Errorf("events = %+v %+v %+v %+v %+v %+v", cast, tireless, veil, reduction, made, replaced)
	}
	for _, e := range []character.RangerEvent{cast, tireless, veil, reduction, made, replaced} {
		var asEvent character.Event = e
		if asEvent.CharacterID() != id && asEvent.CharacterID() != withPrey.ID() {
			t.Errorf("%T carries the id %v, want %v", e, asEvent.CharacterID(), id)
		}
	}
}
