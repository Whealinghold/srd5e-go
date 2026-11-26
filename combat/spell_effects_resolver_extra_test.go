package combat_test

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

func seTableCount() content.CountScaling {
	return luMust(content.NewByCharacterLevelCount(luMust(content.NewCountLevelTable([]content.CountLevelEntry{{Level: 1, Count: 1}, {Level: 5, Count: 2}}))))
}

// seRaySickness: a level 1 spell with a Constitution save, 2d8 Poison and the Poisoned condition on a failure, half damage only on a success.
func seRaySickness() content.Spell {
	cond := luMust(content.NewSpellAppliesCondition(content.ConditionPoisoned, content.ForSpellDuration{}))
	return seSpell("ray_sick", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seSave(ability.Constitution, []content.SpellOutcome{seDamage(seAmount(seDice("2d8"), nil), seFixedType(content.DamagePoison), content.AddsNothing), cond}, content.HalfDamageOnSuccess), seInstances(1, nil))), nil)
}

// seTwoSaveParts: part 0 is a nothing-on-success save and part 1 an automatic one, for a creature area.
func seTwoSaveParts() content.Spell {
	dmg := func() content.SpellOutcome {
		return seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)
	}
	return seSpell("double_burst", 1, seModel(content.CoverageComplete, seArea(content.SubjectCreature, nil),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{dmg()}, seNothing), seInstances(1, nil)), sePart(seAuto(dmg()), seInstances(1, nil))), nil)
}

// seTwoDartParts: two automatic parts of two instances each.
func seTwoDartParts() content.Spell {
	dmg := func() content.SpellOutcome {
		return seDamage(seAmount(seDice("1d4"), nil), seFire, content.AddsNothing)
	}
	return seSpell("dart_pair", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, content.TargetsByInstance{}, nil, content.MayIncludeCaster),
		sePart(seAuto(dmg()), seInstances(2, nil)), sePart(seAuto(dmg()), seInstances(2, nil))), nil)
}

// seModBolt: a ranged spell attack for 1d6 that adds the ability modifier.
func seModBolt() content.Spell {
	return seSpell("mod_bolt", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsSpellcastingAbilityModifier)}, nil), seInstances(1, nil))), nil)
}

// Requirement: spell-effects.md FR-050 - a model that reads any count or amount from the character level needs ScalingLevel: the number of targets (exactly and up to), the instances of a part and an amount.
// Path: unhappy (invalid input).
func TestEveryTableNeedsScalingLevel(t *testing.T) {
	t.Parallel()
	attackPart := func() content.SpellEffectPart {
		return sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seDice("1d6"), nil), seFire, content.AddsNothing)}, nil), seInstances(1, nil))
	}
	exactly := luMust(content.NewExactlyTargets(1, seTableCount()))
	upTo := luMust(content.NewUpToTargets(1, seTableCount()))
	cases := map[string]content.Spell{
		"exactly table": seSpell("t1", 0, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, exactly, nil, content.MayIncludeCaster), attackPart()), nil),
		"up to table":   seSpell("t2", 0, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, upTo, nil, content.MayIncludeCaster), attackPart()), nil),
		"instances":     seEldritchBlast(),
		"amount":        seFireBolt(),
	}
	for name, spell := range cases {
		_, err := seRun(t, seCommand(spell, seReport(string(spell.ID()), 0, 0), seWithFacts("a")), dicetest.NewMinimumRoller())
		if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("%s: err = %v, want ErrInvalidSpellEffectsCommand", name, err)
		}
		if _, err := seRun(t, seCommand(spell, seReport(string(spell.ID()), 0, 5), seWithFacts("a")), dicetest.NewMinimumRoller()); errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("%s: err = %v with a ScalingLevel", name, err)
		}
	}
	// The number of targets grows with the character level.
	two := cases["exactly table"]
	got := seRejected(t, seCommand(two, seReport("t1", 0, 5), seWithFacts("a")), dicetest.NewMinimumRoller())
	if !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.WrongTargetCount}) {
		t.Errorf("one target for two: %v, want WrongTargetCount", got)
	}
}

// Requirement: spell-effects.md FR-050, FR-062, FR-098, FR-031 - the boundaries of the command checks: a report of another spell whichever id sorts first, a chosen Damage Type of -1, a part index one past the last part and far past it, an assignment for the part one past the last, and an assignment longer than the instances.
// Path: unhappy (boundary).
func TestCommandBoundaries(t *testing.T) {
	t.Parallel()
	dart := seTwoDartParts()
	base := func() combat.ResolveSpellEffectsCommand {
		return seCommand(dart, seReport("dart_pair", 1, 0), seTarget("a", nil), seTarget("b", nil))
	}
	assign := func(part int, ids ...string) combat.InstanceAssignment {
		var refs []combat.SpellTargetRef
		for _, id := range ids {
			refs = append(refs, seRef(id))
		}
		return luMust(combat.NewInstanceAssignment(part, refs))
	}
	invalid := map[string]combat.ResolveSpellEffectsCommand{
		"the report of a spell that sorts after": seCommand(seFireBolt(), seReport("fireball", 0, 1), seWithFacts("a")),
		"a chosen type of -1": func() combat.ResolveSpellEffectsCommand {
			c := seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a"))
			c.ChosenDamageType = -1
			return c
		}(),
		"a chosen type for a spell that offers none": func() combat.ResolveSpellEffectsCommand {
			c := base()
			c.ChosenDamageType = content.DamageFire
			return c
		}(),
		"a part index equal to the number of parts": func() combat.ResolveSpellEffectsCommand { c := base(); c.PartIndexes = []int{2}; return c }(),
		"a part index far past the parts":           func() combat.ResolveSpellEffectsCommand { c := base(); c.PartIndexes = []int{5}; return c }(),
		"an assignment for the part past the last": func() combat.ResolveSpellEffectsCommand {
			c := base()
			c.Assignments = []combat.InstanceAssignment{assign(2, "a", "b")}
			return c
		}(),
	}
	for name, cmd := range invalid {
		if _, err := seRun(t, cmd, dicetest.NewMinimumRoller()); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("%s: err = %v, want ErrInvalidSpellEffectsCommand", name, err)
		}
	}
	long := base()
	long.Assignments = []combat.InstanceAssignment{assign(0, "a", "a", "b"), assign(1, "a", "b")}
	if got := seRejected(t, long, dicetest.NewMinimumRoller()); !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.InstanceAssignmentMismatch}) {
		t.Errorf("three assignments for two instances: %v", got)
	}
	// Only the assignment of the other part exists: part 0 has none.
	other := base()
	other.Assignments = []combat.InstanceAssignment{assign(1, "a", "b")}
	if got := seRejected(t, other, dicetest.NewMinimumRoller()); !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.InstanceAssignmentMismatch}) {
		t.Errorf("assignment of the other part only: %v", got)
	}
	ok := base()
	ok.Assignments = []combat.InstanceAssignment{assign(0, "a", "b"), assign(1, "b", "a")}
	res := seResolved(t, ok, dicetest.NewFixedRoller(1, 2, 3, 4))
	if len(res.Targets()[0].Damage()) != 2 || res.Targets()[0].Damage()[1].PartIndex() != 1 {
		t.Errorf("each target takes a dart from each part: %v", res.Targets()[0].Damage())
	}
}

// Requirement: spell-effects.md FR-019 - the boundaries of the target count: fewer targets than an exact count, two creatures for a self spell, no target for a spell of up to some.
// Path: unhappy (boundary).
func TestTargetCountBoundaries(t *testing.T) {
	t.Parallel()
	wcc := []combat.SpellEffectsRejectionKind{combat.WrongTargetCount}
	cases := map[string]combat.ResolveSpellEffectsCommand{
		"no target for exactly one":               seCommand(seHarm(), seReport("harm", 6, 0)),
		"two targets for exactly one":             seCommand(seHarm(), seReport("harm", 6, 0), seTarget("a", nil), seTarget("b", nil)),
		"the caster and another for a self spell": seCommand(seFalseLife(), seReport("false_life", 1, 0), seTarget("caster", nil), seTarget("b", nil)),
	}
	for name, cmd := range cases {
		if got := seRejected(t, cmd, dicetest.NewMinimumRoller()); !slices.Equal(got, wcc) {
			t.Errorf("%s: %v, want WrongTargetCount", name, got)
		}
	}
}

// Requirement: spell-effects.md FR-017, FR-081 - the first resolved part to meet an invalid target decides its perception, and the valid target count of the targets signal leaves invalid targets out.
// Path: unhappy (rule interaction).
func TestFirstPartDecidesPerception(t *testing.T) {
	t.Parallel()
	cannot := seTarget("a", func(p *combat.SpellTargetParams) { p.Validity = combat.TargetCannotBeAffected })
	res := seResolved(t, seCommand(seTwoSaveParts(), seReport("double_burst", 1, 0), cannot), dicetest.NewMinimumRoller())
	if p, _ := res.Targets()[0].Perception(); p != combat.AppearedToSucceed {
		t.Errorf("perception = %v, want AppearedToSucceed from the first part", p)
	}
	rec, opts := recording()
	seResolved(t, seCommand(seTwoSaveParts(), seReport("double_burst", 1, 0), cannot, seTarget("b", nil)), dicetest.NewFixedRoller(2, 3, 4), opts...)
	if tg := rec.Meter.MeasurementsNamed("srd5e.combat.spell_effects.targets"); len(tg) != 1 || tg[0].Value != 1 {
		t.Errorf("targets = %+v, want 1 valid target of 2", tg)
	}
}

// Requirement: spell-effects.md FR-025, FR-037, FR-038 - a halved success applies the damage only and no condition; a failure applies both; the amounts of the evidence list only the outcomes that have an amount.
// Path: unhappy (rule interaction).
func TestHalfDamageOnSuccessAppliesNoCondition(t *testing.T) {
	t.Parallel()
	for name, c := range map[string]struct {
		d20            int
		wantConditions int
		wantHalf       bool
	}{"failed": {5, 1, false}, "saved": {19, 0, true}} {
		res := seResolved(t, seCommand(seRaySickness(), seReport("ray_sick", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(4, 4, c.d20))
		e := res.Targets()[0]
		if len(e.Conditions()) != c.wantConditions || len(e.Damage()) != 1 || (len(e.Damage()[0].Adjustments()) == 1) != c.wantHalf {
			t.Errorf("%s: effects %+v", name, e)
		}
		if am := res.Parts()[0].Instances()[0].Amounts(); len(am) != 1 || am[0].Outcome() != 0 {
			t.Errorf("%s: amounts = %v, want only the damage outcome", name, am)
		}
	}
	hold := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), dicetest.NewFixedRoller(5))
	if am := hold.Parts()[0].Instances()[0].Amounts(); len(am) != 0 {
		t.Errorf("a condition-only part has amounts %v", am)
	}
}

// Requirement: spell-effects.md FR-032, FR-023 - an attack's damage adds the spellcasting ability modifier once, whether or not it is a Critical Hit.
// Path: happy.
func TestAttackDamageAddsTheAbilityModifier(t *testing.T) {
	t.Parallel()
	cmd := seCommand(seModBolt(), seReport("mod_bolt", 1, 0), seWithFacts("a"))
	cmd.AbilityModifier = 3
	res := seResolved(t, cmd, dicetest.NewFixedRoller(15, 4))
	if d := seOnly(t, res, 0); d.Damages()[0].Amount() != 7 {
		t.Errorf("damage = %v, want 4 + 3", d.Damages())
	}
	res = seResolved(t, cmd, dicetest.NewFixedRoller(20, 4, 5))
	if d := seOnly(t, res, 0); d.Damages()[0].Amount() != 12 {
		t.Errorf("critical damage = %v, want 4 + 5 + 3", d.Damages())
	}
}

// Requirement: spell-effects.md FR-022, FR-023, FR-097 - a hit on a target that is not at the targeted location is a miss whatever the d20 shows, and a target whose conditions say "not automatic" takes a normal hit.
// Path: unhappy (rule interaction).
func TestMissOverrideAndExplicitNotAutomatic(t *testing.T) {
	t.Parallel()
	elsewhere := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) {
			f.Situation = combat.AttackSituation{Visibility: combat.TargetUnseen{TargetAtTargetedLocation: false}}
		})
	})
	res := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), elsewhere), dicetest.NewFixedRoller(19, 19))
	if inst := res.Parts()[0].Instances()[0]; inst.Hit() || len(res.Targets()[0].Damage()) != 0 {
		t.Error("an unseen target that is not at the location was hit")
	}
	normal := seTarget("a", func(p *combat.SpellTargetParams) {
		p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) { f.AutomaticCritical = combat.AutomaticCriticalNotAutomatic })
	})
	res = seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), normal), dicetest.NewFixedRoller(15, 5))
	if d := seOnly(t, res, 0); d.HitKind() != combat.NormalHit {
		t.Errorf("kind = %v, want NormalHit", d.HitKind())
	}
}

// Requirement: spell-effects.md FR-043 - a save part whose scaled amount cannot be built is a contradictory command, not a panic.
// Path: unhappy (boundary).
func TestSaveAmountScalingPastTheDiceLimit(t *testing.T) {
	t.Parallel()
	grows := seSpell("endless_burst", 1, seModel(content.CoverageComplete, seArea(content.SubjectCreature, nil),
		sePart(seSave(ability.Dexterity, []content.SpellOutcome{seDamage(seAmount(seDice("1d4"), sePerSlot(1, dice.MaxCount, 0)), seFire, content.AddsNothing)}, seNothing), seInstances(1, nil))), nil)
	_, err := seRun(t, seCommand(grows, seReport("endless_burst", 9, 0), seTarget("a", nil)), dicetest.NewMinimumRoller())
	if !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("err = %v, want ErrInvalidSpellEffectsCommand", err)
	}
	var be *content.BlockError
	if !errors.As(err, &be) {
		t.Errorf("err = %v, want the content error kept as the cause", err)
	}
}

// Requirement: spell-effects.md FR-032 - arithmetic at the limits of int: a fixed amount of math.MaxInt plus a positive modifier saturates instead of wrapping to a negative total, and a fixed amount of math.MaxInt is kept.
// Path: unhappy (boundary).
func TestFixedAmountSaturates(t *testing.T) {
	t.Parallel()
	heal := luMust(content.NewSpellRestoresHitPoints(seAmount(seFixedAmount(math.MaxInt), nil), content.AddsSpellcastingAbilityModifier))
	spell := seSpell("mega_heal", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster), sePart(seAuto(heal), seInstances(1, nil))), nil)
	cmd := seCommand(spell, seReport("mega_heal", 1, 0), seTarget("a", nil))
	cmd.AbilityModifier = 5
	res := seResolved(t, cmd, dicetest.NewMinimumRoller())
	if h := res.Targets()[0].Healing(); len(h) != 1 || h[0].Amount() != math.MaxInt {
		t.Errorf("healing = %v, want math.MaxInt", h)
	}
	cmd.AbilityModifier = -1
	res = seResolved(t, cmd, dicetest.NewMinimumRoller())
	if h := res.Targets()[0].Healing(); len(h) != 1 || h[0].Amount() != math.MaxInt-1 {
		t.Errorf("healing = %v, want math.MaxInt-1", h)
	}
}

// seAttackWith is a ranged spell attack whose OnHit and OnMiss lists are given.
func seAttackWith(id string, onHit, onMiss []content.SpellOutcome) content.Spell {
	return seSpell(id, 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, onHit, onMiss), seInstances(1, nil))), nil)
}

// Requirement: spell-effects.md FR-023, FR-032, FR-036, FR-037, FR-038 - an attack that hits plans its damage, its healing, its Temporary Hit Points and its condition together, each amount from its own outcome; an amount of the hit or of the miss that cannot be scaled is a contradictory command; a miss applies OnMiss.
// Path: unhappy (rule interaction).
func TestAttackWithSeveralOutcomeKinds(t *testing.T) {
	t.Parallel()
	heal := luMust(content.NewSpellRestoresHitPoints(seAmount(seDice("1d4"), nil), content.AddsNothing))
	temp := luMust(content.NewSpellGrantsTemporaryHitPoints(seAmount(seDice("1d6"), nil), content.AddsNothing))
	cond := luMust(content.NewSpellAppliesCondition(content.ConditionProne, content.ForSpellDuration{}))
	dmg := seDamage(seAmount(seDice("1d8"), nil), seFire, content.AddsNothing)
	spell := seAttackWith("mixed_bolt", []content.SpellOutcome{dmg, heal, cond, temp}, nil)
	res := seResolved(t, seCommand(spell, seReport("mixed_bolt", 1, 0), seWithFacts("a")), dicetest.NewFixedRoller(15, 6, 3, 5))
	e := res.Targets()[0]
	if d := e.Damage(); len(d) != 1 || d[0].Damages()[0].Amount() != 6 {
		t.Errorf("damage = %v, want 6", d)
	}
	if h, p := e.Healing(), e.TemporaryHitPoints(); len(h) != 1 || h[0].Amount() != 3 || len(p) != 1 || p[0].Amount() != 5 || len(e.Conditions()) != 1 {
		t.Errorf("healing %v temporary %v conditions %v, want 3, 5 and one", h, p, e.Conditions())
	}
	am := res.Parts()[0].Instances()[0].Amounts()
	if len(am) != 3 || am[0].Outcome() != 0 || am[1].Outcome() != 1 || am[2].Outcome() != 3 {
		t.Errorf("amounts = %v, want the outcomes 0, 1 and 3", am)
	}

	grows := func(d dice.Die) content.ScalableAmount {
		return seAmount(seDice("1d4"), sePerSlot(1, dice.MaxCount, 0))
	}
	badHeal := luMust(content.NewSpellRestoresHitPoints(grows(dice.D4()), content.AddsNothing))
	onHit := seAttackWith("bad_heal_bolt", []content.SpellOutcome{dmg, badHeal}, nil)
	if _, err := seRun(t, seCommand(onHit, seReport("bad_heal_bolt", 9, 0), seWithFacts("a")), dicetest.NewFixedRoller(15, 6)); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("a healing that cannot be scaled: err = %v", err)
	}
	badMiss := seDamage(grows(dice.D4()), seFire, content.AddsNothing)
	onMiss := seAttackWith("bad_miss_bolt", []content.SpellOutcome{dmg}, []content.SpellOutcome{badMiss})
	if _, err := seRun(t, seCommand(onMiss, seReport("bad_miss_bolt", 9, 0), seWithFacts("a")), dicetest.NewFixedRoller(2)); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("a miss damage that cannot be scaled: err = %v", err)
	}
	okMiss := seAttackWith("miss_cond_bolt", []content.SpellOutcome{dmg}, []content.SpellOutcome{cond})
	miss := seResolved(t, seCommand(okMiss, seReport("miss_cond_bolt", 1, 0), seWithFacts("a")), dicetest.NewFixedRoller(2))
	if len(miss.Targets()[0].Conditions()) != 1 || len(miss.Targets()[0].Damage()) != 0 {
		t.Errorf("a miss applies OnMiss only: %+v", miss.Targets()[0])
	}
}

// Requirement: spell-effects.md FR-043, FR-031, FR-050, FR-062 - the instance count that cannot be scaled, a chosen Damage Type below the first and above the last, and an assignment list for the other part only are all refused with the reason the spec names.
// Path: unhappy (boundary).
func TestMoreCommandBoundaries(t *testing.T) {
	t.Parallel()
	explodes := seSpell("many_darts", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, content.TargetsByInstance{}, nil, content.MayIncludeCaster),
		sePart(seAuto(seDamage(seAmount(seDice("1d4"), nil), seFire, content.AddsNothing)), seInstances(2, luMust(content.NewPerSlotLevelCount(1, math.MaxInt))))), nil)
	if _, err := seRun(t, seCommand(explodes, seReport("many_darts", 9, 0), seTarget("a", nil)), dicetest.NewMinimumRoller()); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
		t.Errorf("an instance count that overflows: err = %v", err)
	}
	for _, bad := range []content.DamageType{99, -1} {
		cmd := seCommand(seChromaticOrb(), seReport("chromatic_orb", 1, 0), seWithFacts("a"))
		cmd.ChosenDamageType = bad
		if _, err := seRun(t, cmd, dicetest.NewMinimumRoller()); !errors.Is(err, combat.ErrInvalidSpellEffectsCommand) {
			t.Errorf("chosen type %d: err = %v", bad, err)
		}
	}
	first := luMust(combat.NewInstanceAssignment(0, []combat.SpellTargetRef{seRef("a"), seRef("b")}))
	cmd := seCommand(seTwoDartParts(), seReport("dart_pair", 1, 0), seTarget("a", nil), seTarget("b", nil))
	cmd.Assignments = []combat.InstanceAssignment{first}
	if got := seRejected(t, cmd, dicetest.NewMinimumRoller()); !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.InstanceAssignmentMismatch}) {
		t.Errorf("an assignment for the first part only: %v, want a mismatch for the second", got)
	}
}
