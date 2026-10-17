package d20_test

import (
	"context"
	"errors"
	"fmt"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// A rolled saving throw with Advantage: the higher of the two d20s is used.
func ExampleResolver_Resolve() {
	roller := dicetest.NewFixedRoller(4, 15) // a scripted roller; production code passes a real one
	resolver, err := d20.NewResolver(roller)
	if err != nil {
		fmt.Println(err)
		return
	}

	res, err := resolver.Resolve(context.Background(), d20.SavingThrowCommand{
		Ability: ability.Dexterity,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{
			Modifiers:     d20.NewRollModifiers(modifier.New(2)).WithProficiencyBonus(modifier.New(2)),
			Circumstances: []modifier.RollCircumstance{modifier.Advantage{}},
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	switch r := res.(type) {
	case d20.RolledCheck:
		c := r.Check()
		fmt.Println(c.Roll().Dice(), c.Total(), c.Outcome())
	case d20.RolledAttack, d20.ForcedFailure:
		fmt.Println("not reached")
	}
	// Output: [4 15] 19 success
}

// A natural 20 is a Critical Hit whatever the Armor Class.
func ExampleResolveAttack() {
	roll := d20.MustRoll(modifier.Normal, 20)
	atk, err := d20.ResolveAttack(roll, d20.NewRollModifiers(modifier.New(-5)), 30, d20.DefaultCriticalRange)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(atk.Total(), atk.Hit(), atk.IsCriticalHit())
	// Output: 15 true true
}

// A saving throw may be chosen to fail; an ability check may not.
func ExampleFailWithoutRoll() {
	_, err := d20.FailWithoutRoll(d20.AbilityCheck, d20.ChosenToFail)
	fmt.Println(err)
	// Output: d20: forced failure not allowed: only a saving throw can be chosen to fail
}

// A Resolver is built once from a roller and reused. A nil roller is reported
// as an error, not a panic.
func ExampleNewResolver() {
	resolver, err := d20.NewResolver(dicetest.NewFixedRoller(10))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := resolver.Resolve(context.Background(), d20.AbilityCheckCommand{
		Ability: ability.Strength,
		Target:  d20.DCEasy,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(1))},
	})
	fmt.Println(res.Kind(), res.Outcome(), err)

	_, err = d20.NewResolver(nil)
	fmt.Println(errors.Is(err, srderr.ErrInvalidInput))
	// Output:
	// ability_check success <nil>
	// true
}

// An ability check compares the d20 plus modifiers with a Difficulty Class.
func ExampleAbilityCheckCommand() {
	resolver, err := d20.NewResolver(dicetest.NewFixedRoller(9))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := resolver.Resolve(context.Background(), d20.AbilityCheckCommand{
		Ability: ability.Charisma,
		Target:  d20.DCMedium,
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(3))},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	if r, ok := res.(d20.RolledCheck); ok {
		fmt.Println(r.Check().Total(), r.Outcome())
	}
	// Output: 12 failure
}

// An attack roll compares the total with an Armor Class; a natural 20 is a
// Critical Hit.
func ExampleAttackRollCommand() {
	resolver, err := d20.NewResolver(dicetest.NewFixedRoller(20))
	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := resolver.Resolve(context.Background(), d20.AttackRollCommand{
		Ability: ability.Strength,
		Target:  d20.ArmorClass(30),
		Attempt: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(5))},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	if r, ok := res.(d20.RolledAttack); ok {
		a := r.Attack()
		fmt.Println(a.Total(), a.Hit(), a.IsCriticalHit())
	}
	// Output: 25 true true
}
