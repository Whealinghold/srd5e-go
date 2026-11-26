package combat_test

import (
	"errors"
	"math"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// Requirement: per case
// Path: per case.
func TestNewPerformAttackCommand(t *testing.T) {
	t.Parallel()
	e := encABC(t)
	opportunity := func(provoker string, v combat.Visibility) combat.AttackCost {
		return combat.ReactionCost{Choice: combat.OpportunityAttackReaction{Provoker: cid(provoker), Means: combat.DepartureBySpeed, Visibility: v}}
	}
	form := func(f combat.AttackForm) combat.ResolveAttackCommand {
		q := meleeParams()
		if f == combat.RangedWeaponAttack || f == combat.RangedSpellAttack {
			q = rangedParams()
		}
		q.Form = f
		q.Damage = slashing(1, 8)
		return attackCommand(t, q)
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*combat.PerformAttackParams)
		ok                      bool
	}{
		{"valid", resSpec + "FR-097", "happy", func(*combat.PerformAttackParams) {}, true},
		{"zero encounter", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Encounter = combat.Encounter{} }, false},
		{"blank attacker", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.AttackerID = "" }, false},
		{"blank target", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.TargetID = "" }, false},
		{"zero Attack command", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Attack = combat.ResolveAttackCommand{} }, false},
		{"nil cost", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Cost = nil }, false},
		{"scene distance differs from the attack's", resSpec + "FR-026, FR-135", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Scene = resSeen(t, 10) }, false},
		{"scene distance 0 against an attack at 0", resSpec + "FR-135", "unhappy (boundary)", func(p *combat.PerformAttackParams) {
			q := meleeParams()
			q.Distance = 0
			p.Attack, p.Scene = attackCommand(t, q), resSeen(t, 0)
		}, true},
		{"knock out with a melee weapon attack", resSpec + "FR-037", "unhappy (rule interaction)", func(p *combat.PerformAttackParams) { p.KnockOut = combat.KnockOut }, true},
		{"knock out with a melee spell attack", resSpec + "FR-037, 4.6.1", "unhappy (rule interaction)", func(p *combat.PerformAttackParams) {
			p.KnockOut, p.Attack, p.Cost = combat.KnockOut, form(combat.MeleeSpellAttack), combat.AttackAlreadyPaid{}
		}, true},
		{"knock out with a ranged weapon attack", resSpec + "FR-136", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.KnockOut, p.Attack, p.Scene = combat.KnockOut, form(combat.RangedWeaponAttack), resSeen(t, 30)
		}, false},
		{"knock out with a ranged spell attack", resSpec + "FR-136", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.KnockOut, p.Attack, p.Scene, p.Cost = combat.KnockOut, form(combat.RangedSpellAttack), resSeen(t, 30), combat.AttackAlreadyPaid{}
		}, false},
		{"undefined knock out choice", resSpec + "FR-136", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.KnockOut = combat.KnockOutChoice(3) }, false},
		{"nil adjustment", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.DamageAdjustments = []combat.DamageAdjustment{nil} }, false},
		{"Opportunity Attack with a melee weapon attack", resSpec + "FR-137", "happy", func(p *combat.PerformAttackParams) { p.Cost = opportunity("b", combat.VisibilitySeen) }, true},
		{"Opportunity Attack with an Unarmed Strike form", resSpec + "FR-137", "happy", func(p *combat.PerformAttackParams) {
			p.Cost, p.Attack = opportunity("b", combat.VisibilitySeen), form(combat.UnarmedStrikeAttack)
		}, true},
		{"Opportunity Attack with a ranged weapon attack", resSpec + "FR-137", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost, p.Attack, p.Scene = opportunity("b", combat.VisibilitySeen), form(combat.RangedWeaponAttack), resSeen(t, 30)
		}, false},
		{"Opportunity Attack with a melee spell attack", resSpec + "FR-137", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost, p.Attack = opportunity("b", combat.VisibilitySeen), form(combat.MeleeSpellAttack)
		}, false},
		{"Opportunity Attack with a ranged spell attack", resSpec + "FR-137", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost, p.Attack, p.Scene = opportunity("b", combat.VisibilitySeen), form(combat.RangedSpellAttack), resSeen(t, 30)
		}, false},
		{"Opportunity Attack whose provoker is not the target", resSpec + "FR-138", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Cost = opportunity("c", combat.VisibilitySeen) }, false},
		{"Opportunity Attack whose provoker is the attacker", resSpec + "FR-139", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.TargetID, p.Cost = cid("a"), opportunity("a", combat.VisibilitySeen)
		}, false},
		{"Visibility seen against a scene that cannot see", resSpec + "FR-140, FR-151", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost, p.Scene = opportunity("b", combat.VisibilitySeen), resScene(t, 5, combat.SightRelationCannotSee, combat.SightRelationCanSee)
		}, false},
		{"Visibility unseen against a scene that sees", resSpec + "FR-140, FR-151", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Cost = opportunity("b", combat.VisibilityUnseen) }, false},
		{"Visibility unseen against a scene that cannot see", resSpec + "FR-140, FR-151", "unhappy (boundary)", func(p *combat.PerformAttackParams) {
			p.Cost, p.Scene = opportunity("b", combat.VisibilityUnseen), resScene(t, 5, combat.SightRelationCannotSee, combat.SightRelationCanSee)
		}, true},
		{"Reaction without a choice", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Cost = combat.ReactionCost{} }, false},
		{"Attack action with a melee spell attack", resSpec + "FR-141", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Attack = form(combat.MeleeSpellAttack) }, false},
		{"Extra Attack with a ranged spell attack", resSpec + "FR-141", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Attack, p.Scene, p.Cost = form(combat.RangedSpellAttack), resSeen(t, 30), combat.ContinuesAttackAction{Means: combat.AttackMeansOtherWeapon}
		}, false},
		{"spell attack paid by a Bonus Action", resSpec + "FR-141", "unhappy (boundary)", func(p *combat.PerformAttackParams) {
			p.Attack, p.Cost = form(combat.MeleeSpellAttack), combat.BonusActionCost{}
		}, true},
		{"unarmed means with a weapon form", resSpec + "FR-142", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost = combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansUnarmedStrike}
		}, false},
		{"weapon means with an Unarmed Strike form", resSpec + "FR-142", "unhappy (invalid input)", func(p *combat.PerformAttackParams) { p.Attack = form(combat.UnarmedStrikeAttack) }, false},
		{"unarmed means with an Unarmed Strike form", resSpec + "FR-142", "happy", func(p *combat.PerformAttackParams) {
			p.Attack = form(combat.UnarmedStrikeAttack)
			p.Cost = combat.BeginsAttackAction{Allowance: combat.BaselineAttackAllowance(), Means: combat.AttackMeansUnarmedStrike}
		}, true},
		{"zero allowance", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost = combat.BeginsAttackAction{Means: combat.AttackMeansOtherWeapon}
		}, false},
		{"zero means", resSpec + "FR-096", "unhappy (invalid input)", func(p *combat.PerformAttackParams) {
			p.Cost = combat.ContinuesAttackAction{}
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := resParams(t, e, "a", "b", resBegin())
			c.edit(&p)
			_, err := combat.NewPerformAttackCommand(p)
			if c.ok != (err == nil) || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) {
				t.Errorf("error = %v, want ok=%v with ErrInvalidInput on refusal [%s]", err, c.ok, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-004 - the command takes its own copy of the adjustments: the host editing its slice afterwards changes neither the command nor the damage.
// Path: unhappy (immutability).
func TestNewPerformAttackCommandCopiesAdjustments(t *testing.T) {
	t.Parallel()
	adj := []combat.DamageAdjustment{combat.NewFlatAdjustment(-5, combat.AllDamage{})}
	p := resParams(t, encABC(t), "a", "b", resBegin())
	p.DamageAdjustments = adj
	cmd, err := combat.NewPerformAttackCommand(p)
	if err != nil {
		t.Fatal(err)
	}
	adj[0] = combat.HalfDamage{}
	uc, _ := combat.NewPerformAttack(newFixed(14, 4))
	res, err := uc.Run(ctxBackground(), cmd)
	if got := resAs[combat.PerformAttackCompleted](t, res).Resolutions()[0].Dealt(); err != nil || got != 2 {
		t.Errorf("dealt %d, want 2 (7 less the flat 5 taken at construction): %v", got, err)
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-144, testing.md section 10 - NewDealDamageToCombatantCommand never panics and refuses a negative amount, an invalid Damage Type and an empty list.
// Path: unhappy (invalid input).
func FuzzNewDealDamageToCombatantCommand(f *testing.F) {
	for _, s := range [][2]int{{7, int(content.DamageSlashing)}, {0, 1}, {-1, 1}, {math.MaxInt, 1}, {5, 0}, {5, 9999}} {
		f.Add(s[0], s[1])
	}
	e := encABC(f)
	f.Fuzz(func(t *testing.T, amount, kind int) {
		d, derr := combat.NewDamage(amount, content.DamageType(kind))
		var list []combat.Damage
		if derr == nil {
			list = []combat.Damage{d}
		}
		_, err := combat.NewDealDamageToCombatantCommand(e, cid("b"), list, nil, combat.OtherSource, combat.NormalHit)
		if (derr == nil) != (err == nil) || (err != nil && !errors.Is(err, srderr.ErrInvalidInput)) {
			t.Errorf("NewDamage(%d, %d) error %v, command error %v: want both or neither", amount, kind, derr, err)
		}
	})
}
