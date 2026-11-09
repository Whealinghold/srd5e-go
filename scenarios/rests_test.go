package scenarios_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

// A Fighter with five d10 Hit Dice spends two on a Short Rest, then finishes a
// Long Rest, which regains both; an interrupted Long Rest after 3 hours grants
// the Short Rest benefits and recharges Short Rest features but is not a finished
// Long Rest (SRD 5.2.1, "Short Rest", p. 187; "Long Rest", p. 185). Cites
// rests-and-hit-dice.md FR-026, FR-032, FR-036, FR-039, FR-061, FR-072, FR-054.
func TestRestsAndHitDice(t *testing.T) {
	t.Parallel()
	ctx, id := context.Background(), character.MustIdentifier("fighter")
	d10 := content.HitDieD10
	pool, err := character.NewHitDicePool(map[content.HitPointDie]int{d10: 5})
	if err != nil {
		t.Fatal(err)
	}
	spender, err := character.NewHitDieSpender(dicetest.NewFixedRoller(6, 10))
	if err != nil {
		t.Fatal(err)
	}
	spent, regained := map[content.HitPointDie]int(nil), 0
	for range 2 {
		cmd, err := character.NewSpendHitPointDieCommand(id, pool, spent, d10, ability.MustScore(14).Modifier(), character.ShortRest)
		if err != nil {
			t.Fatal(err)
		}
		res, err := spender.SpendHitPointDie(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		applied := res.(character.SpendHitPointDieApplied)
		regained += applied.HitPointsRegained()
		spent = map[content.HitPointDie]int{d10: applied.SpentHitDice().Count(d10)}
	}
	if regained != 8+12 || spent[d10] != 2 {
		t.Fatalf("regained %d Hit Points with %d dice spent, want 20 and 2", regained, spent[d10])
	}
	stored := character.MustSpentHitDice(spent)

	taker := character.NewRestTaker()
	longEnd := func(end character.RestEnd) character.TakeRestResult {
		rep := character.MustRestReport(character.LongRest, character.StandardLongRest, end)
		return taker.TakeRest(ctx, character.MustTakeRestCommand(id, rep, 20, 0, nil, stored))
	}

	interrupted := longEnd(character.NewRestStopped(character.TakingDamage, 3, 0)).(character.TakeRestAppliedAfterInterruption)
	ev := interrupted.Applied()
	if ev.BenefitsOf() != character.ShortRest || ev.Source() != character.LongRestInterrupted ||
		!ev.BenefitsOf().Recharges(content.RechargeShortRest) || ev.BenefitsOf().Recharges(content.RechargeLongRest) ||
		interrupted.SpentHitDice().Count(d10) != 2 {
		t.Errorf("interrupted Long Rest = %#v", interrupted)
	}

	finished := longEnd(character.NewRestRanItsCourse(8, 0)).(character.TakeRestApplied)
	if fin := finished.Applied(); fin.BenefitsOf() != character.LongRest || fin.Source() != character.RestFinished ||
		!fin.BenefitsOf().Recharges(content.RechargeLongRest) || !fin.BenefitsOf().Recharges(content.RechargeShortOrLongRest) ||
		fin.BenefitsOf().Recharges(content.RechargeShortRest) || finished.SpentHitDice().Total() != 0 {
		t.Errorf("finished Long Rest = %#v", finished)
	}
	var regainedDice int
	for _, e := range finished.Applied().Effects() {
		if h, ok := e.(character.HitPointDiceRegained); ok {
			regainedDice = h.Count()
		}
	}
	if regainedDice != 2 {
		t.Errorf("Hit Point Dice regained = %d, want 2", regainedDice)
	}

	// Only the finished Long Rest starts the 16-hour clock (FR-072).
	soon, err := character.NewLongRestFinishedHoursAgo(10)
	if err != nil {
		t.Fatal(err)
	}
	again := character.MustTakeRestCommand(id, character.MustRestReport(character.LongRest, character.StandardLongRest, character.NewRestRanItsCourse(8, 0)), 20, 0, soon, stored)
	ns, ok := taker.TakeRest(ctx, again).(character.TakeRestNotStarted)
	if !ok || len(ns.Blockers()) != 1 || ns.Blockers()[0].(character.RestStartTooSoonAfterLongRest).HoursStillToWait() != 6 {
		t.Errorf("a second Long Rest 10 hours later = %#v", ns)
	}
}

// One RestApplied is read by two owners of Table R through their public functions (the Fighter.s is in TestFighterFlow): a
// Short Rest leaves the Wizard.s Arcane Recovery used and returns an Orc.s Adrenaline Rush while its Relentless Endurance
// stays spent; a Long Rest returns all of them (SRD 5.2.1, "Short Rest", "Long Rest", p. 187;
// "Level 1: Second Wind", p. 48; "Level 1: Arcane Recovery", p. 78; "Orc Traits", p. 85). Cites rests-and-hit-dice.md FR-060,
// FR-061, class-wizard.md FR-110, species-trait-behaviour.md FR-032, FR-063.
func TestRestOwnersReadTheSameRestApplied(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	orc := must(character.NewSpeciesTraits(content.MustSpeciesID("orc"), character.NoSpeciesChoices{}))
	orc = must(must(orc.WithExpended(character.AdrenalineRushUse, 1)).WithExpended(character.RelentlessEnduranceUse, 1))
	pc := must(character.New(character.NewParams{
		ID: character.MustIdentifier("orc-1"), Class: content.MustClassID("fighter"), HitPointDie: content.HitDieD10,
		Species: content.MustSpeciesID("orc"), Background: content.MustBackgroundID("soldier"), AbilityScores: scoresFor(t, 16, 12, 14, 10, 10, 10),
	})).Character()
	for _, c := range []struct {
		name                      string
		kind                      character.RestKind
		wantAdrenaline, wantRelnt int
		wantArcaneUsed            bool
	}{
		{"short rest", character.ShortRest, 0, 1, true},
		{"long rest", character.LongRest, 0, 0, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rest := finishedRest(t, pc.ID(), c.kind)
			renewed, err := character.NewRenewSpeciesTraitUses().Run(ctx, character.RenewSpeciesTraitUsesCommand{Character: pc, Traits: orc, Rest: rest})
			back, ok := renewed.(character.RenewSpeciesTraitUsesSucceeded)
			if err != nil || !ok {
				t.Fatalf("RenewSpeciesTraitUses = %#v, %v", renewed, err)
			}
			if got := back.Traits().ExpendedOf(character.AdrenalineRushUse); got != c.wantAdrenaline {
				t.Errorf("Adrenaline Rush expended = %d, want %d", got, c.wantAdrenaline)
			}
			if got := back.Traits().ExpendedOf(character.RelentlessEnduranceUse); got != c.wantRelnt {
				t.Errorf("Relentless Endurance expended = %d, want %d", got, c.wantRelnt)
			}
			if got := (character.WizardResources{}).WithArcaneRecoveryUsed().AfterRest(c.kind).ArcaneRecoveryUsed(); got != c.wantArcaneUsed {
				t.Errorf("Arcane Recovery used after a %v = %v, want %v", c.kind, got, c.wantArcaneUsed)
			}
			if got := rest.BenefitsOf(); got != c.kind {
				t.Errorf("BenefitsOf = %v, want %v", got, c.kind)
			}
		})
	}
}
