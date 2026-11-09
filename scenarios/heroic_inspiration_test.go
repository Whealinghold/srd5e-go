package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A Fighter gains Heroic Inspiration, rolls two d20s showing 18 and 3, expends it to
// reroll the 3 (position 1) and is refused a second reroll; a second gain while
// holding it is given to a party member who lacks it (SRD 5.2.1, "Heroic
// Inspiration", p. 8; "Rules Glossary", p. 183). Cites heroic-inspiration.md FR-003,
// FR-005, FR-012, FR-018, FR-024, FR-034.
func TestHeroicInspirationFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	scores := scoresFor(t, 10, 10, 10, 10, 10, 10)
	member := func(id string) character.Character {
		res, err := character.New(character.NewParams{
			ID: character.MustIdentifier(id), Class: "fighter", HitPointDie: content.HitDieD10, Species: "human", Background: "soldier", AbilityScores: scores,
		})
		if err != nil {
			t.Fatal(err)
		}
		return res.Character()
	}
	hero, friend := member("hero"), member("friend")

	gained, err := character.NewGainHeroicInspiration().Gain(ctx, character.GainHeroicInspirationCommand{Holder: hero, Excess: character.LoseExcessInspiration{}})
	if err != nil {
		t.Fatal(err)
	}
	hero = gained.(character.GainHeroicInspirationReceived).Holder()

	roll, err := d20.NewRoll(modifier.WithAdvantage, []int{18, 3})
	if err != nil {
		t.Fatal(err)
	}
	target, err := character.NewRerollTarget(dice.D20(), len(roll.Dice()), 1)
	if err != nil {
		t.Fatal(err)
	}
	expender, err := character.NewExpendHeroicInspiration(dicetest.NewFixedRoller(15))
	if err != nil {
		t.Fatal(err)
	}
	res, err := expender.Expend(ctx, character.ExpendHeroicInspirationCommand{Character: hero, Target: target})
	if err != nil {
		t.Fatal(err)
	}
	rerolled := res.(character.ExpendHeroicInspirationRerolled)
	after, err := roll.ApplyReroll(d20.RerollDie{Index: rerolled.Rerolled().Position(), Face: rerolled.Rerolled().NewValue()})
	if err != nil || after.Natural() != 18 || after.Dice()[1] != 15 {
		t.Fatalf("reroll applied = %v, %v, want dice [18 15]", after.Dice(), err)
	}
	hero = rerolled.Character()
	if again, err := expender.Expend(ctx, character.ExpendHeroicInspirationCommand{Character: hero, Target: target}); err != nil {
		t.Fatal(err)
	} else if _, ok := again.(character.ExpendHeroicInspirationRejected); !ok {
		t.Errorf("second expenditure = %#v, want rejected", again)
	}

	// Holding it again, a second gain is given to a party member who lacks it.
	hero = hero.GrantHeroicInspiration().Character()
	given, err := character.NewGainHeroicInspiration().Gain(ctx, character.GainHeroicInspirationCommand{Holder: hero, Excess: character.GiveExcessInspirationTo{Recipient: friend}})
	if err != nil {
		t.Fatal(err)
	}
	g := given.(character.GainHeroicInspirationExcessGiven)
	if g.Holder().HeroicInspiration() != character.Inspired || g.Recipient().HeroicInspiration() != character.Inspired || len(g.Events()) != 1 {
		t.Errorf("given = %#v", g)
	}
}
