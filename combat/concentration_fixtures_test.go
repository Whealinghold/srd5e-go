package combat_test

import (
	"context"
	"testing"

	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// concState is a SpellcastingState of the character "wizard" that concentrates on a feature when
// concentrating is true and holds no Concentration otherwise. Fixtures are synthetic.
func concState(tb testing.TB, concentrating bool) character.SpellcastingState {
	tb.Helper()
	levels, err := character.NewClassLevels([]character.ClassLevel{{Class: content.MustClassID("wizard"), Level: 1}})
	if err != nil {
		tb.Fatal(err)
	}
	created, err := character.NewCreateSpellcasting().Do(context.Background(), character.CreateSpellcastingCommand{Character: character.MustIdentifier("wizard"), Levels: levels})
	if err != nil {
		tb.Fatal(err)
	}
	state := created.(character.CreateSpellcastingCreated).State()
	if !concentrating {
		return state
	}
	began, err := character.NewBeginConcentration().Do(context.Background(), character.BeginConcentrationCommand{State: state,
		Subject: character.NewConcentrationSubjectFeature(content.MustFeatureID("synthetic_focus"))})
	if err != nil {
		tb.Fatal(err)
	}
	return began.(character.BeginConcentrationBegan).State()
}

// concTracks is the TracksConcentration of a concentrating state with the save modifier and the rule.
func concTracks(tb testing.TB, concentrating bool, save int, rule combat.ConcentrationDamageRule, circumstances ...modifier.RollCircumstance) combat.TracksConcentration {
	tb.Helper()
	t, err := combat.NewTracksConcentration(concState(tb, concentrating), d20.NewRollModifiers(modifier.New(save)), circumstances, rule)
	if err != nil {
		tb.Fatalf("NewTracksConcentration: %v", err)
	}
	return t
}

// concDeal runs DealDamageToCombatant for the damage on the target with the Concentration input and a fixed
// roller of faces; it returns the result and the dice asked for.
func concDeal(tb testing.TB, e combat.Encounter, target string, amount int, in combat.ConcentrationInput, faces ...int) (combat.DealDamageToCombatantResult, []string) {
	tb.Helper()
	cmd, err := combat.NewDealDamageToCombatantCommand(e, cid(target), []combat.Damage{combat.MustDamage(amount, content.DamageFire)}, nil, combat.OtherSource, combat.NormalHit)
	if err != nil {
		tb.Fatal(err)
	}
	if cmd, err = cmd.WithTargetConcentration(in); err != nil {
		tb.Fatalf("WithTargetConcentration: %v", err)
	}
	roller := newFixed(faces...)
	uc, err := combat.NewDealDamageToCombatant(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(ctxBackground(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res, notations(roller)
}

// concTarget is an encounter whose combatant "b" is a living creature of cur of max Hit Points with the
// conditions; "a" is a healthy ally on the other side.
func concTarget(tb testing.TB, cur, max int, conds ...combat.Condition) combat.Encounter {
	tb.Helper()
	e, _ := encStart(tb, encPC(tb, "a", combat.SideA, 20, 20), encC(tb, "b", combat.SideB, pcStatus(tb, cur, max, 0), conds...))
	return e
}
