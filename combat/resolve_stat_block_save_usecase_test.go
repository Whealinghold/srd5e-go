package combat_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// sbSaveBreath is a monster whose one action, Steam Breath, is a DC 13 Dexterity save for 5 Fire damage that half
// as much on a success.
func sbSaveBreath(tb testing.TB) (content.Monster, combat.LimitedUsePartRef) {
	tb.Helper()
	part := sbSavePart("Steam Breath", sbSave(ability.Dexterity, sbDC, "", ""),
		sbClause(content.ClauseFailure, nil, sbDamage(5, sbExpr(1, 8, 1), content.DamageFire)), sbClause(content.ClauseSuccess, content.HalfDamageOnly{}))
	m := sbMonster(tb, nil, []content.StatBlockPart{part}, nil)
	return m, sbAction(tb, m, "Steam Breath")
}

func sbSaveParams(tb testing.TB, e combat.Encounter, conc combat.ConcentrationInput, damage int) combat.ResolveStatBlockSaveParams {
	tb.Helper()
	m, ref := sbSaveBreath(tb)
	return combat.ResolveStatBlockSaveParams{Encounter: e, SourceID: cid(luSelf), TargetID: cid("b"), Monster: m, Part: ref,
		Save: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(0))}, Damage: []combat.Damage{combat.MustDamage(damage, content.DamageFire)},
		Delivery: combat.OtherSource, TargetConcentration: conc}
}

func sbSaveRun(tb testing.TB, p combat.ResolveStatBlockSaveParams, faces ...int) (combat.ResolveStatBlockSaveResult, []string) {
	tb.Helper()
	cmd, err := combat.NewResolveStatBlockSaveCommand(p)
	if err != nil {
		tb.Fatalf("NewResolveStatBlockSaveCommand: %v", err)
	}
	roller := newFixed(faces...)
	uc, err := combat.NewResolveStatBlockSave(roller)
	if err != nil {
		tb.Fatal(err)
	}
	res, err := uc.Run(context.Background(), cmd)
	if err != nil {
		tb.Fatalf("Run: %v", err)
	}
	return res, notations(roller)
}

// Requirement: doc/specs/combat-action-resolution.md FR-166, FR-036, FR-038 - the saving throw effect of a stat block part on one target: a failed save applies the clauses and the full damage in one command, a made save with a "half as much" Success clause applies half, an immune target rolls no die and takes nothing, a target that is paralysed fails a Dexterity save without a die, and Concentration of the target follows the damage.
// Path: per case.
func TestResolveStatBlockSave(t *testing.T) {
	t.Parallel()
	m, ref := sbSaveBreath(t)
	window := luMust(combat.NewSaveImmunityWindow(cid("b"), cid(luSelf), ref, 24*time.Hour))
	immune := combat.SaveImmunities{}.Grant(window).Immunities()
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		immunities              combat.SaveImmunities
		conc                    combat.ConcentrationInput
		faces                   []int
		dice                    []string
		adjust                  combat.SaveEffectDamageAdjustment
		hitPointsLost           int
		save                    string
	}{
		{"failed save", resSpec + "FR-166", "happy", sbEncounter(t), combat.SaveImmunities{}, nil, []int{5}, []string{"1d20"}, combat.SaveEffectDamageUnchanged, 10, "failed"},
		{"made save halves", resSpec + "FR-166, FR-038", "happy", sbEncounter(t), combat.SaveImmunities{}, nil, []int{15}, []string{"1d20"}, combat.SaveEffectDamageHalved, 5, "made"},
		{"immune target", resSpec + "FR-166", "unhappy (state)", sbEncounter(t), immune, nil, nil, nil, 0, 0, "none"},
		{"paralysed target", resSpec + "FR-166, FR-045", "unhappy (rule interaction)", sbEncounter(t, combat.Paralyzed{}), combat.SaveImmunities{}, nil, nil, nil, combat.SaveEffectDamageUnchanged, 10, "forced"},
		{"Concentration after the damage", resSpec + "FR-166, FR-078", "unhappy (rule interaction)", sbEncounter(t), combat.SaveImmunities{}, concTracks(t, true, 0, 0), []int{5, 14}, []string{"1d20", "1d20"}, combat.SaveEffectDamageUnchanged, 10, "failed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := c.e
			// The target of the effect is "b"; make it the one with the conditions when the case needs it.
			if c.name == "paralysed target" {
				e, _ = encStart(t, encPC(t, luSelf, combat.SideB, 20, 20), encC(t, "b", combat.SideA, pcStatus(t, 20, 20, 0), combat.Paralyzed{}))
			}
			p := sbSaveParams(t, e, c.conc, 10)
			p.Monster, p.Part, p.Immunities = m, ref, c.immunities
			res, dice := sbSaveRun(t, p, c.faces...)
			got := resAs[combat.ResolveStatBlockSaveCompleted](t, res)
			if !slices.Equal(dice, c.dice) {
				t.Errorf("dice = %v, want %v [%s]", dice, c.dice, c.requirement)
			}
			label := "none"
			if s, ok := got.Save(); ok {
				switch s.(type) {
				case combat.TestForcedFailure:
					label = "forced"
				default:
					label = map[d20.Outcome]string{d20.Success: "made", d20.Failure: "failed"}[s.Outcome()]
				}
			}
			if label != c.save {
				t.Errorf("save = %s, want %s [%s]", label, c.save, c.requirement)
			}
			switch v := got.Effect().(type) {
			case combat.StatBlockSaveEffectApplies:
				if v.DamageAdjustment() != c.adjust {
					t.Errorf("adjustment = %v, want %v [%s]", v.DamageAdjustment(), c.adjust, c.requirement)
				}
			case combat.StatBlockSaveTargetImmune:
				if len(got.Events()) != 0 || len(got.Resolutions()) != 0 {
					t.Errorf("an immune target changed: events %v [FR-166]", resKinds(got.Events()))
				}
			}
			if lost := resHP(t, e, "b") - resHP(t, got.Encounter(), "b"); lost != c.hitPointsLost {
				t.Errorf("Hit Points lost = %d, want %d [%s]", lost, c.hitPointsLost, c.requirement)
			}
		})
	}
	// The one with Concentration: the save is rolled after the damage.
	p := sbSaveParams(t, sbEncounter(t), concTracks(t, true, 0, 0), 10)
	res, _ := sbSaveRun(t, p, 5, 14)
	if l := concLabel(resAs[combat.ResolveStatBlockSaveCompleted](t, res).Concentration()); l != "held DC 10" {
		t.Errorf("Concentration = %q, want held DC 10 [FR-078]", l)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-166, FR-010, FR-011 - the stops of the source and the target come first (an unknown id is invalid, a dead target refused), and a command that names a part with no Saving Throw notation, a save that is not an attempt, an undefined step or an invalid damage is an error at construction.
// Path: per case.
func TestResolveStatBlockSaveStopsAndConstruction(t *testing.T) {
	t.Parallel()
	e := sbEncounter(t)
	ghost := sbSaveParams(t, e, nil, 10)
	ghost.TargetID = cid("ghost")
	res, dice := sbSaveRun(t, ghost)
	if inv := resAs[combat.ResolveStatBlockSaveInvalid](t, res); inv.Problem() != (combat.UnknownCombatantProblem{ID: "ghost"}) || len(dice) != 0 || !inv.Encounter().Equal(e) {
		t.Errorf("unknown target: %#v, dice %v [FR-011]", inv.Problem(), dice)
	}
	dead, _ := encStart(t, encPC(t, luSelf, combat.SideB, 20, 20), resDead(t, "b", combat.SideA))
	res, dice = sbSaveRun(t, sbSaveParams(t, dead, nil, 10))
	resWant(t, res, dead, combat.ActionLedger{}, combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil)
	if len(dice) != 0 {
		t.Errorf("dice = %v, want none for a dead target [FR-012]", dice)
	}
	bad := map[string]func(*combat.ResolveStatBlockSaveParams){
		"no Saving Throw notation": func(p *combat.ResolveStatBlockSaveParams) {
			m := sbMonster(t, nil, sbPlain("Slam"), nil)
			p.Monster, p.Part = m, sbAction(t, m, "Slam")
		},
		"part not in the stat block": func(p *combat.ResolveStatBlockSaveParams) { p.Part = luRef(t, combat.MonsterPartActions, 9) },
		"nil save":                   func(p *combat.ResolveStatBlockSaveParams) { p.Save = nil },
		"undefined step":             func(p *combat.ResolveStatBlockSaveParams) { p.Step = 99 },
		"zero damage":                func(p *combat.ResolveStatBlockSaveParams) { p.Damage = []combat.Damage{{}} },
		"invalid Concentration":      func(p *combat.ResolveStatBlockSaveParams) { p.TargetConcentration = combat.TracksConcentration{} },
		"zero encounter":             func(p *combat.ResolveStatBlockSaveParams) { p.Encounter = combat.Encounter{} },
	}
	for name, mutate := range bad {
		p := sbSaveParams(t, e, nil, 10)
		mutate(&p)
		if _, err := combat.NewResolveStatBlockSaveCommand(p); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: err = %v, want ErrInvalidInput [FR-096]", name, err)
		}
	}
	if _, err := combat.NewResolveStatBlockSave(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("nil roller: err = %v", err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-166, monster-stat-block-actions.md FR-055 - the Step of the command reaches the clauses: a failed save of a part with a First Failure and a Second Failure clause applies the Failure clause and the clause the step names, and with the zero step only the Failure clause.
// Path: per case.
func TestResolveStatBlockSaveFailureStep(t *testing.T) {
	t.Parallel()
	part := sbSavePart("Venom", sbSave(ability.Constitution, sbDC, "", ""), sbClause(content.ClauseFailure, nil),
		sbClause(content.ClauseFirstFailure, nil), sbClause(content.ClauseSecondFailure, nil))
	m := sbMonster(t, nil, []content.StatBlockPart{part}, nil)
	ref := sbAction(t, m, "Venom")
	fail, first, second := content.ClauseFailure, content.ClauseFirstFailure, content.ClauseSecondFailure
	for _, c := range []struct {
		name, requirement, path string
		step                    combat.SaveFailureStep
		want                    []content.EffectClauseLabel
	}{
		{"first failure", resSpec + "FR-166", "happy", combat.SaveFirstFailure, []content.EffectClauseLabel{fail, first}},
		{"second failure", resSpec + "FR-166", "unhappy (rule interaction)", combat.SaveSecondFailure, []content.EffectClauseLabel{fail, second}},
		{"no step", resSpec + "FR-166", "unhappy (empty and minimal)", 0, []content.EffectClauseLabel{fail}},
	} {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := sbSaveParams(t, sbEncounter(t), nil, 1)
			p.Monster, p.Part, p.Step, p.Damage = m, ref, c.step, nil
			res, _ := sbSaveRun(t, p, 5)
			applies, ok := resAs[combat.ResolveStatBlockSaveCompleted](t, res).Effect().(combat.StatBlockSaveEffectApplies)
			if !ok || !slices.Equal(labelsOf(applies.Clauses()), c.want) {
				t.Errorf("effect = %#v, want the clauses %v [%s]", applies, c.want, c.requirement)
			}
		})
	}
}
