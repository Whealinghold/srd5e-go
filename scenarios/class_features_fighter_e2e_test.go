package scenarios_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// A level 9 Fighter over the bundled Fighter definition fails an ability check and then a saving throw. Tactical Mind adds 1d10 to the check: a roll that still leaves
// the total below the DC spends nothing, a roll that reaches the DC expends one Second Wind use. Indomitable replaces the die of the failed save with a new d20 and
// recomputes the total with the modifiers and the Fighter level: the use is expended even when the new total still fails, and the single use at level 9 is then gone
// (SRD 5.2.1, "Level 2: Tactical Mind", "Level 9: Indomitable", p. 48).
// Requirement: doc/specs/class-fighter.md FR-064, FR-066 to FR-069, FR-077 to FR-082, FR-025 - Tactical Mind and Indomitable end to end through the public API.
// Path: happy, with the unhappy boundary of a reroll that still fails.
func TestFighterTacticalMindAndIndomitableFlow(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewFighterProgression(grpD2Class(t, "fighter")))
	hero := grpD2Flat(t, 10, grpD2CL("fighter", 9))

	failedCheck := grpD2Check(d20.AbilityCheck, modifier.Normal, []int{5}, 0, 15)
	short := dicetest.NewFixedRoller(9)
	res := must2(must2(character.NewUseTacticalMind(prog, short)).Run(ctx, character.UseTacticalMindCommand{Character: hero, FailedCheck: failedCheck}))
	still, ok := res.(character.TacticalMindStillFailed)
	if !ok || still.NewTotal() != 14 || still.Rolled() != 9 || !still.Character().Equal(hero) || len(still.Events()) != 0 {
		t.Fatalf("a d10 of 9 on a 5 against DC 15 = %#v, want still failed at 14 with nothing spent", res)
	}
	if reqs := short.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
		t.Errorf("Tactical Mind requests = %v, want one 1d10", reqs)
	}

	res = must2(must2(character.NewUseTacticalMind(prog, dicetest.NewFixedRoller(10))).Run(ctx, character.UseTacticalMindCommand{Character: hero, FailedCheck: failedCheck}))
	won, ok := res.(character.TacticalMindSucceeded)
	if !ok || won.NewTotal() != 15 || won.Rolled() != 10 || len(won.Events()) != 1 {
		t.Fatalf("a d10 of 10 = %#v, want succeeded at 15", res)
	}
	if _, isUsed := won.Events()[0].(character.TacticalMindUsed); !isUsed {
		t.Errorf("events = %v, want TacticalMindUsed", won.Events())
	}
	state, _, err := character.ReadFeatureState(won.Character().FeatureStates(), character.FighterStateKey())
	if err != nil || state.SecondWindExpended() != 1 {
		t.Errorf("Second Wind expended = %v (%v), want 1", state, err)
	}

	// Indomitable: the failed save is a roll of 8 with +4 against DC 25; the replacement d20 of 5 gives 5 + 4 + 9 = 18, still a failure, and the use is gone.
	save := grpD2Check(d20.SavingThrow, modifier.Normal, []int{8}, 4, 25)
	indomitable := func(face int) character.UseIndomitable {
		return must2(character.NewUseIndomitable(prog, dicetest.NewFixedRoller(face)))
	}
	res2 := must2(indomitable(5).Run(ctx, character.UseIndomitableCommand{Character: won.Character(), FailedSave: save, ReplacedDie: 0}))
	failed, ok := res2.(character.IndomitableStillFailed)
	if !ok || failed.NewTotal() != 18 || failed.Rolled() != 5 || len(failed.Events()) != 1 {
		t.Fatalf("Indomitable roll 5 = %#v, want still failed at 18", res2)
	}
	if left := save.Roll().Dice()[0]; left != 8 {
		t.Errorf("the original roll was altered to %d", left)
	}
	again := must2(indomitable(20).Run(ctx, character.UseIndomitableCommand{Character: failed.Character(), FailedSave: save, ReplacedDie: 0}))
	if r, ok := again.(character.IndomitableNotAvailable); !ok || r.Reason != character.FighterNoUsesRemaining {
		t.Errorf("a second Indomitable at level 9 = %#v, want no uses remaining", again)
	}
	// A new character with the use unspent and a roll of 20 reaches the DC.
	good := must2(indomitable(20).Run(ctx, character.UseIndomitableCommand{Character: hero, FailedSave: save, ReplacedDie: 0}))
	if r, ok := good.(character.IndomitableSucceeded); !ok || r.NewTotal() != 33 {
		t.Errorf("Indomitable roll 20 = %#v, want succeeded at 33", good)
	}
	// A Long Rest returns the expended use.
	spentState, _, err := character.ReadFeatureState(failed.Character().FeatureStates(), character.FighterStateKey())
	if err != nil || spentState.IndomitableExpended() != 1 {
		t.Fatalf("Indomitable expended = %v (%v), want 1", spentState, err)
	}
	rested := must2(character.WithFeatureState(failed.Character(), character.FighterStateKey(), spentState.AfterRest(character.LongRest)))
	r := must2(indomitable(20).Run(ctx, character.UseIndomitableCommand{Character: rested, FailedSave: save, ReplacedDie: 0}))
	if _, ok := r.(character.IndomitableSucceeded); !ok {
		t.Errorf("Indomitable after a Long Rest = %#v, want available", r)
	}
}

// Every refusal of Tactical Mind and Indomitable is a normal result that rolls no die and leaves the character unchanged, in the spec's order: not a Fighter, feature not
// gained, wrong kind of check, check not failed, no uses remaining, and for Indomitable a die index that names no die.
// Requirement: doc/specs/class-fighter.md FR-064 to FR-066, FR-077 to FR-079, FR-127, FR-129 - the refusal reasons of the two use cases.
// Path: unhappy (boundary, state error).
func TestFighterReactionsRefuseWithoutRolling(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewFighterProgression(grpD2Class(t, "fighter")))
	failedCheck := grpD2Check(d20.AbilityCheck, modifier.Normal, []int{5}, 0, 15)
	save := grpD2Check(d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)
	spent := func(c character.Character, sw, ind int) character.Character {
		return must2(character.WithFeatureState(c, character.FighterStateKey(), character.MustFighterState(sw, 0, ind, "", "")))
	}
	for _, c := range []struct {
		name  string
		hero  character.Character
		check d20.CheckResult
		want  character.FighterReason
	}{
		{"a Wizard", grpD2Flat(t, 10, grpD2CL("wizard", 5)), failedCheck, character.FighterNotAFighter},
		{"level 1", grpD2Flat(t, 10, grpD2CL("fighter", 1)), failedCheck, character.FighterFeatureNotGained},
		{"a saving throw", grpD2Flat(t, 10, grpD2CL("fighter", 2)), save, character.FighterNotAnAbilityCheck},
		{"a check that succeeded", grpD2Flat(t, 10, grpD2CL("fighter", 2)), grpD2Check(d20.AbilityCheck, modifier.Normal, []int{15}, 0, 15), character.FighterCheckNotFailed},
		{"no Second Wind use left", spent(grpD2Flat(t, 10, grpD2CL("fighter", 2)), 2, 0), failedCheck, character.FighterNoUsesRemaining},
	} {
		roller := dicetest.NewFixedRoller(5)
		res, err := must2(character.NewUseTacticalMind(prog, roller)).Run(ctx, character.UseTacticalMindCommand{Character: c.hero, FailedCheck: c.check})
		if r, ok := res.(character.TacticalMindNotAvailable); err != nil || !ok || r.Reason != c.want || len(roller.Requests()) != 0 {
			t.Errorf("Tactical Mind, %s = %#v, %v (requests %v), want not available %v", c.name, res, err, roller.Requests(), c.want)
		}
	}
	withAdvantage := grpD2Check(d20.SavingThrow, modifier.WithAdvantage, []int{3, 8}, 4, 15)
	for _, c := range []struct {
		name string
		hero character.Character
		save d20.CheckResult
		die  int
		want character.FighterReason
	}{
		{"a Wizard", grpD2Flat(t, 10, grpD2CL("wizard", 9)), save, 0, character.FighterNotAFighter},
		{"level 8", grpD2Flat(t, 10, grpD2CL("fighter", 8)), save, 0, character.FighterFeatureNotGained},
		{"an ability check", grpD2Flat(t, 10, grpD2CL("fighter", 9)), failedCheck, 0, character.FighterNotASavingThrow},
		{"a die index that names no die", grpD2Flat(t, 10, grpD2CL("fighter", 9)), withAdvantage, 2, character.FighterDieNotRolled},
		{"a save that succeeded", grpD2Flat(t, 10, grpD2CL("fighter", 9)), grpD2Check(d20.SavingThrow, modifier.Normal, []int{18}, 4, 15), 0, character.FighterSaveNotFailed},
		{"no use left", spent(grpD2Flat(t, 10, grpD2CL("fighter", 9)), 0, 1), save, 0, character.FighterNoUsesRemaining},
	} {
		roller := dicetest.NewFixedRoller(5)
		res, err := must2(character.NewUseIndomitable(prog, roller)).Run(ctx, character.UseIndomitableCommand{Character: c.hero, FailedSave: c.save, ReplacedDie: c.die})
		if r, ok := res.(character.IndomitableNotAvailable); err != nil || !ok || r.Reason != c.want || len(roller.Requests()) != 0 {
			t.Errorf("Indomitable, %s = %#v, %v (requests %v), want not available %v", c.name, res, err, roller.Requests(), c.want)
		}
	}
}

// A failing roller is a port failure of the dice port, never a result; bad construction and bad commands are invalid input.
// Requirement: doc/specs/class-fighter.md FR-110, FR-112, FR-149 - roller failure, invalid input and cancellation of the two use cases.
// Path: unhappy (collaborator failure, invalid input, cancellation).
func TestFighterReactionsFailuresAndInvalidInput(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prog := must2(character.NewFighterProgression(grpD2Class(t, "fighter")))
	tmCmd := character.UseTacticalMindCommand{Character: grpD2Flat(t, 10, grpD2CL("fighter", 9)), FailedCheck: grpD2Check(d20.AbilityCheck, modifier.Normal, []int{5}, 0, 15)}
	inCmd := character.UseIndomitableCommand{Character: tmCmd.Character, FailedSave: grpD2Check(d20.SavingThrow, modifier.Normal, []int{5}, 0, 15)}

	res, err := must2(character.NewUseTacticalMind(prog, dicetest.NewFixedRoller())).Run(ctx, tmCmd)
	if res != nil {
		t.Errorf("Tactical Mind with an empty roller returned %#v", res)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)
	res2, err := must2(character.NewUseIndomitable(prog, dicetest.NewFixedRoller())).Run(ctx, inCmd)
	if res2 != nil {
		t.Errorf("Indomitable with an empty roller returned %#v", res2)
	}
	grpD2WantPortFailure(t, err, dice.Port, dicetest.ErrExhausted)

	_, err = character.NewUseTacticalMind(character.FighterProgression{}, dicetest.NewFixedRoller(1))
	grpD2WantInvalid(t, err, "Tactical Mind with a zero progression")
	_, err = character.NewUseTacticalMind(prog, nil)
	grpD2WantInvalid(t, err, "Tactical Mind with a nil roller")
	_, err = character.NewUseIndomitable(character.FighterProgression{}, dicetest.NewFixedRoller(1))
	grpD2WantInvalid(t, err, "Indomitable with a zero progression")
	_, err = character.NewUseIndomitable(prog, nil)
	grpD2WantInvalid(t, err, "Indomitable with a nil roller")

	tm := must2(character.NewUseTacticalMind(prog, dicetest.NewFixedRoller(1)))
	_, err = tm.Run(ctx, character.UseTacticalMindCommand{FailedCheck: tmCmd.FailedCheck})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = tm.Run(ctx, character.UseTacticalMindCommand{Character: tmCmd.Character})
	grpD2WantInvalid(t, err, "a zero check result")
	in := must2(character.NewUseIndomitable(prog, dicetest.NewFixedRoller(1)))
	_, err = in.Run(ctx, character.UseIndomitableCommand{FailedSave: inCmd.FailedSave})
	grpD2WantInvalid(t, err, "a zero character")
	_, err = in.Run(ctx, character.UseIndomitableCommand{Character: inCmd.Character})
	grpD2WantInvalid(t, err, "a zero save result")

	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := tm.Run(cancelled, tmCmd); !errors.Is(err, context.Canceled) {
		t.Errorf("Tactical Mind on a cancelled context: %v, want context.Canceled", err)
	}
	if _, err := in.Run(cancelled, inCmd); !errors.Is(err, context.Canceled) {
		t.Errorf("Indomitable on a cancelled context: %v, want context.Canceled", err)
	}
}
