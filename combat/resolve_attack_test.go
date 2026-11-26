package combat_test

import (
	"context"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
	"github.com/Whealinghold/srd5e-go/telemetry"
	"github.com/Whealinghold/srd5e-go/telemetry/telemetrytest"
)

// Requirement: per case
// Path: per case.
func TestResolveAttackOutcomes(t *testing.T) {
	t.Parallel()
	half := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	at := func(ac int, m d20.RollModifiers) func(*combat.ResolveAttackParams) {
		return func(p *combat.ResolveAttackParams) { p.ArmorClass = d20.ArmorClass(ac); p.Modifiers = m }
	}
	unseen := func(there bool) func(*combat.ResolveAttackParams) {
		return func(p *combat.ResolveAttackParams) {
			p.Situation.Visibility = combat.TargetUnseen{TargetAtTargetedLocation: there}
		}
	}
	cases := []struct {
		name, requirement, path string
		edit                    func(*combat.ResolveAttackParams)
		faces                   []int
		requests                []string
		hit, crit               bool
		total, effective        int
		override                combat.MissOverride
	}{
		{"above the AC", `FR-004; ` + srdAttack, "happy", at(15, mods(3, 2)), []int{12}, []string{"1d20"}, true, false, 17, 15, combat.MissOverrideNone},
		{"equals the AC", `FR-005; ` + srdAttack, "unhappy (boundary)", at(15, mods(3, 2)), []int{10}, []string{"1d20"}, true, false, 15, 15, combat.MissOverrideNone},
		{"one below the AC", `FR-006; ` + srdAttack, "unhappy (boundary)", at(15, mods(3, 2)), []int{9}, []string{"1d20"}, false, false, 14, 15, combat.MissOverrideNone},
		{"natural 20 far below the AC", `FR-008; "Rolling 20 or 1", p. 7`, "unhappy (boundary)", at(30, mods(0, 0)), []int{20}, []string{"1d20"}, true, true, 20, 30, combat.MissOverrideNone},
		{"natural 20 behind Half Cover", `FR-020; "Rolling 20 or 1", p. 7; ` + srdCover, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { at(15, mods(-5, 0))(p); p.Cover = half }, []int{20}, []string{"1d20"}, true, true, 15, 17, combat.MissOverrideNone},
		{"natural 19 below the AC", `FR-010; "Rolling 20 or 1", p. 7`, "unhappy (boundary)", at(30, mods(0, 0)), []int{19}, []string{"1d20"}, false, false, 19, 30, combat.MissOverrideNone},
		{"natural 19 at the AC", `FR-010`, "unhappy (boundary)", at(19, mods(0, 0)), []int{19}, []string{"1d20"}, true, false, 19, 19, combat.MissOverrideNone},
		{"natural 1 above the AC", `FR-009; "Rolling 20 or 1", p. 7`, "unhappy (boundary)", at(5, mods(20, 0)), []int{1}, []string{"1d20"}, false, false, 21, 5, combat.MissOverrideNone},
		{"no modifiers at all", `FR-003; "D20 Tests", p. 6`, "unhappy (empty and minimal case)", at(15, mods(0, 0)), []int{15}, []string{"1d20"}, true, false, 15, 15, combat.MissOverrideNone},
		{"Advantage with dice 1 and 20", `FR-011; "Roll Two D20s", p. 8`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) {
			at(15, mods(0, 0))(p)
			p.Circumstances = []modifier.RollCircumstance{advantage}
		}, []int{1, 20}, []string{"2d20"}, true, true, 20, 15, combat.MissOverrideNone},
		{"Disadvantage with dice 1 and 20", `FR-011`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) {
			at(5, mods(0, 0))(p)
			p.Circumstances = []modifier.RollCircumstance{disadvantage}
		}, []int{20, 1}, []string{"2d20"}, false, false, 1, 5, combat.MissOverrideNone},
		{"Half Cover: +7 meets the effective AC 17", `FR-015; ` + srdCover, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(15, mods(7, 0))(p); p.Cover = half }, []int{10}, []string{"1d20"}, true, false, 17, 17, combat.MissOverrideNone},
		{"Half Cover: +6 misses the effective AC 17", `FR-015`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(15, mods(6, 0))(p); p.Cover = half }, []int{10}, []string{"1d20"}, false, false, 16, 17, combat.MissOverrideNone},
		{"Three-Quarters Cover", `FR-013`, "happy", func(p *combat.ResolveAttackParams) {
			p.Cover = []combat.CoverSource{{Cover: combat.ThreeQuartersCover, AttackOriginatesOppositeSide: true}}
		}, []int{15}, []string{"1d20"}, true, false, 20, 20, combat.MissOverrideNone},
		{"natural 20 against a target that is not at the location", `FR-091, FR-036; "Unseen Attackers and Targets", p. 14`, "unhappy (rule interaction)", unseen(false), []int{20, 20}, []string{"2d20"}, false, false, 25, 15, combat.MissOverrideTargetNotAtLocation},
		{"natural 20 against an unseen target at the location", `FR-034, FR-036`, "unhappy (rule interaction)", unseen(true), []int{20, 20}, []string{"2d20"}, true, true, 25, 15, combat.MissOverrideNone},
		{"unseen target and unseen attacker cancel", `FR-038; "They Don't Stack", p. 8`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { unseen(true)(p); p.Situation.TargetCannotSeeAttacker = true }, []int{12}, []string{"1d20"}, true, false, 17, 15, combat.MissOverrideNone},
		{"target cannot see the attacker", `FR-035`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { p.Situation.TargetCannotSeeAttacker = true }, []int{3, 12}, []string{"2d20"}, true, false, 17, 15, combat.MissOverrideNone},
		{"automatic Critical Hit on a hit", `FR-090; "Paralyzed", p. 186`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { p.AutomaticCritical = combat.AutomaticCriticalAutomatic }, []int{12}, []string{"1d20"}, true, true, 17, 15, combat.MissOverrideNone},
		{"automatic Critical Hit on a miss", `FR-090`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { p.AutomaticCritical = combat.AutomaticCriticalAutomatic }, []int{9}, []string{"1d20"}, false, false, 14, 15, combat.MissOverrideNone},
		{"automatic Critical Hit on a natural 1", `FR-090, FR-009`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) {
			at(5, mods(20, 0))(p)
			p.AutomaticCritical = combat.AutomaticCriticalAutomatic
		}, []int{1}, []string{"1d20"}, false, false, 21, 5, combat.MissOverrideNone},
		{"automatic Critical Hit on a natural 20", `FR-090`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { p.AutomaticCritical = combat.AutomaticCriticalAutomatic }, []int{20}, []string{"1d20"}, true, true, 25, 15, combat.MissOverrideNone},
		{"automatic Critical Hit lost to a forced miss", `FR-090, FR-091`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) {
			unseen(false)(p)
			p.AutomaticCritical = combat.AutomaticCriticalAutomatic
		}, []int{20, 20}, []string{"2d20"}, false, false, 25, 15, combat.MissOverrideTargetNotAtLocation},
		{"threshold 19, natural 19 below the AC", `FR-134; "Rolling 20 or 1", p. 7; "Critical Hit", p. 179`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(30, mods(0, 0))(p); p.CriticalThreshold = 19 }, []int{19}, []string{"1d20"}, true, true, 19, 30, combat.MissOverrideNone},
		{"threshold 18, natural 18 below the AC", `FR-134`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(30, mods(0, 0))(p); p.CriticalThreshold = 18 }, []int{18}, []string{"1d20"}, true, true, 18, 30, combat.MissOverrideNone},
		{"threshold 18, natural 17", `FR-134`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(30, mods(0, 0))(p); p.CriticalThreshold = 18 }, []int{17}, []string{"1d20"}, false, false, 17, 30, combat.MissOverrideNone},
		{"threshold 18, natural 1 is still a miss", `FR-134, FR-009`, "unhappy (boundary)", func(p *combat.ResolveAttackParams) { at(5, mods(20, 0))(p); p.CriticalThreshold = 18 }, []int{1}, []string{"1d20"}, false, false, 21, 5, combat.MissOverrideNone},
		{"threshold 19 lost to a forced miss", `FR-134, FR-091`, "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { unseen(false)(p); p.CriticalThreshold = 19 }, []int{19, 19}, []string{"2d20"}, false, false, 24, 15, combat.MissOverrideTargetNotAtLocation},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := meleeParams()
			c.edit(&p)
			res, reqs := runAttack(t, attackCommand(t, p), c.faces...)
			r, ok := res.(combat.AttackResolved)
			if !ok {
				t.Fatalf("result = %#v [%s]", res, c.requirement)
			}
			if !reflect.DeepEqual(reqs, c.requests) {
				t.Errorf("requests = %v, want %v [%s]", reqs, c.requests, c.requirement)
			}
			if r.Hit() != c.hit || r.CriticalHit() != c.crit || r.Attack().Total() != c.total || int(r.EffectiveArmorClass()) != c.effective ||
				r.MissOverride() != c.override || r.ArmorClass() != p.ArmorClass {
				t.Errorf("hit %v crit %v total %d effective %d override %v, want %v %v %d %d %v [%s]", r.Hit(), r.CriticalHit(), r.Attack().Total(),
					r.EffectiveArmorClass(), r.MissOverride(), c.hit, c.crit, c.total, c.effective, c.override, c.requirement)
			}
			// The roll is exactly what d20.ResolveAttack returns for the same roll and effective AC (FR-002, FR-012, FR-121).
			threshold := d20.DefaultCriticalRange
			if p.CriticalThreshold != 0 {
				threshold = p.CriticalThreshold
			}
			want, err := d20.ResolveAttack(r.Attack().Roll(), p.Modifiers, d20.ArmorClass(c.effective), threshold)
			if err != nil || !r.Attack().Equal(want) || !r.Attack().Modifiers().Equal(p.Modifiers) {
				t.Errorf("attack = %+v, want d20's %+v with the supplied modifiers once [FR-002, FR-121]", r.Attack(), want)
			}
		})
	}
}

// Requirement: per case
// Path: per case.
func TestResolveAttackDamage(t *testing.T) {
	t.Parallel()
	fire := combat.MustDiceDamage(1, dice.D4(), content.DamageFire, 0, false)
	extra := combat.MustDiceDamage(2, dice.D6(), content.DamageSlashing, 0, false)
	main := combat.MustDiceDamage(1, dice.D6(), content.DamageSlashing, 0, true)
	cases := []struct {
		name, requirement, path string
		spec                    combat.DamageSpec
		auto                    combat.AutomaticCritical
		faces                   []int
		requests                []string
		kind                    combat.HitKind
		want                    []combat.Damage
	}{
		{"1d8 plus the ability modifier on a hit", `FR-051, FR-053, FR-060; "Damage Rolls", p. 16`, "happy", slashing(1, 8), 0, []int{15, 5}, []string{"1d20", "1d8"}, combat.NormalHit, []combat.Damage{combat.MustDamage(8, content.DamageSlashing)}},
		{"Critical Hit doubles the dice, not the modifier", `FR-056, FR-058; "Critical Hits", p. 16`, "happy", slashing(1, 8), 0, []int{20, 5, 6}, []string{"1d20", "2d8"}, combat.CriticalHit, []combat.Damage{combat.MustDamage(14, content.DamageSlashing)}},
		{"automatic Critical Hit doubles the dice", `FR-090, FR-056`, "unhappy (rule interaction)", slashing(1, 8), combat.AutomaticCriticalAutomatic, []int{12, 5, 6}, []string{"1d20", "2d8"}, combat.CriticalHit, []combat.Damage{combat.MustDamage(14, content.DamageSlashing)}},
		{"extra dice are doubled too", `FR-057`, "unhappy (rule interaction)", combat.MustDamageSpec(main, extra), 0, []int{20, 1, 2, 3, 4, 5, 6}, []string{"1d20", "2d6", "4d6"}, combat.CriticalHit,
			[]combat.Damage{combat.MustDamage(3+3, content.DamageSlashing), combat.MustDamage(18, content.DamageSlashing)}},
		{"two Damage Types, the flag on the first", `FR-061, FR-096`, "happy", combat.MustDamageSpec(main, fire), 0, []int{15, 3, 2}, []string{"1d20", "1d6", "1d4"}, combat.NormalHit,
			[]combat.Damage{combat.MustDamage(6, content.DamageSlashing), combat.MustDamage(2, content.DamageFire)}},
		{"a fixed component draws no die", `FR-054, FR-059`, "unhappy (rule interaction)", combat.MustDamageSpec(combat.MustFixedDamage(1, content.DamageBludgeoning, 0, false)), 0, []int{20}, []string{"1d20"}, combat.CriticalHit,
			[]combat.Damage{combat.MustDamage(1, content.DamageBludgeoning)}},
		{"no DamageSpec on a hit", `FR-062; "Making an Attack", step 3, p. 15`, "unhappy (empty and minimal case)", combat.DamageSpec{}, 0, []int{15}, []string{"1d20"}, combat.NormalHit, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := meleeParams()
			p.Damage, p.AutomaticCritical = c.spec, c.auto
			res, reqs := runAttack(t, attackCommand(t, p), c.faces...)
			r := res.(combat.AttackResolved)
			if !reflect.DeepEqual(reqs, c.requests) || r.HitKind() != c.kind || !slices.Equal(r.Damage(), c.want) || !r.Hit() {
				t.Errorf("requests %v kind %v damage %v, want %v %v %v [%s]", reqs, r.HitKind(), r.Damage(), c.requests, c.kind, c.want, c.requirement)
			}
			if r.Delivery() != combat.AttackDeliveryMelee {
				t.Errorf("Delivery = %v, want melee [FR-111]", r.Delivery())
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-052, FR-091, FR-036 - a miss and a forced miss roll no damage dice and report no
// damage; the d20 is still reported for the forced miss.
// Path: per case.
func TestResolveAttackMissRollsNoDamage(t *testing.T) {
	t.Parallel()
	for name, edit := range map[string]func(*combat.ResolveAttackParams){
		"a miss":        func(*combat.ResolveAttackParams) {},
		"a forced miss": func(p *combat.ResolveAttackParams) { p.Situation.Visibility = combat.TargetUnseen{} },
	} {
		p := meleeParams()
		p.Damage = slashing(1, 8)
		edit(&p)
		faces := []int{2}
		if name == "a forced miss" {
			faces = []int{20, 20}
		}
		res, reqs := runAttack(t, attackCommand(t, p), faces...)
		r := res.(combat.AttackResolved)
		if r.Hit() || len(r.Damage()) != 0 || len(reqs) != 1 || r.Attack().Roll().Natural() != faces[0] {
			t.Errorf("%s: hit %v damage %v requests %v, want a miss with the d20 reported and no damage dice", name, r.Hit(), r.Damage(), reqs)
		}
	}
}

// Requirement: per case
// Path: per case.
func TestResolveAttackRefusalsDrawNoDice(t *testing.T) {
	t.Parallel()
	total := []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	cases := []struct {
		name, requirement, path string
		p                       combat.ResolveAttackParams
		want                    combat.ResolveAttackResult
	}{
		{"Total Cover", `FR-019; ` + srdCover, "unhappy (state error)", func() combat.ResolveAttackParams { p := meleeParams(); p.Cover = total; return p }(), combat.ResolveAttackResult(nil)},
		{"beyond reach", "FR-031", "unhappy (boundary)", func() combat.ResolveAttackParams { p := meleeParams(); p.Distance = 6; return p }(), nil},
		{"beyond long range", "FR-025", "unhappy (boundary)", func() combat.ResolveAttackParams { p := rangedParams(); p.Distance = 321; return p }(), nil},
		{"Total Cover and automatic Critical Hit", "FR-090, FR-019", "unhappy (state error)", func() combat.ResolveAttackParams {
			p := meleeParams()
			p.Cover, p.AutomaticCritical = total, combat.AutomaticCriticalAutomatic
			return p
		}(), nil},
		{"underwater ranged beyond normal range", "FR-041", "unhappy (rule interaction)", func() combat.ResolveAttackParams {
			p := rangedParams()
			p.Distance, p.Situation.Underwater, p.Damage = 81, true, slashing(1, 8)
			return p
		}(), nil},
	}
	wantReasons := []combat.AttackNotPossibleReason{combat.AttackNotPossibleReasonTotalCover, combat.AttackNotPossibleReasonBeyondReach,
		combat.AttackNotPossibleReasonBeyondRange, combat.AttackNotPossibleReasonTotalCover, 0}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			res, reqs := runAttack(t, attackCommand(t, c.p))
			if len(reqs) != 0 {
				t.Errorf("requests = %v, want no dice [%s]", reqs, c.requirement)
			}
			switch r := res.(type) {
			case combat.AttackNotPossible:
				if r.Reason() != wantReasons[i] {
					t.Errorf("reason = %v, want %v [%s]", r.Reason(), wantReasons[i], c.requirement)
				}
			case combat.AttackAutomaticallyMissed:
				if wantReasons[i] != 0 || r.Reason() != combat.AutomaticMissReasonUnderwaterBeyondNormalRange {
					t.Errorf("automatic miss = %+v [%s]", r, c.requirement)
				}
			default:
				t.Errorf("result = %#v, want a refusal or an automatic miss [%s]", res, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/attacks.md FR-037, FR-041 - a hidden attacker's location is given away by a hit, a miss and an
// automatic miss alike.
// Path: per case.
func TestResolveAttackRevealsAHiddenAttacker(t *testing.T) {
	t.Parallel()
	for _, face := range []int{15, 2} {
		p := meleeParams()
		p.Situation.AttackerHidden = true
		res, _ := runAttack(t, attackCommand(t, p), face)
		if !res.(combat.AttackResolved).RevealsAttackerLocation() {
			t.Errorf("a d20 of %d did not reveal the attacker", face)
		}
	}
	p := rangedParams()
	p.Distance, p.Situation = 81, combat.AttackSituation{Underwater: true, AttackerHidden: true}
	if res, _ := runAttack(t, attackCommand(t, p)); !res.(combat.AttackAutomaticallyMissed).RevealsAttackerLocation() {
		t.Errorf("an automatic miss did not reveal the attacker")
	}
	if res, _ := runAttack(t, attackCommand(t, meleeParams()), 15); res.(combat.AttackResolved).RevealsAttackerLocation() {
		t.Errorf("an attacker that is not hidden was revealed")
	}
}

// Requirement: doc/specs/attacks.md FR-043, FR-044, FR-024, FR-027 - the circumstances the facts give and the caller's are
// combined by the normal rule: long range plus close combat is one Disadvantage; long range plus the caller's Advantage
// cancels to one d20.
// Path: per case.
func TestResolveAttackCombinesCircumstances(t *testing.T) {
	t.Parallel()
	p := rangedParams()
	p.Distance = 100
	p.NearbyEnemies = []combat.NearbyEnemy{combat.MustNearbyEnemy(5, true, false)}
	res, reqs := runAttack(t, attackCommand(t, p), 12, 4)
	r := res.(combat.AttackResolved)
	if !reflect.DeepEqual(reqs, []string{"2d20"}) || r.Attack().Roll().Natural() != 4 || r.Attack().Roll().Mode() != modifier.WithDisadvantage ||
		len(r.Circumstances()) != 2 {
		t.Errorf("long range plus close combat: requests %v natural %d, want 2d20 and the lower die", reqs, r.Attack().Roll().Natural())
	}
	p = rangedParams()
	p.Distance, p.Circumstances = 100, []modifier.RollCircumstance{advantage}
	res, reqs = runAttack(t, attackCommand(t, p), 12)
	if r := res.(combat.AttackResolved); !reflect.DeepEqual(reqs, []string{"1d20"}) || r.Attack().Roll().Mode() != modifier.Normal {
		t.Errorf("long range plus Advantage: requests %v mode %v, want one d20 in the normal mode", reqs, r.Attack().Roll().Mode())
	}
}

// Requirement: doc/specs/attacks.md FR-079, FR-080, FR-082 - the same command and dice give the same result; a command is
// not altered; slices in a result are copies; all-minimum and all-maximum rolls equal the table cases.
// Path: unhappy (determinism, immutability, boundary).
func TestResolveAttackDeterminismImmutabilityExtremes(t *testing.T) {
	t.Parallel()
	p := meleeParams()
	p.Damage = slashing(1, 8)
	p.Circumstances = []modifier.RollCircumstance{advantage}
	cmd := attackCommand(t, p)
	before := attackCommand(t, p)
	a, _ := runAttack(t, cmd, 14, 3, 6)
	b, _ := runAttack(t, cmd, 14, 3, 6)
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(cmd, before) {
		t.Errorf("results differ or the command changed: %v vs %v", a, b)
	}
	r := a.(combat.AttackResolved)
	r.Damage()[0] = combat.MustDamage(99, content.DamageFire)
	r.Circumstances()[0] = disadvantage
	if again := a.(combat.AttackResolved); again.Damage()[0].Amount() != 9 || again.Circumstances()[0] != advantage {
		t.Errorf("a returned slice reached the result: %v %v", again.Damage(), again.Circumstances())
	}
	uc, _ := combat.NewResolveAttack(dicetest.NewMinimumRoller())
	lo, err := uc.Run(context.Background(), cmd)
	uc, _ = combat.NewResolveAttack(dicetest.NewMaximumRoller())
	hi, err2 := uc.Run(context.Background(), cmd)
	if err != nil || err2 != nil {
		t.Fatal(err, err2)
	}
	l, h := lo.(combat.AttackResolved), hi.(combat.AttackResolved)
	// Minimum: natural 1 misses; maximum: natural 20 is a Critical Hit with 2d8 of 8 plus the modifier 3.
	if l.Hit() || l.Attack().Roll().Natural() != 1 || !h.CriticalHit() || h.Damage()[0].Amount() != 19 {
		t.Errorf("minimum %+v and maximum %+v rolls, want a miss and a Critical Hit of 19", l, h)
	}
}

// Requirement: doc/specs/attacks.md FR-081, FR-085, FR-112 - a roller that fails, panics or answers another expression, in
// the attack roll or in the damage roll, is a *srderr.PortError (dice_roller_error) with no result, recorded once on the
// span and once in srd5e.combat.errors, with no operations increment, no histogram sample and no log.
// Path: per case.
func TestResolveAttackPortFailure(t *testing.T) {
	t.Parallel()
	cause := errors.New("boom")
	p := meleeParams()
	p.Damage = slashing(1, 8)
	for _, stage := range []string{"attack roll", "damage roll"} {
		rollers := map[string]func(good func(dice.Expression) (dice.Result, error)) combat.Roller{
			"fails": func(good func(dice.Expression) (dice.Result, error)) combat.Roller {
				return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
					if e.Die() == dice.D20() == (stage == "attack roll") {
						return dice.Result{}, cause
					}
					return good(e)
				})
			},
			"panics": func(good func(dice.Expression) (dice.Result, error)) combat.Roller {
				return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
					if e.Die() == dice.D20() == (stage == "attack roll") {
						panic(cause)
					}
					return good(e)
				})
			},
			"answers another expression": func(good func(dice.Expression) (dice.Result, error)) combat.Roller {
				return dicetest.CallbackRoller(func(e dice.Expression) (dice.Result, error) {
					if e.Die() == dice.D20() == (stage == "attack roll") {
						return dice.NewResult(dice.MustParseExpression("1d3"), []dice.DieRoll{dice.MustDieRoll(dice.D3(), 2)})
					}
					return good(e)
				})
			},
		}
		for name, mk := range rollers {
			t.Run(stage+" "+name, func(t *testing.T) {
				t.Parallel()
				good := func(e dice.Expression) (dice.Result, error) {
					rolls := make([]dice.DieRoll, e.Count())
					for i := range rolls {
						rolls[i] = dice.MustDieRoll(e.Die(), min(e.Die().Sides(), 15))
					}
					return dice.NewResult(e, rolls)
				}
				rec, opts := recording()
				uc, _ := combat.NewResolveAttack(mk(good), opts...)
				res, err := uc.Run(context.Background(), attackCommand(t, p))
				var pe *srderr.PortError
				if res != nil || !errors.As(err, &pe) || pe.Port != dice.Port {
					t.Fatalf("= %v, %v, want nil and a dice_roller *PortError [FR-081]", res, err)
				}
				switch name {
				case "fails":
					if !errors.Is(err, cause) {
						t.Errorf("the cause is not reachable: %v", err)
					}
				case "panics":
					if !errors.Is(err, srderr.ErrPortPanic) {
						t.Errorf("a panic is not an ErrPortPanic: %v", err)
					}
				default:
					if !errors.Is(err, srderr.ErrContractViolation) {
						t.Errorf("a wrong answer is not an ErrContractViolation: %v [FR-112]", err)
					}
				}
				spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_attack")
				if len(spans) != 1 || spans[0].Status != telemetry.StatusError || len(spans[0].Errors) != 1 || !spans[0].Ended {
					t.Errorf("spans = %+v, want one ended span with one recorded error [FR-085]", spans)
				}
				errs := measured(rec, "srd5e.combat.errors")
				if len(errs) != 1 || errs[0]["error.type"] != srderr.ErrorType(err) || errs[0]["srd5e.combat.operation"] != "resolve_attack" {
					t.Errorf("errors = %v, want exactly one with error.type [FR-085]", errs)
				}
				for _, n := range []string{"srd5e.combat.operations", "srd5e.combat.attack.modifiers"} {
					if m := rec.Meter.MeasurementsNamed(n); len(m) != 0 {
						t.Errorf("%s recorded %v on a failure [FR-130]", n, m)
					}
				}
				if logs := rec.Logger.Records(); len(logs) != 0 {
					t.Errorf("logged %v on a failure [FR-131]", logs)
				}
				if v := rec.Violations(); len(v) != 0 {
					t.Errorf("telemetry misuse: %v", v)
				}
			})
		}
	}
}

// Requirement: doc/specs/attacks.md FR-115, testing.md §4 - a nil roller and the zero use case or command are validation
// errors, and a cancelled ctx comes back as ctx.Err() with no signal and no die.
// Path: unhappy (invalid input, cancellation).
func TestResolveAttackInvalidUse(t *testing.T) {
	t.Parallel()
	if _, err := combat.NewResolveAttack(nil); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("NewResolveAttack(nil) error = %v, want ErrInvalidInput [FR-115]", err)
	}
	if _, err := (combat.ResolveAttack{}).Run(context.Background(), attackCommand(t, meleeParams())); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero use case error = %v, want ErrInvalidInput", err)
	}
	uc, _ := combat.NewResolveAttack(dicetest.NewFixedRoller(10))
	if _, err := uc.Run(context.Background(), combat.ResolveAttackCommand{}); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("zero command error = %v, want ErrInvalidInput", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	roller := dicetest.NewFixedRoller(10)
	rec, opts := recording()
	uc, _ = combat.NewResolveAttack(roller, opts...)
	if res, err := uc.Run(ctx, attackCommand(t, meleeParams())); res != nil || !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled ctx = %v, %v, want nil and context.Canceled", res, err)
	}
	if n := len(rec.Tracer.Spans()) + len(rec.Meter.Measurements()) + len(roller.Requests()); n != 0 {
		t.Errorf("%d signals or dice on a cancelled ctx, want none", n)
	}
}

// Requirement: doc/specs/attacks.md FR-083, FR-084, FR-086, FR-110, FR-116, FR-117, FR-118, FR-122, FR-130, FR-131 - the
// signals of ResolveAttack: the shared span and operations counter with the extras, class and result for every result
// value, the reason only for not_possible, roll_mode and critical_hit only when a d20 was rolled, one modifiers sample per
// returned result, one debug log, no d20 signal, and a result identical with and without telemetry.
// Path: per case.
func TestResolveAttackTelemetry(t *testing.T) {
	t.Parallel()
	total := []combat.CoverSource{{Cover: combat.TotalCover, AttackOriginatesOppositeSide: true}}
	half := []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	cases := []struct {
		name, requirement, path string
		edit                    func(*combat.ResolveAttackParams)
		faces                   []int
		result, class, outcome  string
		form                    string
		cover, reason           string
		mode                    string
		critical                any
		modifiers               float64
		auto                    bool
	}{
		{"hit", "FR-122", "happy", func(p *combat.ResolveAttackParams) { p.Cover = half }, []int{12, 3}, "hit", "applied", "success", "melee_weapon", "half", "", "normal", false, 2, false},
		{"critical hit", "FR-122, FR-118", "happy", func(p *combat.ResolveAttackParams) { p.Modifiers = mods(3, 2, 1, 1) }, []int{20, 3, 3}, "critical_hit", "applied", "success", "melee_weapon", "none", "", "normal", true, 4, false},
		{"miss", "FR-122, FR-086", "unhappy (boundary)", func(p *combat.ResolveAttackParams) { p.Circumstances = []modifier.RollCircumstance{disadvantage} }, []int{2, 18}, "miss", "applied", "success", "melee_weapon", "none", "", "disadvantage", false, 2, false},
		{"forced miss reports the natural 20 of the roll", "FR-118, FR-122", "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) {
			p.Situation.Visibility = combat.TargetUnseen{}
			p.AutomaticCritical = combat.AutomaticCriticalAutomatic
		}, []int{20, 20}, "miss", "applied", "success", "melee_weapon", "none", "", "disadvantage", true, 2, true},
		{"automatic Critical Hit is not the roll's", "FR-118", "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) { p.AutomaticCritical = combat.AutomaticCriticalAutomatic }, []int{12, 4, 4}, "critical_hit", "applied", "success", "melee_weapon", "none", "", "normal", false, 2, true},
		{"automatic miss", "FR-122, FR-118", "unhappy (rule interaction)", func(p *combat.ResolveAttackParams) {
			p.Form, p.Range, p.Distance, p.Situation.Underwater = combat.RangedWeaponAttack, combat.MustAttackRange(80, 320), 81, true
		}, nil, "automatic_miss", "applied", "success", "ranged_weapon", "none", "", "", nil, 2, false},
		{"not possible", "FR-086, FR-116, FR-117", "unhappy (state error)", func(p *combat.ResolveAttackParams) { p.Cover = total }, nil, "not_possible", "refused", "failure", "melee_weapon", "total", "total_cover", "", nil, 2, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := meleeParams()
			p.Damage = slashing(1, 4)
			c.edit(&p)
			cmd := attackCommand(t, p)
			rec, opts := recording()
			uc, _ := combat.NewResolveAttack(dicetest.NewFixedRoller(c.faces...), opts...)
			res, err := uc.Run(context.Background(), cmd)
			if err != nil {
				t.Fatal(err)
			}
			plain, _ := runAttack(t, cmd, c.faces...)
			if !reflect.DeepEqual(res, plain) {
				t.Errorf("a recording bundle changed the result [FR-083]")
			}
			ops := measured(rec, "srd5e.combat.operations")
			if len(ops) != 1 {
				t.Fatalf("operations = %v, want one [FR-110]", ops)
			}
			got := ops[0]
			want := map[string]any{"srd5e.combat.operation": "resolve_attack", "srd5e.outcome": c.outcome, "srd5e.combat.result": c.result,
				"srd5e.combat.result_class": c.class, "srd5e.combat.attack.form": c.form, "srd5e.combat.attack.cover": c.cover,
				"srd5e.combat.attack.automatic_critical": c.auto}
			if c.reason != "" {
				want["srd5e.combat.attack.reason"] = c.reason
			}
			if c.mode != "" {
				want["srd5e.d20_test.roll_mode"], want["srd5e.d20_test.critical_hit"] = c.mode, c.critical
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("operations attributes = %v, want %v [%s]", got, want, c.requirement)
			}
			spans := rec.Tracer.SpansNamed("srd5e.combat.resolve_attack")
			if len(spans) != 1 || spans[0].Status != telemetry.StatusUnset || !spans[0].Ended {
				t.Errorf("spans = %+v, want one ended span without error status (a refusal is not an error) [FR-086]", spans)
			} else if v, _ := spans[0].AttrValue("srd5e.combat.result"); v != c.result {
				t.Errorf("span result = %v, want %s", v, c.result)
			}
			if h := rec.Meter.MeasurementsNamed("srd5e.combat.attack.modifiers"); len(h) != 1 || h[0].Value != c.modifiers || h[0].Unit != "{modifier}" ||
				attrMap(h[0].Attrs)["srd5e.combat.attack.form"] != c.form || len(h[0].Attrs) != 1 {
				t.Errorf("modifiers histogram = %+v, want one sample of %v with the form [FR-130]", h, c.modifiers)
			}
			if d := rec.Meter.MeasurementsNamed("srd5e.combat.operation_duration"); len(d) != 1 {
				t.Errorf("operation_duration = %v, want one", d)
			}
			if e := rec.Meter.MeasurementsNamed("srd5e.combat.events"); len(e) != 0 {
				t.Errorf("events = %v, want none (no domain events)", e)
			}
			logs := rec.Logger.Records()
			if len(logs) != 1 || logs[0].Level != telemetry.LevelDebug || logs[0].Message != "attack resolved" ||
				!reflect.DeepEqual(attrMap(logs[0].Attrs), map[string]any{"srd5e.combat.result": c.result}) {
				t.Errorf("logs = %+v, want one debug 'attack resolved' with the result [FR-131]", logs)
			}
			checkNoD20Signals(t, rec)
		})
	}
}

// checkNoD20Signals asserts the P12 rule and the hygiene of the recording double: no srd5e.d20_test.* metric or span
// name, and no recorded misuse.
func checkNoD20Signals(t *testing.T, rec *telemetrytest.Recorder) {
	t.Helper()
	for _, m := range rec.Meter.Measurements() {
		if strings.HasPrefix(m.Name, "srd5e.d20_test.") {
			t.Errorf("a combat call recorded the d20 signal %s (plan P12, FR-110)", m.Name)
		}
	}
	for _, s := range rec.Tracer.Spans() {
		if strings.HasPrefix(s.Name, "srd5e.d20_test.") {
			t.Errorf("a combat call started the d20 span %s (plan P12, FR-110)", s.Name)
		}
	}
	if v := rec.Violations(); len(v) != 0 {
		t.Errorf("telemetry misuse: %v", v)
	}
}

// Requirement: doc/specs/attacks.md FR-009 - the lowest and highest Strength/Proficiency sums stay exact at the limits of
// int: the total of an attack with an extreme modifier is the modifier plus the die, and damage saturates.
// Path: unhappy (boundary).
func TestResolveAttackHugeNumbers(t *testing.T) {
	t.Parallel()
	p := meleeParams()
	p.ArmorClass = math.MaxInt
	p.Modifiers = mods(0, 0)
	p.Damage = combat.MustDamageSpec(combat.MustFixedDamage(math.MaxInt, content.DamageForce, math.MaxInt, false))
	res, _ := runAttack(t, attackCommand(t, p), 20)
	r := res.(combat.AttackResolved)
	if !r.CriticalHit() || r.Damage()[0].Amount() != math.MaxInt {
		t.Errorf("= %+v, want a Critical Hit of math.MaxInt damage, not a wrapped sum", r)
	}
	p.Cover = []combat.CoverSource{{Cover: combat.HalfCover, AttackOriginatesOppositeSide: true}}
	res, _ = runAttack(t, attackCommand(t, p), 20)
	if got := res.(combat.AttackResolved).EffectiveArmorClass(); got != math.MaxInt {
		t.Errorf("effective AC = %d, want it saturated at math.MaxInt", got)
	}
}
