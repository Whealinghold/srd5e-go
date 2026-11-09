package scenarios_test

import (
	"context"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A ranger coaxes a Hostile Beast (Intelligence 3, default DC 15) with Animal Handling,
// helped by an ally whose Help grants Advantage: Hostile Disadvantage and the caller's
// Advantage cancel, so one d20 is rolled; 11 plus Wisdom +2 and Proficiency Bonus +2 reaches
// 15 and the Beast does as urged. A second urging in the same way after a 5 fails and
// waits 24 hours, and an urging of a Willing or Unwilling monster rolls nothing (SRD 5.2.1,
// "Influence [Action]", p. 184; "Hostile [Attitude]", p. 183; "They Don't Stack", p. 8).
// Cites social-interaction.md FR-013 to FR-016, FR-018, FR-019.
func TestInfluenceFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	score, err := ability.NewScore(3)
	if err != nil {
		t.Fatal(err)
	}
	beast, err := character.NewInfluenceTarget(score, content.CreatureTypeBeast, character.DoesNotUnderstand, character.WithAttitude(character.Hostile))
	if err != nil {
		t.Fatal(err)
	}
	if !character.IsSuggested(character.Coax, beast) {
		t.Fatal("Coax is the table's row for a Beast")
	}
	attempt := character.NewInfluenceRoll(
		map[ability.Ability]modifier.Modifier{ability.Wisdom: modifier.New(2)},
		character.WithProficiencyBonus(modifier.New(2)), character.WithCircumstances(modifier.Advantage{}))
	cmd, err := character.NewInfluenceHesitantCommand(character.Coax, beast, attempt)
	if err != nil {
		t.Fatal(err)
	}

	roller := dicetest.NewFixedRoller(11, 5)
	use, err := character.NewInfluenceResolver(roller)
	if err != nil {
		t.Fatal(err)
	}
	first, err := use.ResolveInfluence(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	won, ok := first.(character.ResolveInfluenceCheckSucceeded)
	if !ok || won.Check().Total() != 15 || won.DC() != 15 || won.DCBasis() != character.DefaultRule || won.Attitude() != character.Hostile {
		t.Fatalf("first urging = %#v", first)
	}
	second, err := use.ResolveInfluence(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	lost, ok := second.(character.ResolveInfluenceCheckFailed)
	if !ok || lost.RetryAfter() != 24*time.Hour || lost.Check().Total() != 9 {
		t.Fatalf("second urging = %#v", second)
	}
	for i, req := range roller.Requests() {
		if req != dice.MustExpression(1, dice.D20(), 0) {
			t.Errorf("request %d = %v, want one d20", i, req)
		}
	}
	for _, c := range []character.InfluenceCommand{character.InfluenceWillingCommand{}, character.InfluenceUnwillingCommand{}} {
		if _, err := use.ResolveInfluence(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(roller.Requests()); n != 2 {
		t.Errorf("%d requests after Willing and Unwilling, want 2", n)
	}
}
