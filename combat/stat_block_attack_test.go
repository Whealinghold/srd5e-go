package combat_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func sbNoRange() content.StatBlockAttackRange { return content.StatBlockAttackRange{} }

func sbNumbers(tb testing.TB) combat.DamageNotationChoices {
	tb.Helper()
	return luMust(combat.AllDamageNotation(combat.DamageNotationNumber))
}

func sbDiceChoices(tb testing.TB) combat.DamageNotationChoices {
	tb.Helper()
	return luMust(combat.AllDamageNotation(combat.DamageNotationDice))
}

// sbSlam is the Animated Armor shape (SRD 5.2.1, p. 259): melee +4, reach 5, 5 (1d6 + 2) Bludgeoning.
func sbSlam() content.StatBlockPart {
	return sbAttackPart("Slam", sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange()),
		sbClause(content.ClauseHit, nil, sbDamage(5, sbExpr(1, 6, 2), content.DamageBludgeoning)))
}

func sbPlanOK(t *testing.T, part content.StatBlockPart, c combat.DamageNotationChoices, opts ...combat.AttackPlanOption) combat.StatBlockAttackPlan {
	t.Helper()
	plan, err := combat.PlanStatBlockAttack(part, c, opts...)
	if err != nil || plan.IsZero() {
		t.Fatalf("PlanStatBlockAttack = %+v, %v, want a plan", plan, err)
	}
	return plan
}

// Requirement: doc/specs/monster-stat-block-actions.md table AT-1 (FR-022 to FR-025, FR-053) - the printed bonus is the whole attack
// modifier, the form follows the delivery, the reach or range is the printed one, and a bad combination is an error (SRD 5.2.1,
// "Attack Notation", p. 256).
// Path: per case.
func TestPlanStatBlockAttackNotation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		atk                     content.StatBlockAttack
		opts                    []combat.AttackPlanOption
		form                    combat.AttackForm
		reach                   int
		rng                     [2]int
		bonus                   int
	}{
		{"whole bonus", `AT-1; "Aboleth", p. 258`, "happy", sbAttackNotation(content.AttackMelee, 9, "", 15, sbNoRange()), nil, combat.MeleeWeaponAttack, 15, [2]int{}, 9},
		{"ranged with long range", `AT-1; "Assassin", p. 260`, "happy", sbAttackNotation(content.AttackRanged, 7, "", 0, sbRange(80, 320)), nil, combat.RangedWeaponAttack, 0, [2]int{80, 320}, 7},
		{"single range", `AT-1; "Priest", p. 316`, "unhappy (boundary)", sbAttackNotation(content.AttackRanged, 4, "", 0, sbRange(60, 0)), nil, combat.RangedWeaponAttack, 0, [2]int{60, 0}, 4},
		{"either delivery, melee", `AT-1; "Bugbear Stalker", p. 271`, "unhappy (rule interaction)", sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 10, sbRange(30, 120)), []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryMelee)}, combat.MeleeWeaponAttack, 10, [2]int{}, 4},
		{"either delivery, ranged", `AT-1; "Bugbear Stalker", p. 271`, "unhappy (rule interaction)", sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 10, sbRange(30, 120)), []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryRanged)}, combat.RangedWeaponAttack, 0, [2]int{30, 120}, 4},
		{"either delivery, other range", `AT-1; "Bugbear Warrior", p. 272`, "unhappy (boundary)", sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 5, sbRange(20, 60)), []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryRanged)}, combat.RangedWeaponAttack, 0, [2]int{20, 60}, 4},
		{"fixed delivery omitted", `AT-1, FR-053; "Aboleth", p. 258`, "happy", sbAttackNotation(content.AttackMelee, 9, "", 15, sbNoRange()), []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryMelee)}, combat.MeleeWeaponAttack, 15, [2]int{}, 9},
		{"negative bonus", `AT-1; p. 256`, "unhappy (boundary)", sbAttackNotation(content.AttackMelee, -1, "", 5, sbNoRange()), nil, combat.MeleeWeaponAttack, 5, [2]int{}, -1},
		{"zero bonus", `AT-1; p. 256`, "unhappy (boundary)", sbAttackNotation(content.AttackMelee, 0, "", 5, sbNoRange()), nil, combat.MeleeWeaponAttack, 5, [2]int{}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			plan := sbPlanOK(t, sbAttackPart("Hit", c.atk), combat.DamageNotationChoices{}, c.opts...)
			if plan.Form() != c.form || !plan.Modifiers().Equal(d20.NewRollModifiers(modifier.New(c.bonus))) || plan.Modifiers().Count() != 1 {
				t.Errorf("form %v modifiers %+v, want %v and the bonus %d alone [%s]", plan.Form(), plan.Modifiers(), c.form, c.bonus, c.requirement)
			}
			reach, hasReach := plan.Reach()
			rng, hasRange := plan.Range()
			if hasReach != (c.reach != 0) || (hasReach && reach.Feet() != c.reach) || hasRange != (c.rng[0] != 0) || (hasRange && (rng.Normal() != c.rng[0] || rng.Long() != c.rng[1])) {
				t.Errorf("reach %v %v range %v %v, want reach %d range %v", reach, hasReach, rng, hasRange, c.reach, c.rng)
			}
			if _, ok := plan.Damage(); ok {
				t.Error("a part with no Hit damage has a DamageSpec")
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md table AT-1 "no attack notation", "delivery missing", "melee without reach"
// (FR-024, FR-025, FR-053) - a part with no Attack Notation, a Melee or Ranged attack with no delivery, a delivery the attack cannot take,
// no printed reach or range, damage choices that do not fit and damage the rules refuse are errors matching srderr.ErrInvalidInput.
// Path: per case.
func TestPlanStatBlockAttackRejectsInvalidInput(t *testing.T) {
	t.Parallel()
	either := sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 10, sbRange(30, 120))
	meleeOnlyEither := sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 10, sbNoRange())
	rangedOnlyEither := sbAttackNotation(content.AttackMeleeOrRanged, 4, "", 0, sbRange(30, 120))
	zeroReach := luMust(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: 1, HasReach: true}))
	rangedNoRange := sbAttackNotation(content.AttackRanged, 4, "", 0, sbNoRange())
	meleeNoReach := sbAttackNotation(content.AttackMelee, 4, "", 0, sbNoRange())
	hit := func(d ...content.StatBlockDamage) content.EffectClause { return sbClause(content.ClauseHit, nil, d...) }
	wide := sbDamage(5, sbExpr(dice.MaxCount, 6, 0), content.DamageFire)
	described := luMust(content.NewStatBlockDamage(3, dice.Expression{}, luMust(content.NewDescribedDamageType("the chosen type")), false))
	melee := func(c ...content.EffectClause) content.StatBlockPart {
		return sbAttackPart("Hit", sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange()), c...)
	}
	cases := []struct {
		name, requirement, path string
		part                    content.StatBlockPart
		choices                 combat.DamageNotationChoices
		opts                    []combat.AttackPlanOption
	}{
		{"trait with no attack", `FR-053`, "unhappy (invalid input)", sbPart("Trait", content.StatBlockPartFields{}), combat.DamageNotationChoices{}, nil},
		{"save-only part", `FR-053; AT-1 "no attack notation"`, "unhappy (invalid input)", sbSavePart("Breath", sbSave(1, 12, "", "")), combat.DamageNotationChoices{}, nil},
		{"delivery missing", `FR-024; AT-1 "delivery missing"`, "unhappy (invalid input)", sbAttackPart("Javelin", either), combat.DamageNotationChoices{}, nil},
		{"delivery undefined", `FR-024`, "unhappy (invalid input)", sbAttackPart("Javelin", either), combat.DamageNotationChoices{}, []combat.AttackPlanOption{combat.WithDelivery(7)}},
		{"ranged chosen for a melee attack", `FR-025; AT-1 "melee without reach"`, "unhappy (invalid input)", melee(), combat.DamageNotationChoices{}, []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryRanged)}},
		{"melee chosen for a ranged attack", `FR-025; AT-1 "melee without reach"`, "unhappy (invalid input)", sbAttackPart("Bow", sbAttackNotation(content.AttackRanged, 4, "", 0, sbRange(30, 60))), combat.DamageNotationChoices{}, []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryMelee)}},
		{"melee with no printed reach", `FR-025`, "unhappy (invalid input)", sbAttackPart("Javelin", rangedOnlyEither), combat.DamageNotationChoices{}, []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryMelee)}},
		{"ranged with no printed range", `FR-025`, "unhappy (invalid input)", sbAttackPart("Javelin", meleeOnlyEither), combat.DamageNotationChoices{}, []combat.AttackPlanOption{combat.WithDelivery(combat.AttackDeliveryRanged)}},
		{"melee attack printing no reach", `FR-025`, "unhappy (invalid input)", sbAttackPart("Claw", meleeNoReach), combat.DamageNotationChoices{}, nil},
		{"ranged attack printing no range", `FR-025`, "unhappy (invalid input)", sbAttackPart("Bow", rangedNoRange), combat.DamageNotationChoices{}, nil},
		{"printed reach of 0 feet", `FR-025`, "unhappy (boundary)", sbAttackPart("Claw", zeroReach), combat.DamageNotationChoices{}, nil},
		{"dice chosen for a number only", `FR-027; AT-1 "dice not printed"`, "unhappy (boundary)", melee(hit(sbDamage(1, dice.Expression{}, content.DamageSlashing))), sbDiceChoices(t), nil},
		{"two instances, one choice", `FR-027; AT-1 "wrong count"`, "unhappy (invalid input)", melee(hit(sbDamage(7, sbExpr(1, 6, 4), content.DamagePiercing), sbDamage(17, sbExpr(5, 6, 0), content.DamagePoison))), luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice)), nil},
		{"choices for a hit with no damage", `FR-027`, "unhappy (invalid input)", melee(), luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice)), nil},
		{"choices for a hit clause with no damage", `FR-027`, "unhappy (invalid input)", melee(hit()), luMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber)), nil},
		{"no choices for a hit with damage", `FR-027`, "unhappy (empty and minimal)", melee(hit(sbDamage(7, sbExpr(1, 6, 4), content.DamagePiercing))), combat.DamageNotationChoices{}, nil},
		{"damage type printed as text", `FR-026`, "unhappy (invalid input)", melee(hit(described)), sbNumbers(t), nil},
		{"dice more than a Critical Hit can double", `FR-029`, "unhappy (boundary)", melee(hit(wide)), sbDiceChoices(t), nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			plan, err := combat.PlanStatBlockAttack(c.part, c.choices, c.opts...)
			if !errors.Is(err, srderr.ErrInvalidInput) || !plan.IsZero() {
				t.Errorf("PlanStatBlockAttack = %+v, %v, want the zero plan and an invalid-input error [%s]", plan, err, c.requirement)
			}
		})
	}
	// Choosing the number for the same wide dice is fine: no die is ever rolled.
	if _, err := combat.PlanStatBlockAttack(melee(hit(wide)), sbNumbers(t)); err != nil {
		t.Errorf("number chosen for wide dice = %v, want a plan", err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md table AT-1 "number chosen", "dice chosen", "two instances, mixed", "no ability modifier",
// "Hit damage only", "no Hit damage" (FR-026, FR-028, FR-030) - each instance is rolled by its number (FixedDamage) or its dice (DiceDamage with the
// printed flat modifier), never both, never adding the ability modifier, from the Hit clause only (SRD 5.2.1, "Damage Notation", p. 257).
// Path: per case.
func TestPlanStatBlockAttackDamage(t *testing.T) {
	t.Parallel()
	assassin := sbAttackPart("Shortsword", sbAttackNotation(content.AttackMelee, 7, "", 5, sbNoRange()),
		sbClause(content.ClauseHit, nil, sbDamage(7, sbExpr(1, 6, 4), content.DamagePiercing), sbDamage(17, sbExpr(5, 6, 0), content.DamagePoison)),
		sbClause(content.ClauseHitOrMiss, nil, sbDamage(9, sbExpr(2, 8, 0), content.DamageFire)))
	slam := sbSlam()
	cases := []struct {
		name, requirement, path string
		part                    content.StatBlockPart
		choices                 combat.DamageNotationChoices
		want                    []combat.DamageComponent
	}{
		{"number chosen", `AT-1; "Animated Armor", p. 259`, "happy", slam, sbNumbers(t), []combat.DamageComponent{combat.MustFixedDamage(5, content.DamageBludgeoning, 0, false)}},
		{"dice chosen", `AT-1; "Animated Armor", p. 259`, "happy", slam, sbDiceChoices(t), []combat.DamageComponent{combat.MustDiceDamage(1, dice.D6(), content.DamageBludgeoning, 2, false)}},
		{"two instances, mixed", `AT-1; "Assassin", p. 260`, "unhappy (rule interaction)", assassin, luMust(combat.NewDamageNotationChoices(combat.DamageNotationNumber, combat.DamageNotationDice)),
			[]combat.DamageComponent{combat.MustFixedDamage(7, content.DamagePiercing, 0, false), combat.MustDiceDamage(5, dice.D6(), content.DamagePoison, 0, false)}},
		{"two instances, all dice, in printed order", `FR-030`, "unhappy (rule interaction)", assassin, sbDiceChoices(t),
			[]combat.DamageComponent{combat.MustDiceDamage(1, dice.D6(), content.DamagePiercing, 4, false), combat.MustDiceDamage(5, dice.D6(), content.DamagePoison, 0, false)}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			spec, ok := sbPlanOK(t, c.part, c.choices).Damage()
			if !ok || !slices.Equal(spec.Components(), c.want) {
				t.Errorf("Damage() = %v, %v, want %v [%s]", spec.Components(), ok, c.want, c.requirement)
			}
			for _, comp := range spec.Components() {
				switch v := comp.(type) {
				case combat.FixedDamage:
					if v.AddsAbilityModifier() {
						t.Error("a planned component adds the ability modifier (FR-028)")
					}
				case combat.DiceDamage:
					if v.AddsAbilityModifier() {
						t.Error("a planned component adds the ability modifier (FR-028)")
					}
				}
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-023 - the printed qualifier of an attack is exposed verbatim and never interpreted.
// Path: unhappy (rule interaction).
func TestPlanStatBlockAttackQualifierIsVerbatim(t *testing.T) {
	t.Parallel()
	const q = "Advantage if the target is Grappled"
	part := sbAttackPart("Bite", sbAttackNotation(content.AttackMelee, 5, q, 5, sbNoRange()))
	plan := sbPlanOK(t, part, combat.DamageNotationChoices{})
	if plan.Qualifier() != q || !plan.Modifiers().Equal(d20.NewRollModifiers(modifier.New(5))) {
		t.Errorf("Qualifier() = %q with modifiers %+v, want the text verbatim and no change to the roll", plan.Qualifier(), plan.Modifiers())
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-029, section 8 - a planned command is accepted by the real ResolveAttack with a fixed
// roller: the printed bonus is the whole modifier, a dice instance doubles its dice on a Critical Hit and keeps its flat modifier once, and a number
// instance stays as printed (SRD 5.2.1, "Critical Hits", p. 16; "Damage Notation", p. 257).
// Path: per case.
func TestPlannedAttackIsAcceptedByResolveAttack(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, requirement, path string
		choices                 combat.DamageNotationChoices
		faces                   []int
		requests                []string
		wantDamage              int
		wantCritical            bool
	}{
		{"hit with dice", `FR-029; "Damage Notation", p. 257`, "happy", sbDiceChoices(t), []int{10, 4}, []string{"1d20", "1d6"}, 6, false},
		{"hit with the number", `FR-026`, "happy", sbNumbers(t), []int{10}, []string{"1d20"}, 5, false},
		{"Critical Hit with dice doubles the dice once", `FR-029; "Critical Hits", p. 16`, "unhappy (rule interaction)", sbDiceChoices(t), []int{20, 4, 3}, []string{"1d20", "2d6"}, 9, true},
		{"Critical Hit with the number adds nothing", `FR-029`, "unhappy (rule interaction)", sbNumbers(t), []int{20}, []string{"1d20"}, 5, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			plan := sbPlanOK(t, sbSlam(), c.choices)
			spec, _ := plan.Damage()
			reach, _ := plan.Reach()
			cmd := attackCommand(t, combat.ResolveAttackParams{Form: plan.Form(), ArmorClass: 14, Modifiers: plan.Modifiers(), Distance: 5, Reach: reach, Damage: spec})
			res, requests := runAttack(t, cmd, c.faces...)
			resolved, ok := res.(combat.AttackResolved)
			if !ok {
				t.Fatalf("result = %#v, want AttackResolved", res)
			}
			var total int
			for _, d := range resolved.Damage() {
				total += d.Amount()
			}
			if total != c.wantDamage || !slices.Equal(requests, c.requests) {
				t.Errorf("damage %d from requests %v, want %d from %v [%s]", total, requests, c.wantDamage, c.requests, c.requirement)
			}
		})
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md table AT-1 "clauses on a hit", "clauses on a miss", "a printed Miss", "not possible" (FR-031) - a hit,
// a Critical Hit included, applies Hit and Hit or Miss; a miss applies Miss and Hit or Miss; Hit and Miss never both; an attack that was not possible
// applies none; the clauses come in printed order (SRD 5.2.1, "Attack Notation", p. 256).
// Path: per case.
func TestAppliedClausesForAttack(t *testing.T) {
	t.Parallel()
	atk := sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange())
	hit, miss, either := content.ClauseHit, content.ClauseMiss, content.ClauseHitOrMiss
	skirmisher := sbAttackPart("Ocean Spear", atk, sbClause(hit, nil), sbClause(either, nil))
	printedMiss := sbAttackPart("Bite", atk, sbClause(either, nil), sbClause(hit, nil), sbClause(miss, nil))
	cases := []struct {
		name, requirement, path string
		part                    content.StatBlockPart
		hit, possible           bool
		want                    []content.EffectClauseLabel
	}{
		{"clauses on a hit", `AT-1; "Merfolk Skirmisher", p. 308`, "happy", skirmisher, true, true, []content.EffectClauseLabel{hit, either}},
		{"clauses on a miss", `AT-1; "Merfolk Skirmisher", p. 308`, "unhappy (rule interaction)", skirmisher, false, true, []content.EffectClauseLabel{either}},
		{"a printed Miss, on a miss", `AT-1; p. 256`, "unhappy (boundary)", printedMiss, false, true, []content.EffectClauseLabel{either, miss}},
		{"a printed Miss, on a hit", `AT-1; p. 256`, "unhappy (boundary)", printedMiss, true, true, []content.EffectClauseLabel{either, hit}},
		{"not possible, hit", `AT-1 "not possible"`, "unhappy (state error)", printedMiss, true, false, nil},
		{"not possible, miss", `AT-1 "not possible"`, "unhappy (state error)", printedMiss, false, false, nil},
		{"no clauses", `FR-031`, "unhappy (empty and minimal)", sbAttackPart("Claw", atk), true, true, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.AppliedClausesForAttack(c.part, c.hit, c.possible)
			var labels []content.EffectClauseLabel
			for _, cl := range got {
				labels = append(labels, cl.Label())
			}
			if err != nil || !slices.Equal(labels, c.want) {
				t.Errorf("AppliedClausesForAttack = %v, %v, want %v [%s]", labels, err, c.want, c.requirement)
			}
		})
	}
	if got, err := combat.AppliedClausesForAttack(sbSavePart("Breath", sbSave(1, 12, "", "")), true, true); got != nil || !errors.Is(err, srderr.ErrInvalidInput) {
		t.Errorf("part with no Attack Notation = %v, %v, want an invalid-input error", got, err)
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-026, FR-027 - the choices are fixed before any roll, one per instance in printed order or
// the same for all; a list of another length, no choice and an undefined choice are errors; the zero value fits a part with no damage.
// Path: per case.
func TestDamageNotationChoices(t *testing.T) {
	t.Parallel()
	one := []content.StatBlockDamage{sbDamage(5, sbExpr(1, 6, 2), content.DamageFire)}
	two := append(slices.Clone(one), sbDamage(1, dice.Expression{}, content.DamageCold))
	each := luMust(combat.NewDamageNotationChoices(combat.DamageNotationDice, combat.DamageNotationNumber))
	cases := []struct {
		name, requirement, path string
		choices                 combat.DamageNotationChoices
		damage                  []content.StatBlockDamage
		want                    []combat.DamageNotationChoice
		ok                      bool
	}{
		{"one per instance", `FR-027`, "happy", each, two, []combat.DamageNotationChoice{combat.DamageNotationDice, combat.DamageNotationNumber}, true},
		{"the same for all, one", `FR-027`, "happy", sbNumbers(t), one, []combat.DamageNotationChoice{combat.DamageNotationNumber}, true},
		{"the same for all, two", `FR-027`, "happy", sbNumbers(t), two, []combat.DamageNotationChoice{combat.DamageNotationNumber, combat.DamageNotationNumber}, true},
		{"too few", `FR-027`, "unhappy (boundary)", each, one, nil, false},
		{"too many", `FR-027`, "unhappy (boundary)", each, append(slices.Clone(two), two[0]), nil, false},
		{"dice for an instance with no dice", `FR-027`, "unhappy (boundary)", sbDiceChoices(t), two, nil, false},
		{"zero value, no damage", `FR-027`, "unhappy (empty and minimal)", combat.DamageNotationChoices{}, nil, nil, true},
		{"zero value, damage", `FR-027`, "unhappy (empty and minimal)", combat.DamageNotationChoices{}, one, nil, false},
		{"all for no damage", `FR-027`, "unhappy (empty and minimal)", sbNumbers(t), nil, []combat.DamageNotationChoice{}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.choices.For(c.damage)
			if (err == nil) != c.ok || (!c.ok && !errors.Is(err, srderr.ErrInvalidInput)) || (c.ok && !slices.Equal(got, c.want)) {
				t.Errorf("For = %v, %v, want %v (ok %v) [%s]", got, err, c.want, c.ok, c.requirement)
			}
		})
	}
	for name, build := range map[string]func() (combat.DamageNotationChoices, error){
		"no choice":   func() (combat.DamageNotationChoices, error) { return combat.NewDamageNotationChoices() },
		"zero choice": func() (combat.DamageNotationChoices, error) { return combat.NewDamageNotationChoices(0) },
		"choice past the second": func() (combat.DamageNotationChoices, error) {
			return combat.NewDamageNotationChoices(combat.DamageNotationDice, 3)
		},
		"all, zero":            func() (combat.DamageNotationChoices, error) { return combat.AllDamageNotation(0) },
		"all, past the second": func() (combat.DamageNotationChoices, error) { return combat.AllDamageNotation(3) },
	} {
		if c, err := build(); !errors.Is(err, srderr.ErrInvalidInput) || !c.Equal(combat.DamageNotationChoices{}) {
			t.Errorf("%s: %+v, %v, want the zero choices and an invalid-input error", name, c, err)
		}
	}
	// The slice given is copied, and For returns a copy.
	in := []combat.DamageNotationChoice{combat.DamageNotationDice, combat.DamageNotationNumber}
	c := luMust(combat.NewDamageNotationChoices(in...))
	in[0] = combat.DamageNotationNumber
	got, _ := c.For(two)
	got[1] = combat.DamageNotationDice
	if again, _ := c.For(two); again[0] != combat.DamageNotationDice || again[1] != combat.DamageNotationNumber {
		t.Error("the choices alias the caller's slice or the slice returned")
	}
	if !each.Equal(c) || each.Equal(sbNumbers(t)) || sbNumbers(t).Equal(sbDiceChoices(t)) {
		t.Error("Equal does not tell choices apart")
	}
	if combat.DamageNotationNumber.String() != "number" || combat.DamageNotationDice.String() != "dice" || combat.DamageNotationChoice(0).String() != "unknown" {
		t.Error("choice names are not number, dice and unknown")
	}
}

// FuzzNewDamageNotationChoices checks that no list of choices and no number of instances makes the constructors or For panic, that a
// list that is accepted fits exactly the instance count it has, and that "the same for all" fits every count.
func FuzzNewDamageNotationChoices(f *testing.F) {
	f.Add([]byte{1, 2}, 2)
	f.Add([]byte{}, 0)
	f.Add([]byte{0, 9}, 3)
	f.Fuzz(func(t *testing.T, raw []byte, instances int) {
		if instances < 0 || instances > 8 {
			return
		}
		damage := make([]content.StatBlockDamage, instances)
		for i := range damage {
			damage[i] = sbDamage(1, sbExpr(1, 6, 0), content.DamageFire)
		}
		choices := make([]combat.DamageNotationChoice, len(raw))
		for i, b := range raw {
			choices[i] = combat.DamageNotationChoice(b % 4)
		}
		c, err := combat.NewDamageNotationChoices(choices...)
		if err != nil {
			if !errors.Is(err, srderr.ErrInvalidInput) {
				t.Fatalf("error %v does not match ErrInvalidInput", err)
			}
			return
		}
		got, err := c.For(damage)
		if (err == nil) != (len(choices) == instances) || (err == nil && !slices.Equal(got, choices)) {
			t.Fatalf("For(%d instances) with %v = %v, %v", instances, choices, got, err)
		}
		all, _ := combat.AllDamageNotation(combat.DamageNotationDice)
		if got, err := all.For(damage); err != nil || len(got) != instances {
			t.Fatalf("all.For(%d) = %v, %v", instances, got, err)
		}
	})
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-022 - a bonus at either end of the printed range is a bonus: the plan holds it as the whole modifier.
// Path: unhappy (boundary).
func TestPlanStatBlockAttackAtTheEndsOfTheBonusRange(t *testing.T) {
	t.Parallel()
	for _, bonus := range []int{content.MonsterMaxNumber, -content.MonsterMaxNumber} {
		part := sbAttackPart("Hit", sbAttackNotation(content.AttackMelee, bonus, "", 5, sbNoRange()))
		if plan := sbPlanOK(t, part, combat.DamageNotationChoices{}); !plan.Modifiers().Equal(d20.NewRollModifiers(modifier.New(bonus))) {
			t.Errorf("bonus %d: modifiers = %+v", bonus, plan.Modifiers())
		}
	}
}

// Requirement: doc/specs/monster-stat-block-actions.md FR-026, FR-027, FR-030, FR-031 - the zero plan reports neither a reach nor a range; a Hit clause that prints no damage plans no DamageSpec with no
// choices; the clauses of a saving throw (Failure, Success and the others) are no clause of an attack, whatever the outcome; choices compare equal only to the same choices, whichever side asks.
// Path: per case.
func TestStatBlockAttackEdgeCases(t *testing.T) {
	t.Parallel()
	var zero combat.StatBlockAttackPlan
	if _, ok := zero.Reach(); ok {
		t.Error("the zero plan has a reach")
	}
	if _, ok := zero.Range(); ok {
		t.Error("the zero plan has a range")
	}
	atk := sbAttackNotation(content.AttackMelee, 4, "", 5, sbNoRange())
	noDamage := sbAttackPart("Claw", atk, sbClause(content.ClauseHit, nil))
	if _, ok := sbPlanOK(t, noDamage, combat.DamageNotationChoices{}).Damage(); ok {
		t.Error("a Hit clause with no damage planned a DamageSpec")
	}
	mixed := sbAttackPart("Claw", atk, sbClause(content.ClauseHit, nil), sbClause(content.ClauseMiss, nil), sbClause(content.ClauseHitOrMiss, nil),
		sbClause(content.ClauseFailure, nil), sbClause(content.ClauseSuccess, nil), sbClause(content.ClauseFailureOrSuccess, nil), sbClause(content.ClauseFailureByFiveOrMore, nil))
	for _, hit := range []bool{true, false} {
		got, err := combat.AppliedClausesForAttack(mixed, hit, true)
		want := []content.EffectClauseLabel{content.ClauseMiss, content.ClauseHitOrMiss}
		if hit {
			want = []content.EffectClauseLabel{content.ClauseHit, content.ClauseHitOrMiss}
		}
		if err != nil || !slices.Equal(labelsOf(got), want) {
			t.Errorf("hit %v: clauses = %v, %v, want %v", hit, labelsOf(got), err, want)
		}
	}
	if sbDiceChoices(t).Equal(sbNumbers(t)) || sbNumbers(t).Equal(sbDiceChoices(t)) {
		t.Error("the choices of dice equal the choices of numbers")
	}
}
