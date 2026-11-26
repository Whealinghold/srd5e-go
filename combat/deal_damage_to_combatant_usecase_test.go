package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func resDeal(tb testing.TB, e combat.Encounter, target string, damage []combat.Damage, adj []combat.DamageAdjustment) (combat.DealDamageToCombatantResult, error) {
	tb.Helper()
	cmd, err := combat.NewDealDamageToCombatantCommand(e, cid(target), damage, adj, combat.OtherSource, combat.NormalHit)
	if err != nil {
		tb.Fatalf("NewDealDamageToCombatantCommand: %v", err)
	}
	uc, err := combat.NewDealDamageToCombatant(newFixed())
	if err != nil {
		tb.Fatal(err)
	}
	return uc.Run(ctxBackground(), cmd)
}

func resDamage(amount int, t content.DamageType) combat.Damage { return combat.MustDamage(amount, t) }

// Requirement: doc/specs/combat-action-resolution.md FR-075, FR-076, FR-077, FR-040, 4.13 - damage that is not an attack is applied in one command: the resolutions and events equal the root's for the same inputs, the host's adjustments (half damage on a made save) are passed through, and the call has no attacker (no AttackPerformed).
// Path: per case.
func TestDealDamageToCombatantApplies(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		damage                  []combat.Damage
		adjustments             []combat.DamageAdjustment
	}{
		{"one component", resSpec + "FR-075, FR-040", "happy", []combat.Damage{resDamage(7, content.DamageFire)}, nil},
		{"two components in one command", resSpec + "FR-075, FR-036", "unhappy (rule interaction)", []combat.Damage{resDamage(3, content.DamageFire), resDamage(4, content.DamageCold)}, nil},
		{"half damage on a made save", resSpec + "FR-076", "unhappy (rule interaction)", []combat.Damage{resDamage(7, content.DamageFire)}, []combat.DamageAdjustment{combat.HalfDamage{}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			e := encABC(t)
			res, err := resDeal(t, e, "b", c.damage, c.adjustments)
			if err != nil {
				t.Fatal(err)
			}
			got := resAs[combat.DealDamageToCombatantCompleted](t, res)
			direct, err := e.ApplyDamage(cid("b"), c.damage, c.adjustments, combat.OtherSource, combat.NormalHit)
			if err != nil || !slices.Equal(got.Resolutions(), direct.Resolutions()) || !slices.Equal(encKinds(got.Events()), encKinds(direct.Events())) || !got.Encounter().Equal(direct.Encounter()) {
				t.Errorf("resolutions %v and events %v, want what the root returns %v and %v: %v [%s]", got.Resolutions(), encKinds(got.Events()), direct.Resolutions(), encKinds(direct.Events()), err, c.requirement)
			}
			if slices.Contains(encKinds(got.Events()), "attack_performed") {
				t.Errorf("a damage call has no attack [FR-077]")
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-085, FR-088, FR-120, 4.15.1 hit to 0 - the prompts that follow damage come after the encounter's events, in turn order, the Speed reviews before the grapple checks: a creature that falls Unconscious gets a Speed review; a grappler that falls Unconscious gets a review and a cue for each grappled creature; a grappler that dies without being Incapacitated gives the cue only; a grappler already Incapacitated gives none.
// Path: per case.
func TestDealDamageToCombatantPrompts(t *testing.T) {
	t.Parallel()
	byG := combat.Grappled{Grappler: cid("g")}
	monster := func(id string, conds ...combat.Condition) combat.Combatant {
		return encC(t, id, combat.SideB, status(t, 7, 7, 0, combat.Alive{}, combat.DiesInstantly, combat.NoAffinities()), conds...)
	}
	cases := []struct {
		name, requirement, path string
		e                       combat.Encounter
		target                  string
		damage                  int
		tail                    []string
		prompts                 []combat.Event
	}{
		{"falls Unconscious", resSpec + "FR-085, 4.15.1", "happy", encABC(t), "b", 20,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied", "movement_speeds_review_became_due"}, nil},
		{"a grappler falls Unconscious", resSpec + "FR-120, 4.15.1 grappler", "unhappy (rule interaction)",
			resThree(t, resPC(t, "a", combat.SideA), resPC(t, "g", combat.SideB), resPC(t, "t", combat.SideA, byG)), "g", 20,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied", "movement_speeds_review_became_due", "grapple_check_became_due"}, nil},
		{"a grappler dies without being Incapacitated", resSpec + "FR-120, Q-15", "unhappy (rule interaction)",
			resThree(t, resPC(t, "a", combat.SideA), monster("g"), resPC(t, "t", combat.SideA, byG)), "g", 7,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "died", "grapple_check_became_due"}, nil},
		{"a grappler already Incapacitated", resSpec + "FR-120", "unhappy (boundary)",
			resThree(t, resPC(t, "a", combat.SideA), resPC(t, "g", combat.SideB, combat.Incapacitated{}), resPC(t, "t", combat.SideA, byG)), "g", 20,
			[]string{"damage_taken", "dropped_to_zero_hit_points", "fell_unconscious", "condition_applied", "movement_speeds_review_became_due"}, nil},
		{"damage that leaves the creature standing", resSpec + "FR-085", "unhappy (boundary)", encABC(t), "b", 5, []string{"damage_taken"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, err := resDeal(t, c.e, c.target, []combat.Damage{resDamage(c.damage, content.DamageSlashing)}, nil)
			if err != nil {
				t.Fatal(err)
			}
			got := resAs[combat.DealDamageToCombatantCompleted](t, res)
			if k := resKinds(got.Events()); !slices.Equal(k, c.tail) {
				t.Errorf("events = %v, want %v [%s]", k, c.tail, c.requirement)
			}
			for _, ev := range got.Events() {
				if cue, ok := ev.(combat.GrappleCheckBecameDue); ok && !slices.Equal(cue.Combatants(), []combat.CombatantID{cid("t"), cid("g")}) {
					t.Errorf("cue names %v, want the grappled t and then the grappler g", cue.Combatants())
				}
			}
			if before, ok := c.e.Find(cid("t")); ok && encMember(t, got.Encounter(), "t").Conditions() != before.Conditions() {
				t.Errorf("Grappled was removed: the call must not apply the end of a grapple [FR-087]")
			}
		})
	}
}

// Requirement: doc/specs/combat-action-resolution.md FR-010, FR-011, FR-012, FR-144, FR-096 - a dead target and an ended encounter are refused, an unknown target is invalid; the command refuses an empty damage list, a nil adjustment, an invalid delivery or hit kind, a zero encounter and a blank id; and a nil roller is refused by the constructor.
// Path: per case.
func TestDealDamageToCombatantStopsAndCommand(t *testing.T) {
	t.Parallel()
	e, _ := encStart(t, resPC(t, "a", combat.SideA), resDead(t, "b", combat.SideB))
	damage := []combat.Damage{resDamage(3, content.DamageSlashing)}
	res, err := resDeal(t, e, "b", damage, nil)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.DealDamageToCombatantRejected](t, res), e, combat.ActionLedger{}, combat.EncounterStateRejection{Rejection: combat.CombatantIsDeadRejection{ID: cid("b")}}, nil)
	res, err = resDeal(t, e, "ghost", damage, nil)
	if inv := resAs[combat.DealDamageToCombatantInvalid](t, res); err != nil || inv.Problem() != (combat.UnknownCombatantProblem{ID: cid("ghost")}) || !inv.Encounter().Equal(e) {
		t.Errorf("= %#v, %v, want an invalid result naming ghost", res, err)
	}
	ended := resEnded(t, e)
	res, err = resDeal(t, ended, "a", damage, nil)
	if err != nil {
		t.Fatal(err)
	}
	resWant(t, resAs[combat.DealDamageToCombatantRejected](t, res), ended, combat.ActionLedger{}, combat.EncounterStateRejection{Rejection: combat.EncounterAlreadyEndedRejection{}}, nil)
	for name, f := range map[string]func() error{
		"empty damage list": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(e, cid("a"), nil, nil, combat.OtherSource, combat.NormalHit)
			return err
		},
		"nil adjustment": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(e, cid("a"), damage, []combat.DamageAdjustment{nil}, combat.OtherSource, combat.NormalHit)
			return err
		},
		"undefined delivery": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(e, cid("a"), damage, nil, combat.DamageDelivery(0), combat.NormalHit)
			return err
		},
		"undefined hit kind": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(e, cid("a"), damage, nil, combat.OtherSource, combat.HitKind(0))
			return err
		},
		"zero encounter": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(combat.Encounter{}, cid("a"), damage, nil, combat.OtherSource, combat.NormalHit)
			return err
		},
		"blank target": func() error {
			_, err := combat.NewDealDamageToCombatantCommand(e, "", damage, nil, combat.OtherSource, combat.NormalHit)
			return err
		},
		"nil roller": func() error { _, err := combat.NewDealDamageToCombatant(nil); return err },
	} {
		if err := f(); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", name, err)
		}
	}
}
