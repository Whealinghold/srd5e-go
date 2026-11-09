package character_test

import (
	"context"
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

// Requirement: doc/specs/class-warlock.md FR-127, FR-128, FR-130, FR-145 - the use case rolls 1d10 through the dice port and passes the die to the rules unchanged: a fixed roller returning 1 and 10 gives +1 and +10, one use is spent, the event carries the die and the original Character is not altered.
// Path: per case (determinism, extremes 1 and 10).
func TestUseDarkOnesOwnLuckRollsTheDie(t *testing.T) {
	t.Parallel()
	for _, rolled := range []int{1, 10} {
		c := warChar(t, 6, 16, "fiend_patron", character.WarlockState{})
		roller := dicetest.NewFixedRoller(rolled)
		res, err := must(character.NewUseDarkOnesOwnLuck(roller)).Run(context.Background(), character.UseDarkOnesOwnLuckCommand{Character: c, Charisma: warMod(16), Kind: d20.AbilityCheck})
		got, ok := res.(character.UseDarkOnesOwnLuckApplied)
		if err != nil || !ok || got.Bonus() != modifier.New(rolled) || warStateOf(t, got.Character()).DarkOnesOwnLuckSpent() != 1 {
			t.Fatalf("roll %d: = %#v, %v", rolled, res, err)
		}
		if reqs := roller.Requests(); len(reqs) != 1 || reqs[0] != dice.MustExpression(1, dice.D10(), 0) {
			t.Errorf("requests = %v, want one 1d10", reqs)
		}
		if ev := got.Events(); len(ev) != 1 || ev[0].(character.DarkOnesOwnLuckUsed).Rolled() != rolled || ev[0].CharacterID() != c.ID() {
			t.Errorf("events = %+v", ev)
		}
		got.Events()[0] = nil
		if got.Events()[0] == nil {
			t.Error("Events must return a copy")
		}
		if warStateOf(t, c).DarkOnesOwnLuckSpent() != 0 {
			t.Error("the command's Character was altered")
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-128, FR-129, FR-130 - a refusal (an attack roll, a second use on the roll, no use left, not a Fiend Patron) never reaches the roller and leaves the Character unchanged.
// Path: per case (unhappy: state errors).
func TestUseDarkOnesOwnLuckRefusalsDoNotRoll(t *testing.T) {
	t.Parallel()
	spent := warState(warFiend(6), character.WarlockStateParts{DarkOnesOwnLuckSpent: 3})
	for _, c := range []struct {
		name string
		char func() character.Character
		kind d20.Kind
		on   character.LuckOnRoll
		want character.WarlockRejectionKind
	}{
		{"attack roll", func() character.Character { return warChar(t, 6, 16, "fiend_patron", character.WarlockState{}) }, d20.AttackRoll, 0, character.WarlockRollKindNotAllowed},
		{"second use on the roll", func() character.Character { return warChar(t, 6, 16, "fiend_patron", character.WarlockState{}) }, d20.SavingThrow, character.LuckUsedOnRoll, character.WarlockAlreadyUsedOnThisRoll},
		{"no use left", func() character.Character { return warChar(t, 6, 16, "fiend_patron", spent) }, d20.SavingThrow, 0, character.WarlockNoUsesLeft},
		{"no subclass", func() character.Character { return warChar(t, 6, 16, "", character.WarlockState{}) }, d20.SavingThrow, 0, character.WarlockNotFiendPatron},
		{"level 5", func() character.Character { return warChar(t, 5, 16, "fiend_patron", character.WarlockState{}) }, d20.SavingThrow, 0, character.WarlockFeatureNotGained},
	} {
		roller := dicetest.NewFixedRoller(4)
		res, err := must(character.NewUseDarkOnesOwnLuck(roller)).Run(context.Background(), character.UseDarkOnesOwnLuckCommand{Character: c.char(), Charisma: warMod(16), Kind: c.kind, OnRoll: c.on})
		got, ok := res.(character.UseDarkOnesOwnLuckRejected)
		if err != nil || !ok || got.Rejection.Kind != c.want || len(roller.Requests()) != 0 {
			t.Errorf("%s: = %#v, %v, requests %v; want %v and no roll", c.name, res, err, roller.Requests(), c.want)
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-149, architecture.md section 6 - a roller that returns its error, panics or answers outside its contract is a *srderr.PortError for dice.Port with the cause reachable; the result is nil, the failure is recorded once (span error status, one feature_used failure, one errors increment with error.type, no log) and no use is consumed.
// Path: per case (unhappy: collaborator failure).
func TestUseDarkOnesOwnLuckRollerFailures(t *testing.T) {
	t.Parallel()
	boom := errors.New("roller down")
	for _, c := range []struct {
		name   string
		roller character.Roller
		cause  error
	}{
		{"an error", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { return dice.Result{}, boom }), boom},
		{"exhausted", dicetest.NewFixedRoller(), dicetest.ErrExhausted},
		{"a panic", dicetest.CallbackRoller(func(dice.Expression) (dice.Result, error) { panic(boom) }), boom},
		{"a value out of range", dicetest.NewFixedRoller(11), dicetest.ErrValueOutOfRange},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := warTelemetry()
			res, err := must(character.NewUseDarkOnesOwnLuck(c.roller, opts...)).Run(context.Background(), character.UseDarkOnesOwnLuckCommand{
				Character: warChar(t, 6, 16, "fiend_patron", character.WarlockState{}), Charisma: warMod(16), Kind: d20.SavingThrow})
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Fatalf("= %v, %v; want a PortError for dice with the cause", res, err)
			}
			warCheck(t, rec, warWant{span: character.SpanUseDarkOnesOwnLuck, feature: "dark_ones_own_luck", outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err)})
		})
	}
}
