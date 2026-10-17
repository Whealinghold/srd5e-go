package scenarios_test

import (
	"context"
	"fmt"
	"math/rand/v2"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Roll dice described by text, with randomness you supply.
func Example_rollingDice() {
	expr, err := dice.ParseExpression("2d8 + 2")
	if err != nil {
		fmt.Println(err)
		return
	}
	roller := dice.NewSeededRoller(rand.New(rand.NewPCG(7, 7))) // you own the generator
	res, err := dice.Roll(roller, expr)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(expr, res.Total() >= expr.Minimum(), res.Total() <= expr.Maximum())
	// Output: 2d8+2 true true
}

// Resolve a Dexterity saving throw with a modifier from an ability score.
func Example_savingThrow() {
	dex := ability.MustScore(14)
	resolver, err := d20.NewResolver(dice.NewSeededRoller(rand.New(rand.NewPCG(1, 1))))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := resolver.Resolve(context.Background(), d20.SavingThrowCommand{
		Ability: ability.Dexterity,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{
			Modifiers: d20.NewRollModifiers(modifier.New(dex.Modifier().Value())),
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	// An expected outcome is a value, not an error.
	fmt.Println(res.Kind(), res.Outcome() == d20.Success || res.Outcome() == d20.Failure)
	// Output: saving_throw true
}

// Record the signals a resolver emits. The recording doubles stand in for your
// own tracing and metrics backends; a Resolver with no telemetry is silent.
func Example_telemetry() {
	rec := telemetrytest.NewRecorder()
	resolver, err := d20.NewResolver(
		dicetest.NewFixedRoller(12), // a scripted d20; production code passes a real roller
		d20.WithTelemetry(rec.Telemetry()),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = resolver.Resolve(context.Background(), d20.SavingThrowCommand{
		Ability: ability.Wisdom,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(3))},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	spans := rec.Tracer.SpansNamed(d20.SpanResolve)
	if len(spans) != 1 {
		fmt.Println("unexpected span count:", len(spans))
		return
	}
	outcome, _ := spans[0].AttrValue(telemetry.AttrOutcome)
	fmt.Println(spans[0].Name, outcome, len(rec.Meter.MeasurementsNamed(d20.MetricResolved)), len(rec.Violations()))
	// Output: srd5e.d20_test.resolve success 1 0
}
