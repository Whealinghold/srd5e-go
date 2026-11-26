package combat_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

func smD4() combat.BonusDie { return combat.MustBonusDie(dice.MustExpression(1, dice.D4(), 0), false) }

// Requirement: doc/specs/spell-granted-modifiers.md table R rows R9 to R21, FR-010, FR-016 to FR-021 - what each modelled spell adds to the roll it is asked about, and nothing to any other roll (SRD 5.2.1, "Guidance", p. 138; "Haste", p. 139; "Slow", p. 163; "Bestow Curse", pp. 112-113; "Hunter's Mark", p. 141).
// Path: per case.
func TestQueryRollContributions(t *testing.T) {
	t.Parallel()
	guidance := []combat.SpellModifierSource{smSrc(t, "guidance", "c", "a", 0, combat.ChosenSkill{Skill: content.SkillStealth})}
	haste, slow := smPlain(t, "haste", 0), smPlain(t, "slow", 1)
	curseAbility := smSrc(t, "bestow_curse", "c", "a", 0, combat.ChosenCurse{Effect: combat.CurseAbility{Ability: ability.Wisdom}})
	curseAttacks := smSrc(t, "bestow_curse", "c", "a", 0, combat.ChosenCurse{Effect: combat.CurseAttackRolls{}})
	curseOther := []combat.SpellModifierSource{smSrc(t, "bestow_curse", "c", "a", 0, combat.ChosenCurse{Effect: combat.CurseDodge{}}), smSrc(t, "bestow_curse", "c", "b", 0, combat.ChosenCurse{Effect: combat.CurseExtraDamage{}})}
	mark := smSrc(t, "hunters_mark", "c", "m", 0, combat.NoChoice{})
	adv, dis := []modifier.RollCircumstance{modifier.Advantage{}}, []modifier.RollCircumstance{modifier.Disadvantage{}}
	check := func(who string, a ability.Ability, s content.Skill, seeking string) combat.AbilityCheckContext {
		c := combat.AbilityCheckContext{Creature: cid(who), Ability: a, Skill: s}
		if seeking != "" {
			c.Seeking = cid(seeking)
		}
		return c
	}
	save := func(who string, a ability.Ability) combat.SavingThrowContext {
		return combat.SavingThrowContext{Creature: cid(who), Ability: a}
	}
	cases := []struct {
		name, row, path string
		sources         []combat.SpellModifierSource
		ctx             combat.RollContext
		dice            int
		mods            []modifier.Modifier
		circ            []modifier.RollCircumstance
	}{
		{"Guidance, Stealth check", "R9", "happy", guidance, check("a", ability.Dexterity, content.SkillStealth, ""), 1, nil, nil},
		{"Guidance, Stealth with another ability", "FR-016", "unhappy (boundary)", guidance, check("a", ability.Strength, content.SkillStealth, ""), 1, nil, nil},
		{"Guidance, Acrobatics check", "R10", "unhappy (boundary)", guidance, check("a", ability.Dexterity, content.SkillAcrobatics, ""), 0, nil, nil},
		{"Guidance, raw Dexterity check", "R11", "unhappy (boundary)", guidance, check("a", ability.Dexterity, 0, ""), 0, nil, nil},
		{"Guidance, check of another creature", "FR-010", "unhappy (rule interaction)", guidance, check("b", ability.Dexterity, content.SkillStealth, ""), 0, nil, nil},
		{"Guidance, a save", "FR-016", "unhappy (boundary)", guidance, save("a", ability.Dexterity), 0, nil, nil},
		{"Guidance, an attack roll", "FR-016", "unhappy (boundary)", guidance, combat.AttackRollContext{Attacker: cid("a"), Target: cid("b")}, 0, nil, nil},
		{"Guidance, a Death Saving Throw", "FR-016", "unhappy (boundary)", guidance, combat.DeathSavingThrowContext{Creature: cid("a")}, 0, nil, nil},
		{"Haste, Dexterity save", "R12", "happy", []combat.SpellModifierSource{haste}, save("a", ability.Dexterity), 0, nil, adv},
		{"Haste, Wisdom save", "R13", "unhappy (boundary)", []combat.SpellModifierSource{haste}, save("a", ability.Wisdom), 0, nil, nil},
		{"Haste, Dexterity save of another creature", "FR-010", "unhappy (rule interaction)", []combat.SpellModifierSource{haste}, save("b", ability.Dexterity), 0, nil, nil},
		{"Haste, Dexterity check", "FR-017", "unhappy (boundary)", []combat.SpellModifierSource{haste}, check("a", ability.Dexterity, 0, ""), 0, nil, nil},
		{"Slow, Dexterity save", "R14", "happy", []combat.SpellModifierSource{slow}, save("a", ability.Dexterity), 0, []modifier.Modifier{modifier.New(-2)}, nil},
		{"Slow, Strength save", "FR-018", "unhappy (boundary)", []combat.SpellModifierSource{slow}, save("a", ability.Strength), 0, nil, nil},
		{"Haste and Slow, Dexterity save", "R15", "unhappy (rule interaction)", []combat.SpellModifierSource{slow, haste}, save("a", ability.Dexterity), 0, []modifier.Modifier{modifier.New(-2)}, adv},
		{"Bestow Curse on Wisdom, save", "R18", "happy", []combat.SpellModifierSource{curseAbility}, save("a", ability.Wisdom), 0, nil, dis},
		{"Bestow Curse on Wisdom, check", "R18", "happy", []combat.SpellModifierSource{curseAbility}, check("a", ability.Wisdom, content.SkillPerception, ""), 0, nil, dis},
		{"Bestow Curse on Wisdom, Strength save", "FR-020", "unhappy (boundary)", []combat.SpellModifierSource{curseAbility}, save("a", ability.Strength), 0, nil, nil},
		{"Bestow Curse on Wisdom, Death Saving Throw", "FR-020", "unhappy (boundary)", []combat.SpellModifierSource{curseAbility}, combat.DeathSavingThrowContext{Creature: cid("a")}, 0, nil, nil},
		{"Bestow Curse on Wisdom, attack roll", "FR-020", "unhappy (boundary)", []combat.SpellModifierSource{curseAbility}, combat.AttackRollContext{Attacker: cid("a"), Target: cid("c")}, 0, nil, nil},
		{"curse of attack rolls, A attacks the caster", "R19", "unhappy (rule interaction)", []combat.SpellModifierSource{curseAttacks}, combat.AttackRollContext{Attacker: cid("a"), Target: cid("c")}, 0, nil, dis},
		{"curse of attack rolls, A attacks D", "R19", "unhappy (rule interaction)", []combat.SpellModifierSource{curseAttacks}, combat.AttackRollContext{Attacker: cid("a"), Target: cid("d")}, 0, nil, nil},
		{"curse of attack rolls, the caster attacks A", "R19", "unhappy (rule interaction)", []combat.SpellModifierSource{curseAttacks}, combat.AttackRollContext{Attacker: cid("c"), Target: cid("a")}, 0, nil, nil},
		{"curse of attack rolls, a save", "FR-020", "unhappy (boundary)", []combat.SpellModifierSource{curseAttacks}, save("a", ability.Wisdom), 0, nil, nil},
		{"curse of Dodge and of extra damage add no roll contribution", "FR-020", "unhappy (boundary)", curseOther, save("a", ability.Wisdom), 0, nil, nil},
		{"Hunter's Mark, Survival seeking the mark", "R20", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Wisdom, content.SkillSurvival, "m"), 0, nil, adv},
		{"Hunter's Mark, Perception seeking the mark", "FR-021", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Wisdom, content.SkillPerception, "m"), 0, nil, adv},
		{"Hunter's Mark, seeking another creature", "R20", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Wisdom, content.SkillSurvival, "n"), 0, nil, nil},
		{"Hunter's Mark, seeking no creature", "FR-021", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Wisdom, content.SkillSurvival, ""), 0, nil, nil},
		{"Hunter's Mark, another skill", "FR-021", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Wisdom, content.SkillInsight, "m"), 0, nil, nil},
		{"Hunter's Mark, another ability", "FR-021", "unhappy (boundary)", []combat.SpellModifierSource{mark}, check("c", ability.Intelligence, content.SkillSurvival, "m"), 0, nil, nil},
		{"Hunter's Mark, the marked creature seeks the caster", "FR-010", "unhappy (rule interaction)", []combat.SpellModifierSource{mark}, check("m", ability.Wisdom, content.SkillSurvival, "c"), 0, nil, nil},
		{"Hunter's Mark, the creature seeking is not the caster", "FR-010", "unhappy (rule interaction)", []combat.SpellModifierSource{mark}, check("d", ability.Wisdom, content.SkillSurvival, "m"), 0, nil, nil},
		{"spell that is not modelled", "R21", "unhappy (missing content)", []combat.SpellModifierSource{smPlain(t, "fireball", 0)}, save("a", ability.Dexterity), 0, nil, nil},
		{"no sources", "FR-024", "unhappy (empty and minimal)", nil, save("a", ability.Dexterity), 0, nil, nil},
		{"empty sources", "FR-024", "unhappy (empty and minimal)", []combat.SpellModifierSource{}, save("a", ability.Dexterity), 0, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, err := combat.QueryRollContributions(c.sources, c.ctx)
			if err != nil {
				t.Fatal(err)
			}
			var wantDice []combat.BonusDie
			if c.dice > 0 {
				wantDice = []combat.BonusDie{smD4()}
			}
			if !slices.Equal(got.BonusDice(), wantDice) || !slices.Equal(got.Modifiers(), c.mods) || !slices.Equal(got.Circumstances(), c.circ) {
				t.Errorf("contributions = %v %v %v, want %v %v %v [%s%s]", got.BonusDice(), got.Modifiers(), got.Circumstances(), wantDice, c.mods, c.circ, spellModifiersSpec, c.row)
			}
		})
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-008, FR-084 - two sources with one spell and one target are ErrDuplicateSource (which also matches srderr.ErrInvalidInput) from every query, even for a roll they do not touch; the same spell on two targets, and two spells with equal CastOrder, are legal; a zero source is invalid input.
// Path: unhappy (rule interaction; invalid input).
func TestQueriesRejectDuplicateAndZeroSources(t *testing.T) {
	t.Parallel()
	dup := []combat.SpellModifierSource{smPlain(t, "haste", 0), smSrc(t, "haste", "d", "a", 1, combat.NoChoice{})}
	legal := []combat.SpellModifierSource{smPlain(t, "haste", 0), smSrc(t, "haste", "c", "b", 0, combat.NoChoice{}), smPlain(t, "slow", 0)}
	zeroes := []combat.SpellModifierSource{{}}
	rc := combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}
	queries := map[string]func(s []combat.SpellModifierSource) error{
		"roll": func(s []combat.SpellModifierSource) error { _, err := combat.QueryRollContributions(s, rc); return err },
		"armor class": func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryArmorClass(s, cid("a"), false)
			return err
		},
		"speed": func(s []combat.SpellModifierSource) error {
			_, err := combat.QuerySpeeds(s, cid("a"), luMust(combat.NewMovementSpeeds(30)))
			return err
		},
		"damage riders": func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryDamageRiders(s, cid("c"), cid("a"), combat.OtherDamageEvent{})
			return err
		},
		"damage reduction": func(s []combat.SpellModifierSource) error {
			_, err := combat.QueryDamageReduction(s, cid("a"), content.DamageFire, false)
			return err
		},
	}
	for name, q := range queries {
		if err := q(dup); !errors.Is(err, combat.ErrDuplicateSource) || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("%s query of a duplicate = %v, want ErrDuplicateSource", name, err)
		}
		if err := q(legal); err != nil {
			t.Errorf("%s query of legal sources = %v", name, err)
		}
		if err := q(zeroes); !errors.Is(err, srderr.ErrInvalidInput) || errors.Is(err, combat.ErrDuplicateSource) {
			t.Errorf("%s query of a zero source = %v, want ErrInvalidInput", name, err)
		}
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-084 - an invalid roll context (nil, a zero creature, an invalid ability or skill, an unknown variant) is invalid input and gives an empty result; a valid context with a zero skill and no seeking is valid.
// Path: unhappy (invalid input).
func TestQueryRollContributionsRejectsInvalidContext(t *testing.T) {
	t.Parallel()
	bad := []combat.RollContext{nil, combat.AttackRollContext{Attacker: cid("a")}, combat.AttackRollContext{Target: cid("a")},
		combat.SavingThrowContext{Ability: ability.Dexterity}, combat.SavingThrowContext{Creature: cid("a")},
		combat.DeathSavingThrowContext{}, combat.AbilityCheckContext{Ability: ability.Dexterity}, combat.AbilityCheckContext{Creature: cid("a")},
		combat.AbilityCheckContext{Creature: cid("a"), Ability: ability.Dexterity, Skill: content.Skill(-1)}}
	for i, rc := range bad {
		got, err := combat.QueryRollContributions([]combat.SpellModifierSource{smPlain(t, "haste", 0)}, rc)
		if !errors.Is(err, srderr.ErrInvalidInput) || !got.Equal(combat.SpellRollContributions{}) {
			t.Errorf("context %d: got %+v, %v, want empty and ErrInvalidInput", i, got, err)
		}
	}
	if _, err := combat.QueryRollContributions(nil, combat.AbilityCheckContext{Creature: cid("a"), Ability: ability.Dexterity}); err != nil {
		t.Errorf("raw check context = %v, want valid", err)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-007, FR-023, FR-024 - the output lists dice and circumstances in ascending CastOrder (equal orders keep the input order), the input list is not altered, equal inputs give equal outputs, the accessors return copies and Equal compares every list.
// Path: unhappy (determinism; immutability).
func TestRollContributionsOrderAndImmutability(t *testing.T) {
	t.Parallel()
	a := smSrc(t, "slow", "c", "a", 5, combat.NoChoice{})
	b := smSrc(t, "haste", "c", "a", 2, combat.NoChoice{})
	g := smSrc(t, "guidance", "c", "a", 3, combat.ChosenSkill{Skill: content.SkillStealth})
	in := []combat.SpellModifierSource{a, g, b}
	copyIn := slices.Clone(in)
	rc := combat.AbilityCheckContext{Creature: cid("a"), Ability: ability.Dexterity, Skill: content.SkillStealth}
	got, err := combat.QueryRollContributions(in, rc)
	if err != nil || !slices.Equal(in, copyIn) {
		t.Fatalf("QueryRollContributions = %v, input %v want %v", err, in, copyIn)
	}
	sv := combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}
	save, _ := combat.QueryRollContributions(in, sv)
	if want := []modifier.RollCircumstance{modifier.Advantage{}}; !slices.Equal(save.Circumstances(), want) || !slices.Equal(save.Modifiers(), []modifier.Modifier{modifier.New(-2)}) {
		t.Errorf("save contributions = %v %v", save.Circumstances(), save.Modifiers())
	}
	reordered, _ := combat.QueryRollContributions([]combat.SpellModifierSource{b, a, g}, sv)
	if !save.Equal(reordered) || !got.Equal(got) || save.Equal(got) {
		t.Error("equal inputs in another order must give equal contributions, and different ones unequal")
	}
	// Equal orders keep the input order: Haste's Advantage and Bestow Curse's Disadvantage on one save.
	c := smSrc(t, "bestow_curse", "d", "a", 2, combat.ChosenCurse{Effect: combat.CurseAbility{Ability: ability.Dexterity}})
	tie, _ := combat.QueryRollContributions([]combat.SpellModifierSource{c, b}, sv)
	tie2, _ := combat.QueryRollContributions([]combat.SpellModifierSource{b, c}, sv)
	if !slices.Equal(tie.Circumstances(), []modifier.RollCircumstance{modifier.Disadvantage{}, modifier.Advantage{}}) || !slices.Equal(tie2.Circumstances(), []modifier.RollCircumstance{modifier.Advantage{}, modifier.Disadvantage{}}) {
		t.Errorf("ties = %v and %v, want the input order", tie.Circumstances(), tie2.Circumstances())
	}
	d := got.BonusDice()
	d[0] = combat.BonusDie{}
	m, ci := save.Modifiers(), save.Circumstances()
	m[0], ci[0] = modifier.New(9), modifier.Disadvantage{}
	if got.BonusDice()[0].IsZero() || save.Modifiers()[0] != modifier.New(-2) || save.Circumstances()[0] != (modifier.Advantage{}) {
		t.Error("an accessor returned the contributions' own slice")
	}
	// Order by CastOrder across kinds: Slow's -2 (order 5) after Haste (2) in modifiers list order is irrelevant, so check the dice list instead.
	two := []combat.SpellModifierSource{smSrc(t, "guidance", "c", "a", 9, combat.ChosenSkill{Skill: content.SkillStealth})}
	if r, _ := combat.QueryRollContributions(two, rc); len(r.BonusDice()) != 1 {
		t.Errorf("one Guidance gave %d dice, want 1 (FR-014)", len(r.BonusDice()))
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-022, R15, FR-014 - the circumstances are handed on unresolved: Haste's Advantage and a Disadvantage from another source cancel as a whole at modifier.Resolve; a roll with Advantage still has one die per spell.
// Path: unhappy (rule interaction).
func TestCircumstancesAreHandedOnForResolve(t *testing.T) {
	t.Parallel()
	sv := combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}
	got, err := combat.QueryRollContributions([]combat.SpellModifierSource{smPlain(t, "haste", 0), smSrc(t, "bestow_curse", "d", "a", 1, combat.ChosenCurse{Effect: combat.CurseAbility{Ability: ability.Dexterity}})}, sv)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Circumstances()) != 2 || modifier.Resolve(got.Circumstances()...) != modifier.Normal {
		t.Errorf("Haste and the curse resolve to %v, want Normal", modifier.Resolve(got.Circumstances()...))
	}
	if m := modifier.Resolve(got.Circumstances()[:1]...); m != modifier.WithAdvantage {
		t.Errorf("Haste alone resolves to %v, want Advantage", m)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-022, section 9 Q4 - rule interactions between active spells and conditions: Restrained gives Disadvantage on Dexterity saves, so with Haste on the creature the two cancel at modifier.Resolve, and a Haste Advantage against an automatic failure leaves the failure to the conditions (SRD 5.2.1, "Haste", p. 139; "Restrained", p. 187; "Advantage", p. 176).
// Path: unhappy (rule interaction).
func TestSpellCircumstancesCombineWithConditions(t *testing.T) {
	t.Parallel()
	restrained := condSet(t, condConds(combat.Restrained{}))
	paralyzed := condSet(t, condConds(combat.Paralyzed{}))
	sv := combat.SavingThrowContext{Creature: cid("a"), Ability: ability.Dexterity}
	haste, err := combat.QueryRollContributions([]combat.SpellModifierSource{smPlain(t, "haste", 0)}, sv)
	if err != nil {
		t.Fatal(err)
	}
	effects := combat.ConditionEffects{}
	circ := haste.Circumstances()
	if effects.SavingThrow(restrained, ability.Dexterity) == combat.SavingThrowEffectDisadvantage {
		circ = append(circ, modifier.Disadvantage{})
	}
	if got := modifier.Resolve(circ...); got != modifier.Normal {
		t.Errorf("Haste and Restrained resolve to %v, want Normal", got)
	}
	if got := effects.SavingThrow(paralyzed, ability.Dexterity); got != combat.SavingThrowEffectAutomaticFailure || modifier.Resolve(haste.Circumstances()...) != modifier.WithAdvantage {
		t.Errorf("Paralyzed save = %v: the failure is the conditions' and Haste still gives Advantage", got)
	}
}

// Requirement: doc/specs/spell-granted-modifiers.md FR-001, FR-081 - BonusDie accepts a valid expression with a sign, rejects the zero expression with an error matching srderr.ErrInvalidInput, and the zero BonusDie is invalid; Must panics with the error.
// Path: unhappy (invalid input).
func TestBonusDie(t *testing.T) {
	t.Parallel()
	d, err := combat.NewBonusDie(dice.MustExpression(1, dice.D4(), 0), true)
	if err != nil || d.IsZero() || !d.Negative() || d.Expression().Notation() != "1d4" {
		t.Errorf("NewBonusDie = %+v, %v", d, err)
	}
	if z, err := combat.NewBonusDie(dice.Expression{}, false); !errors.Is(err, srderr.ErrInvalidInput) || !z.IsZero() {
		t.Errorf("NewBonusDie(zero) = %+v, %v", z, err)
	}
	if !(combat.BonusDie{}).IsZero() || !reflect.DeepEqual(smD4(), smD4()) {
		t.Error("zero BonusDie must be zero")
	}
	defer func() {
		if err, ok := recover().(error); !ok || !errors.Is(err, srderr.ErrInvalidInput) {
			t.Errorf("MustBonusDie panic = %v", err)
		}
	}()
	combat.MustBonusDie(dice.Expression{}, false)
}
