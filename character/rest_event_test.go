package character_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// Requirement: doc/specs/rests-and-hit-dice.md FR-068, FR-036 - RestApplied,
// RestInterrupted and HitPointDieSpent are the three variants of RestEvent, each an
// Event that carries its CharacterID, so a host switches exhaustively over the
// family and the events of two characters are distinguishable.
// Path: happy.
func TestRestEventsAreTheClosedFamilyOfEvent(t *testing.T) {
	t.Parallel()
	other := character.MustIdentifier("other")
	cmd := func(id character.Identifier, end character.RestEnd) character.TakeRestCommand {
		rep := character.MustRestReport(character.LongRest, character.StandardLongRest, end)
		return character.MustTakeRestCommand(id, rep, 5, 0, nil, character.SpentHitDice{})
	}
	var events []character.RestEvent
	for _, id := range []character.Identifier{restHero, other} {
		res := character.NewRestTaker().TakeRest(context.Background(), cmd(id, stopped(rDmg, 3, 0))).(character.TakeRestAppliedAfterInterruption)
		events = append(events, res.Applied(), res.Interrupted())
		sp, err := newSpender(t, dicetest.NewFixedRoller(3)).SpendHitPointDie(context.Background(), spendCmdFor(t, id))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, sp.(character.SpendHitPointDieApplied).HitPointDieSpent())
	}
	seen := map[string]int{}
	for i, e := range events {
		var asEvent character.Event = e
		want := restHero
		if i >= 3 {
			want = other
		}
		if asEvent.CharacterID() != want {
			t.Errorf("event %d concerns %v, want %v", i, asEvent.CharacterID(), want)
		}
		switch e.(type) {
		case character.RestApplied:
			seen["applied"]++
		case character.RestInterrupted:
			seen["interrupted"]++
		case character.HitPointDieSpent:
			seen["spent"]++
		default:
			t.Errorf("unexpected RestEvent %T", e)
		}
	}
	if seen["applied"] != 2 || seen["interrupted"] != 2 || seen["spent"] != 2 {
		t.Errorf("seen = %v", seen)
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-048, architecture.md section 4
// (handoffs) - the effects of a RestApplied are a copy.
// Path: unhappy (immutability).
func TestRestAppliedEffectsAreACopy(t *testing.T) {
	t.Parallel()
	res := character.NewRestTaker().TakeRest(context.Background(), simpleRest(t, character.ShortRest, ran(1, 0))).(character.TakeRestApplied)
	res.Applied().Effects()[0] = allHP
	if _, ok := res.Applied().Effects()[0].(character.SpendHitPointDiceAvailable); !ok {
		t.Error("mutating the returned slice changed the event")
	}
}

// Requirement: doc/specs/rests-and-hit-dice.md FR-036 - RestBenefitsGranted has two
// distinct values and the zero value is neither (the zero value is invalid).
// Path: unhappy (boundary: zero value).
func TestRestBenefitsGrantedHasTwoValuesAndAnInvalidZero(t *testing.T) {
	t.Parallel()
	var zero character.RestBenefitsGranted
	if zero == character.RestBenefitsOfShortRest || zero == character.RestBenefitsNone || character.RestBenefitsOfShortRest == character.RestBenefitsNone {
		t.Errorf("values: zero %d, short rest %d, none %d", int(zero), int(character.RestBenefitsOfShortRest), int(character.RestBenefitsNone))
	}
}
