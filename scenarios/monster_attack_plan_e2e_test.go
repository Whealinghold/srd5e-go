package scenarios_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// grpFAttack builds an Attack Roll notation of a kind, a bonus and a qualifier, with an optional reach and range.
func grpFAttack(kind content.StatBlockAttackKind, bonus int, qualifier string, reach int, rng content.StatBlockAttackRange) content.StatBlockAttack {
	return grpCMust(content.NewStatBlockAttack(content.StatBlockAttackFields{
		Kind: kind, Bonus: bonus, Qualifier: qualifier, Reach: reach, HasReach: reach > 0, Range: rng}))
}

// grpFAttackPart is an attack part with the clauses.
func grpFAttackPart(name string, atk content.StatBlockAttack, clauses ...content.EffectClause) content.StatBlockPart {
	return grpCPart(name, content.StatBlockPartFields{Attack: atk, Clauses: clauses})
}

// grpFClaw is the Claw of the scenarios: Melee Attack Roll +5, reach 10 ft., a Hit of 7 (1d8 + 3) Slashing and 4 (1d6 + 1) Poison, a Miss that prints Poison damage, and a Hit or Miss clause.
func grpFClaw() content.StatBlockPart {
	slash := grpCDamage(7, dice.MustExpression(1, dice.D8(), 3), content.DamageSlashing)
	poison := grpCDamage(4, dice.MustExpression(1, dice.D6(), 1), content.DamagePoison)
	return grpFAttackPart("Claw", grpFAttack(content.AttackMelee, 5, "(Advantage while the target is Prone)", 10, content.StatBlockAttackRange{}),
		grpFClause(content.ClauseHit, nil, slash, poison),
		grpFClause(content.ClauseMiss, nil, grpCDamage(2, dice.Expression{}, content.DamagePoison)),
		grpFClause(content.ClauseHitOrMiss, nil))
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-022, FR-023, FR-024, FR-025, FR-030 - a melee stat-block attack plans MeleeWeaponAttack with the printed reach and the printed bonus as the whole roll modifier (a negative and a zero bonus are valid, nothing is added), the qualifier is exposed verbatim, a ranged one plans RangedWeaponAttack with a single or a normal and long range, and the damage of the plan is the Hit clause's alone (absent when it prints none, never the Miss clause's).
// Path: happy.
func TestGrpFAttackPlanMapsTheNotation(t *testing.T) {
	t.Parallel()
	all := grpCMust(combat.AllDamageNotation(combat.DamageNotationDice))

	melee := grpCMust(combat.PlanStatBlockAttack(grpFClaw(), all))
	if reach, ok := melee.Reach(); melee.Form() != combat.MeleeWeaponAttack || !ok || reach.Feet() != 10 {
		t.Errorf("Claw plan: form %v reach %v %v, want melee with 10 feet", melee.Form(), reach.Feet(), ok)
	}
	if _, ok := melee.Range(); ok {
		t.Errorf("a melee plan has a range")
	}
	if want := d20.NewRollModifiers(modifier.New(5)); !reflect.DeepEqual(melee.Modifiers(), want) {
		t.Errorf("Claw modifiers = %+v, want the printed +5 alone %+v", melee.Modifiers(), want)
	}
	if melee.Qualifier() != "(Advantage while the target is Prone)" {
		t.Errorf("qualifier = %q, want it verbatim", melee.Qualifier())
	}
	spec, hasDamage := melee.Damage()
	if !hasDamage || len(spec.Components()) != 2 {
		t.Fatalf("Claw damage = %v %v, want the two instances of the Hit clause and none of the Miss clause", spec.Components(), hasDamage)
	}

	long := grpCMust(content.NewStatBlockAttackRange(80, 320))
	bow := grpFAttackPart("Bow", grpFAttack(content.AttackRanged, -1, "", 0, long), grpFClause(content.ClauseHit, nil, grpCDamage(5, dice.MustExpression(1, dice.D6(), 2), content.DamagePiercing)))
	ranged := grpCMust(combat.PlanStatBlockAttack(bow, all))
	if rng, ok := ranged.Range(); ranged.Form() != combat.RangedWeaponAttack || !ok || rng.Normal() != 80 || rng.Long() != 320 {
		t.Errorf("Bow plan: form %v range %v/%v %v, want ranged 80/320", ranged.Form(), rng.Normal(), rng.Long(), ok)
	}
	if _, ok := ranged.Reach(); ok {
		t.Errorf("a ranged plan has a reach")
	}
	if want := d20.NewRollModifiers(modifier.New(-1)); !reflect.DeepEqual(ranged.Modifiers(), want) {
		t.Errorf("Bow modifiers = %+v, want the printed -1 alone", ranged.Modifiers())
	}
	single := grpFAttackPart("Spit", grpFAttack(content.AttackRanged, 0, "", 0, grpCMust(content.NewStatBlockAttackRange(20, 0))))
	spit := grpCMust(combat.PlanStatBlockAttack(single, combat.DamageNotationChoices{}))
	if rng, ok := spit.Range(); !ok || rng.Normal() != 20 || rng.Long() != 0 {
		t.Errorf("Spit range = %v/%v %v, want a single range of 20", rng.Normal(), rng.Long(), ok)
	}
	if want := d20.NewRollModifiers(modifier.New(0)); !reflect.DeepEqual(spit.Modifiers(), want) {
		t.Errorf("a +0 bonus planned %+v, want a plain zero modifier", spit.Modifiers())
	}
	if _, has := spit.Damage(); has {
		t.Errorf("a Hit clause with no damage planned a damage spec")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-024, FR-025, FR-053 - a "Melee or Ranged" attack must be planned with an explicit delivery and never defaults, a melee-only or ranged-only attack takes its own delivery when none is given and refuses a different one, choosing melee on an attack with no reach or ranged on one with no range is invalid input, and a part with no attack notation is invalid input.
// Path: unhappy (invalid input).
func TestGrpFAttackPlanDeliveryIsChosenNotGuessed(t *testing.T) {
	t.Parallel()
	none := combat.DamageNotationChoices{}
	either := grpFAttackPart("Hurl", grpFAttack(content.AttackMeleeOrRanged, 6, "", 5, grpCMust(content.NewStatBlockAttackRange(30, 60))))
	if _, err := combat.PlanStatBlockAttack(either, none); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a Melee or Ranged attack with no delivery error = %v, want ErrInvalidInput", err)
	}
	if p := grpCMust(combat.PlanStatBlockAttack(either, none, combat.WithDelivery(combat.AttackDeliveryMelee))); p.Form() != combat.MeleeWeaponAttack {
		t.Errorf("Hurl as melee = %v", p.Form())
	} else if reach, _ := p.Reach(); reach.Feet() != 5 {
		t.Errorf("Hurl as melee reach = %d, want 5", reach.Feet())
	}
	if p := grpCMust(combat.PlanStatBlockAttack(either, none, combat.WithDelivery(combat.AttackDeliveryRanged))); p.Form() != combat.RangedWeaponAttack {
		t.Errorf("Hurl as ranged = %v", p.Form())
	} else if rng, _ := p.Range(); rng.Normal() != 30 || rng.Long() != 60 {
		t.Errorf("Hurl as ranged range = %d/%d, want 30/60", rng.Normal(), rng.Long())
	}
	if _, err := combat.PlanStatBlockAttack(either, none, combat.WithDelivery(combat.AttackDelivery(9))); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined delivery error = %v, want ErrInvalidInput", err)
	}

	// A melee-only attack takes melee without being told, agrees with melee, and refuses ranged.
	claw := grpFClaw()
	if p := grpCMust(combat.PlanStatBlockAttack(claw, grpCMust(combat.AllDamageNotation(combat.DamageNotationNumber)), combat.WithDelivery(combat.AttackDeliveryMelee))); p.Form() != combat.MeleeWeaponAttack {
		t.Errorf("Claw with the matching delivery = %v", p.Form())
	}
	if _, err := combat.PlanStatBlockAttack(claw, grpCMust(combat.AllDamageNotation(combat.DamageNotationNumber)), combat.WithDelivery(combat.AttackDeliveryRanged)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a ranged delivery on a melee-only attack error = %v, want ErrInvalidInput", err)
	}

	// No reach to plan a melee attack with; no range to plan a ranged one with.
	noReach := grpFAttackPart("Slap", grpFAttack(content.AttackMelee, 3, "", 0, content.StatBlockAttackRange{}))
	if _, err := combat.PlanStatBlockAttack(noReach, none); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a melee attack with no printed reach error = %v, want ErrInvalidInput", err)
	}
	noRange := grpFAttackPart("Sling", grpFAttack(content.AttackRanged, 3, "", 0, content.StatBlockAttackRange{}))
	if _, err := combat.PlanStatBlockAttack(noRange, none); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("a ranged attack with no printed range error = %v, want ErrInvalidInput", err)
	}
	reachOnly := grpFAttackPart("Hand", grpFAttack(content.AttackMeleeOrRanged, 3, "", 5, content.StatBlockAttackRange{}))
	if _, err := combat.PlanStatBlockAttack(reachOnly, none, combat.WithDelivery(combat.AttackDeliveryRanged)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("ranged on an attack with a reach only error = %v, want ErrInvalidInput", err)
	}

	// A part with no attack notation (a trait) is invalid input for the planner and for the applied clauses.
	trait := grpCUsage("Keen Smell", content.PartUnlimited{})
	if _, err := combat.PlanStatBlockAttack(trait, none); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("planning a part with no attack error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.AppliedClausesForAttack(trait, true, true); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("clauses of a part with no attack error = %v, want ErrInvalidInput", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-026, FR-027, FR-028, FR-030 - each damage instance is rolled by exactly one of the two notations chosen before any roll: number gives the printed number as fixed damage, dice gives the printed dice with their flat modifier as dice damage, neither adds the attack's ability modifier, the choices come one per instance in printed order or one for all, and a list of another length, no choice, an undefined choice, dice for an instance printed with no die, or choices for a Hit that prints no damage are invalid input.
// Path: per case.
func TestGrpFAttackPlanDamageNotation(t *testing.T) {
	t.Parallel()
	number, diceChoice := combat.DamageNotationNumber, combat.DamageNotationDice
	claw := grpFClaw()
	plain := grpFAttackPart("Nip", grpFAttack(content.AttackMelee, 2, "", 5, content.StatBlockAttackRange{}),
		grpFClause(content.ClauseHit, nil, grpCDamage(1, dice.Expression{}, content.DamageSlashing)))
	hitless := grpFAttackPart("Glare", grpFAttack(content.AttackMelee, 2, "", 5, content.StatBlockAttackRange{}), grpFClause(content.ClauseHit, nil))
	described := grpCMust(content.NewStatBlockDamage(3, dice.Expression{}, grpCMust(content.NewDescribedDamageType("the chosen type")), false))
	textType := grpFAttackPart("Chosen", grpFAttack(content.AttackMelee, 2, "", 5, content.StatBlockAttackRange{}), grpFClause(content.ClauseHit, nil, described))

	for _, c := range []struct {
		name    string
		part    content.StatBlockPart
		choices combat.DamageNotationChoices
		want    []string // "fixed amount" or "dice count d sides + modifier" per component, empty when an error is expected
	}{
		{"number for both", claw, grpCMust(combat.NewDamageNotationChoices(number, number)), []string{"fixed 7 slashing", "fixed 4 poison"}},
		{"dice for both", claw, grpCMust(combat.NewDamageNotationChoices(diceChoice, diceChoice)), []string{"dice 1d8+3 slashing", "dice 1d6+1 poison"}},
		{"number then dice", claw, grpCMust(combat.NewDamageNotationChoices(number, diceChoice)), []string{"fixed 7 slashing", "dice 1d6+1 poison"}},
		{"dice then number", claw, grpCMust(combat.NewDamageNotationChoices(diceChoice, number)), []string{"dice 1d8+3 slashing", "fixed 4 poison"}},
		{"one choice for all, number", claw, grpCMust(combat.AllDamageNotation(number)), []string{"fixed 7 slashing", "fixed 4 poison"}},
		{"one choice for all, dice", claw, grpCMust(combat.AllDamageNotation(diceChoice)), []string{"dice 1d8+3 slashing", "dice 1d6+1 poison"}},
		{"a printed number with no die as a number", plain, grpCMust(combat.AllDamageNotation(number)), []string{"fixed 1 slashing"}},
	} {
		spec, has := grpCMust(combat.PlanStatBlockAttack(c.part, c.choices)).Damage()
		if !has {
			t.Errorf("%s: no damage planned", c.name)
			continue
		}
		var got []string
		for _, comp := range spec.Components() {
			if comp.AddsAbilityModifier() {
				t.Errorf("%s: a component adds the ability modifier, which the printed numbers already hold", c.name)
			}
			switch d := comp.(type) {
			case combat.FixedDamage:
				got = append(got, "fixed "+strconv.Itoa(d.Amount())+" "+strings.ToLower(d.Type().String()))
				if d.Modifier() != 0 {
					t.Errorf("%s: a fixed component carries a modifier of %d", c.name, d.Modifier())
				}
			case combat.DiceDamage:
				got = append(got, "dice "+strconv.Itoa(d.Count())+"d"+strconv.Itoa(d.Die().Sides())+"+"+strconv.Itoa(d.Modifier())+" "+strings.ToLower(d.Type().String()))
			default:
				t.Fatalf("%s: component %T is neither fixed nor dice", c.name, comp)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: components %v, want %v", c.name, got, c.want)
		}
	}

	for _, c := range []struct {
		name    string
		part    content.StatBlockPart
		choices combat.DamageNotationChoices
	}{
		{"too few choices", claw, grpCMust(combat.NewDamageNotationChoices(number))},
		{"too many choices", claw, grpCMust(combat.NewDamageNotationChoices(number, number, number))},
		{"dice for an instance with no die", plain, grpCMust(combat.AllDamageNotation(diceChoice))},
		{"a per-instance choice for a Hit that prints no damage", hitless, grpCMust(combat.NewDamageNotationChoices(number))},
		{"a damage type printed as text", textType, grpCMust(combat.AllDamageNotation(number))},
	} {
		if _, err := combat.PlanStatBlockAttack(c.part, c.choices); !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s: error = %v, want ErrInvalidInput", c.name, err)
		}
	}
	if p := grpCMust(combat.PlanStatBlockAttack(hitless, grpCMust(combat.AllDamageNotation(number)))); !func() bool { _, has := p.Damage(); return !has }() {
		t.Errorf("a choice for all instances planned damage for a Hit that prints none")
	}
	if _, err := combat.NewDamageNotationChoices(); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("no choice error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.NewDamageNotationChoices(number, combat.DamageNotationChoice(9)); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("an undefined choice error = %v, want ErrInvalidInput", err)
	}
	if _, err := combat.AllDamageNotation(0); !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("the zero choice for all error = %v, want ErrInvalidInput", err)
	}
	if !grpCMust(combat.AllDamageNotation(number)).Equal(grpCMust(combat.AllDamageNotation(number))) ||
		grpCMust(combat.AllDamageNotation(number)).Equal(grpCMust(combat.AllDamageNotation(diceChoice))) {
		t.Errorf("DamageNotationChoices.Equal does not tell equal choices from different ones")
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-031 - the clauses of an attack that was made are listed in printed order: a hit, a Critical Hit included, applies Hit and Hit or Miss; a miss applies Miss and Hit or Miss; Hit and Miss never both apply; an attack that was not possible applies none.
// Path: per case.
func TestGrpFAppliedClausesOfAnAttack(t *testing.T) {
	t.Parallel()
	claw := grpFClaw()
	for _, c := range []struct {
		name     string
		hit      bool
		possible bool
		want     []content.EffectClauseLabel
	}{
		{"a hit", true, true, []content.EffectClauseLabel{content.ClauseHit, content.ClauseHitOrMiss}},
		{"a miss", false, true, []content.EffectClauseLabel{content.ClauseMiss, content.ClauseHitOrMiss}},
		{"a hit that was not possible", true, false, nil},
		{"a miss that was not possible", false, false, nil},
	} {
		got, err := combat.AppliedClausesForAttack(claw, c.hit, c.possible)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		var labels []content.EffectClauseLabel
		for _, cl := range got {
			labels = append(labels, cl.Label())
		}
		if !slices.Equal(labels, c.want) {
			t.Errorf("%s: clauses %v, want %v", c.name, labels, c.want)
		}
	}
	// The damage of the Miss clause is the host's to roll, not part of the planned hit.
	if got, _ := combat.AppliedClausesForAttack(claw, false, true); len(got) != 2 || len(got[0].Damage()) != 1 {
		t.Errorf("the Miss clause lost its printed damage: %v", got)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-022, FR-026, FR-028, FR-029 and doc/specs/attacks.md FR-056 to FR-059 - a planned Claw is accepted by the real attack resolver: the printed +5 is the whole attack modifier and the printed numbers get no ability modifier; a hit deals the fixed 7 and the rolled 1d8 + 3, a Critical Hit doubles the dice and leaves the fixed number as it is, and a target beyond the printed reach is not possible.
// Path: happy, then unhappy (boundary).
func TestGrpFPlannedClawResolvesThroughTheAttackResolver(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	plan := grpCMust(combat.PlanStatBlockAttack(grpFClaw(), grpCMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber, combat.DamageNotationDice))))
	spec, _ := plan.Damage()
	reach, _ := plan.Reach()
	run := func(distance int, rolls ...int) combat.ResolveAttackResult {
		t.Helper()
		cmd, err := combat.NewResolveAttackCommand(combat.ResolveAttackParams{
			Form: plan.Form(), ArmorClass: 12, Modifiers: plan.Modifiers(), Distance: distance, Reach: reach, Damage: spec})
		if err != nil {
			t.Fatal(err)
		}
		uc, err := combat.NewResolveAttack(dicetest.NewFixedRoller(rolls...))
		if err != nil {
			t.Fatal(err)
		}
		res, err := uc.Run(ctx, cmd)
		if err != nil {
			t.Fatal(err)
		}
		return res
	}

	// A 7 on the d20 is 12 with the printed +5: it meets Armor Class 12 and hits. The d8 shows 6.
	hit := run(10, 7, 6).(combat.AttackResolved)
	if !hit.Hit() || hit.CriticalHit() || hit.Attack().Total() != 12 {
		t.Fatalf("hit = %v critical %v total %d, want a plain hit on a total of 12", hit.Hit(), hit.CriticalHit(), hit.Attack().Total())
	}
	if want := []combat.Damage{grpCMust(combat.NewDamage(7, content.DamageSlashing)), grpCMust(combat.NewDamage(6+1, content.DamagePoison))}; !slices.Equal(hit.Damage(), want) {
		t.Errorf("damage = %v, want the fixed 7 Slashing and the d6 (6) plus the printed 1 as Poison, with no ability modifier", hit.Damage())
	}
	// A 6 is 11, one short of the Armor Class: a miss that deals nothing.
	if miss := run(10, 6).(combat.AttackResolved); miss.Hit() || len(miss.Damage()) != 0 {
		t.Errorf("a total of 11 hit or dealt %v", miss.Damage())
	}
	// A natural 20 is a Critical Hit: the fixed 7 stays, the dice double (two d6 instead of one).
	crit := run(10, 20, 2, 3).(combat.AttackResolved)
	if !crit.Hit() || !crit.CriticalHit() {
		t.Fatalf("a natural 20 = hit %v critical %v", crit.Hit(), crit.CriticalHit())
	}
	got := crit.Damage()
	if len(got) != 2 || got[0] != grpCMust(combat.NewDamage(7, content.DamageSlashing)) || got[1] != grpCMust(combat.NewDamage(2+3+1, content.DamagePoison)) {
		t.Errorf("critical damage = %v, want the fixed 7 Slashing and 2d6 + 1 = 6 Poison", got)
	}
	// 15 feet is past the printed reach of 10: not possible, and no die is rolled.
	if _, ok := run(15, 20).(combat.AttackNotPossible); !ok {
		t.Errorf("an attack from 15 feet with a 10-foot reach was possible")
	}
}
