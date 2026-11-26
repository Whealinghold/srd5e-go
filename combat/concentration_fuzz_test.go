package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: doc/specs/combat-action-resolution.md FR-157, FR-158, testing.md section 10 - NewConcentrationEndedWithoutSave and NewTracksConcentration never panic on any reason or rule, accept exactly the values the spec allows and otherwise return an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func FuzzNewConcentrationInputs(f *testing.F) {
	for _, n := range []int{0, 1, 2, 3, 4, 5, 6, -1, math.MaxInt} {
		f.Add(n, n)
	}
	state := concState(f, true)
	mods := d20.NewRollModifiers(modifier.New(0))
	f.Fuzz(func(t *testing.T, reason, rule int) {
		_, err := combat.NewConcentrationEndedWithoutSave(character.ConcentrationBreakReason(reason), nil)
		want := reason == int(character.BreakIncapacitated) || reason == int(character.BreakDead)
		if (err == nil) != want || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("NewConcentrationEndedWithoutSave(%d) = %v, want ok %v", reason, err, want)
		}
		got, err := combat.NewTracksConcentration(state, mods, nil, combat.ConcentrationDamageRule(rule))
		valid := rule == 0 || combat.ConcentrationDamageRule(rule).Valid()
		if (err == nil) != valid || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) || (err == nil && !got.DamageRule().Valid()) {
			t.Errorf("NewTracksConcentration(rule %d) = %v, want ok %v", rule, err, valid)
		}
	})
}

// Requirement: doc/specs/combat-action-resolution.md FR-166, FR-096, testing.md section 10 - NewResolveStatBlockSaveCommand never panics on any failure step or damage amount: it accepts a defined step and a positive amount and otherwise returns an error matching srderr.ErrInvalidInput.
// Path: unhappy (invalid input).
func FuzzNewResolveStatBlockSaveCommand(f *testing.F) {
	for _, s := range []int{0, 1, 2, 3, 4, -1, math.MaxInt} {
		f.Add(s, 10)
		f.Add(1, s)
	}
	e := sbEncounter(f)
	f.Fuzz(func(t *testing.T, step, amount int) {
		p := sbSaveParams(t, e, nil, 10)
		p.Step = combat.SaveFailureStep(step)
		if amount > 0 && amount <= 1<<20 {
			p.Damage = []combat.Damage{combat.MustDamage(amount, content.DamageFire)}
		} else {
			p.Damage = []combat.Damage{{}}
		}
		_, err := combat.NewResolveStatBlockSaveCommand(p)
		stepOK := step == 0 || (step >= int(combat.SaveFirstFailure) && step <= int(combat.SaveSubsequentFailures))
		if want := stepOK && amount > 0 && amount <= 1<<20; (err == nil) != want || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("NewResolveStatBlockSaveCommand(step %d, amount %d) = %v, want ok %v", step, amount, err, want)
		}
	})
}
