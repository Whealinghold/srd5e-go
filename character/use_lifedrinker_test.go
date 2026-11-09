package character_test

import (
	"errors"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
)

func warLifeCmd(t *testing.T, mutate func(*character.UseLifedrinkerCommand)) character.UseLifedrinkerCommand {
	t.Helper()
	cmd := character.UseLifedrinkerCommand{Character: warChar(t, 5, 16, "", warKnows("lifedrinker")), Attack: character.WarlockPactWeaponHit, DamageType: content.DamageNecrotic,
		Choice: character.HitDieExpend, Pool: warPool(t, 3), Die: content.HitDieD8, Constitution: warMod(14)}
	mutate(&cmd)
	return cmd
}

// Requirement: doc/specs/class-warlock.md FR-077, FR-078, FR-145 - the use case rolls the chosen Hit Point Die (a d8: one request for 1d8) and passes the face to the rules unchanged: a fixed roller returning 1 and 8 heals 1+2 and 8+2, records one spent d8, returns the 1d6 extra damage and the event; the same fixed roll gives the same result twice.
// Path: per case (determinism, extremes 1 and 8).
func TestUseLifedrinkerRollsTheDie(t *testing.T) {
	t.Parallel()
	for _, rolled := range []int{1, 8} {
		roller := dicetest.NewFixedRoller(rolled, rolled)
		uc := must(character.NewUseLifedrinker(roller))
		cmd := warLifeCmd(t, func(*character.UseLifedrinkerCommand) {})
		var results [2]character.UseLifedrinkerApplied
		for i := range results {
			res, err := uc.Run(t.Context(), cmd)
			got, ok := res.(character.UseLifedrinkerApplied)
			if err != nil || !ok {
				t.Fatalf("roll %d: %#v, %v", rolled, res, err)
			}
			results[i] = got
		}
		healed, ok := results[0].HitPointsRegained()
		if healed != rolled+2 || !ok || results[0].SpentHitDice().Count(content.HitDieD8) != 1 || results[0].ExtraDamage().Notation() != "1d6" || results[0].Character().ID() != cmd.Character.ID() {
			t.Errorf("roll %d: healed %d/%v, spent %v, damage %v", rolled, healed, ok, results[0].SpentHitDice(), results[0].ExtraDamage())
		}
		if ev, isEv := results[0].Events()[0].(character.LifedrinkerUsed); !isEv || ev.HitPointsRegained() != rolled+2 {
			t.Errorf("events = %v", results[0].Events())
		}
		if results[0].SpentHitDice().Equal(results[1].SpentHitDice()) == false || results[0].ExtraDamage() != results[1].ExtraDamage() {
			t.Error("the same roll must give the same result")
		}
		if reqs := roller.Requests(); len(reqs) != 2 || reqs[0] != dice.MustExpression(1, dice.D8(), 0) {
			t.Errorf("requests = %v, want two 1d8", reqs)
		}
		results[0].Events()[0] = nil
		if results[0].Events()[0] == nil {
			t.Error("Events must return a copy")
		}
	}
}

// Requirement: doc/specs/class-warlock.md FR-077 - declining the Hit Point Die never reaches the roller, heals nothing and returns the spent dice of the command; a refusal (Fire damage, a miss, no die left) never reaches the roller and leaves the Character unchanged; an invalid die is an error recorded on the span only.
// Path: per case (happy; unhappy: state errors, invalid input).
func TestUseLifedrinkerWithoutARoll(t *testing.T) {
	t.Parallel()
	spent := must(character.NewSpentHitDice(map[content.HitPointDie]int{content.HitDieD8: 3}))
	roller := dicetest.NewFixedRoller(5)
	uc := must(character.NewUseLifedrinker(roller))
	res, err := uc.Run(t.Context(), warLifeCmd(t, func(c *character.UseLifedrinkerCommand) { c.Choice, c.Spent = character.HitDieDecline, spent }))
	got, ok := res.(character.UseLifedrinkerApplied)
	if _, healed := got.HitPointsRegained(); err != nil || !ok || healed || !got.SpentHitDice().Equal(spent) {
		t.Fatalf("decline = %#v, %v", res, err)
	}
	for name, mutate := range map[string]func(*character.UseLifedrinkerCommand){
		"fire":        func(c *character.UseLifedrinkerCommand) { c.DamageType = content.DamageFire },
		"miss":        func(c *character.UseLifedrinkerCommand) { c.Attack = character.WarlockMissed },
		"no die left": func(c *character.UseLifedrinkerCommand) { c.Spent = spent },
	} {
		res, err := uc.Run(t.Context(), warLifeCmd(t, mutate))
		if _, ok := res.(character.UseLifedrinkerRejected); err != nil || !ok {
			t.Errorf("%s = %#v, %v, want a rejection", name, res, err)
		}
	}
	if len(roller.Requests()) != 0 {
		t.Errorf("a roll happened without need: %v", roller.Requests())
	}
	rec, opts := warTelemetry()
	if res, err := must(character.NewUseLifedrinker(roller, opts...)).Run(t.Context(), warLifeCmd(t, func(c *character.UseLifedrinkerCommand) { c.Die = 0 })); res != nil || !srderrIsInvalid(err) {
		t.Errorf("invalid die = %v, %v", res, err)
	}
	if len(rec.Meter.MeasurementsNamed(character.MetricFeatureUsed)) != 0 || len(roller.Requests()) != 0 {
		t.Error("an invalid die must not count as a use or roll")
	}
	if !srderrIsInvalid(errOf(character.NewUseLifedrinker(nil))) {
		t.Error("a nil roller was accepted")
	}
	if _, err := (character.UseLifedrinker{}).Run(t.Context(), warLifeCmd(t, func(*character.UseLifedrinkerCommand) {})); !srderrIsInvalid(err) {
		t.Errorf("zero use case: %v", err)
	}
}

// Requirement: doc/specs/class-warlock.md FR-149 - a roller that returns its error, panics or answers outside its contract is a *srderr.PortError for dice.Port with the cause reachable; the result is nil, the failure is recorded once (span error status, one feature_used failure, one errors increment with error.type) and no state changes.
// Path: per case (unhappy: collaborator failure).
func TestUseLifedrinkerRollerFailures(t *testing.T) {
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
		{"a value out of range", dicetest.NewFixedRoller(9), dicetest.ErrValueOutOfRange},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			rec, opts := warTelemetry()
			res, err := must(character.NewUseLifedrinker(c.roller, opts...)).Run(t.Context(), warLifeCmd(t, func(*character.UseLifedrinkerCommand) {}))
			var pe *srderr.PortError
			if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port || !errors.Is(err, c.cause) {
				t.Fatalf("= %v, %v; want a PortError for dice with the cause", res, err)
			}
			warCheck(t, rec, warWant{span: character.SpanUseLifedrinker, feature: "lifedrinker", outcome: "failure", status: telemetry.StatusError, errorType: srderr.ErrorType(err)})
		})
	}
}
