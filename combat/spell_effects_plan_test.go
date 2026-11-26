package combat_test

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// Requirement: spell-effects.md FR-042, FR-064, FR-065 - the plan and its evidence expose what the resolver decided, through accessors that return copies: the part and instance of each PlannedDamage and PlannedAmount, the targets, saves, rolled amounts and attack of each instance, the ability of a saving throw part, and the kind of each rolled amount.
// Path: happy.
func TestPlanAndEvidenceAccessors(t *testing.T) {
	t.Parallel()
	heal := seResolved(t, seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(4, 5))
	h := heal.Targets()[0].Healing()
	if len(h) != 1 || h[0].Amount() != 12 || h[0].PartIndex() != 0 || h[0].InstanceIndex() != 0 {
		t.Errorf("healing = %v, want 4 + 5 + 3 from part 0 instance 0", h)
	}
	if abil, isSave := heal.Parts()[0].SaveAbility(); isSave || abil != 0 {
		t.Errorf("an automatic part has no save ability: %v %v", abil, isSave)
	}
	am := heal.Parts()[0].Instances()[0].Amounts()
	if len(am) != 1 || am[0].Outcome() != 0 || am[0].Amount() != 12 || !slices.Equal(am[0].Faces(), []int{4, 5}) {
		t.Errorf("amounts = %v, want outcome 0, faces 4 and 5, amount 12", am)
	}
	fire := seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)), dicetest.NewFixedRoller(slices.Concat(seEightD6, []int{19})...))
	part := fire.Parts()[0]
	inst := part.Instances()[0]
	if abil, isSave := part.SaveAbility(); !isSave || abil != ability.Dexterity || part.Index() != 0 || inst.Index() != 0 {
		t.Errorf("part ability %v %v index %d/%d, want Dexterity of part 0 instance 0", abil, isSave, part.Index(), inst.Index())
	}
	if ts := inst.Targets(); len(ts) != 1 || ts[0] != seRef("a") {
		t.Errorf("instance targets = %v", ts)
	}
	saves := inst.Saves()
	rolled, ok := saves[0].Save().(combat.UnarmedStrikeSaveRolled)
	if !ok || saves[0].Target() != seRef("a") || rolled.Result().Roll().Natural() != 19 || rolled.Outcome() != d20.Success {
		t.Errorf("save evidence = %#v, want the rolled 19 of a", saves)
	}
	pd := seOnly(t, fire, 0)
	if pd.PartIndex() != 0 || pd.InstanceIndex() != 0 || pd.Damages()[0] != combat.MustDamage(28, content.DamageFire) {
		t.Errorf("planned damage = %v", pd)
	}
	if _, rolledAttack := inst.Attack(); rolledAttack || inst.Hit() || inst.CriticalHit() {
		t.Error("a save instance reports an attack")
	}
	if reason, miss := inst.AutomaticMiss(); miss || reason != 0 {
		t.Error("a save instance reports an automatic miss")
	}
	if !inst.Equal(inst) || inst.Equal(heal.Parts()[0].Instances()[0]) || !part.Equal(part) || part.Equal(heal.Parts()[0]) {
		t.Error("evidence equality")
	}
	var lost combat.SpellEffectsRejection
	if len(lost.Reasons()) != 0 || !lost.Equal(combat.SpellEffectsRejection{}) {
		t.Error("the zero rejection lists nothing")
	}
	rej, _ := seRun(t, seCommand(seHoldPerson(), seReport("hold_person", 3, 0), seTarget("a", nil), seTarget("b", nil), seTarget("c", nil)), dicetest.NewMinimumRoller())
	r := rej.(combat.SpellEffectsRejected)
	if r.Coverage() != content.CoverageComplete || !r.Rejection().Equal(r.Rejection()) || r.Rejection().Equal(combat.SpellEffectsRejection{}) {
		t.Errorf("rejection = %v coverage %v", r.Rejection(), r.Coverage())
	}
}

// Requirement: spell-effects.md FR-023, FR-064 - a Critical Hit's evidence carries the d20 result, the hit and the critical flags, and a missed attack plans only the OnMiss outcomes (none here), with its attack roll kept as evidence.
// Path: unhappy (boundary).
func TestAttackEvidence(t *testing.T) {
	t.Parallel()
	res := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), dicetest.NewFixedRoller(20, 3, 4))
	inst := res.Parts()[0].Instances()[0]
	atk, ok := inst.Attack()
	if !ok || atk.Roll().Natural() != 20 || !inst.Hit() || !inst.CriticalHit() || len(inst.Amounts()) != 1 || !slices.Equal(inst.Amounts()[0].Faces(), []int{3, 4}) {
		t.Errorf("evidence = %v %v hit %v critical %v amounts %v", atk, ok, inst.Hit(), inst.CriticalHit(), inst.Amounts())
	}
	if len(inst.Targets()) != 1 {
		t.Errorf("targets = %v", inst.Targets())
	}
}

// seTouchSpell is a level 1 melee spell attack for 1d8 Lightning.
func seTouchSpell() content.Spell {
	return seSpell("jolt_touch", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackMelee, []content.SpellOutcome{seDamage(seAmount(seDice("1d8"), nil), seFixedType(content.DamageLightning), content.AddsNothing)}, nil), seInstances(1, nil))), nil)
}

// Requirement: spell-effects.md FR-022, FR-087 - a melee spell attack is made as a MeleeSpellAttack: in reach it rolls and hits, beyond the reach of 5 feet it is the rejection SpellAttackNotPossible before any die.
// Path: unhappy (boundary).
func TestMeleeSpellAttackReach(t *testing.T) {
	t.Parallel()
	at := func(feet int) combat.SpellTarget {
		return seTarget("a", func(p *combat.SpellTargetParams) {
			p.AttackFacts = seFacts(func(f *combat.SpellAttackFactsParams) { f.Distance = feet })
		})
	}
	res := seResolved(t, seCommand(seTouchSpell(), seReport("jolt_touch", 1, 0), at(5)), dicetest.NewFixedRoller(15, 4))
	if d := seOnly(t, res, 0); d.Damages()[0] != combat.MustDamage(4, content.DamageLightning) {
		t.Errorf("damage = %v, want Lightning 4", d.Damages())
	}
	roller := dicetest.NewMinimumRoller()
	got := seRejected(t, seCommand(seTouchSpell(), seReport("jolt_touch", 1, 0), at(6)), roller)
	if !slices.Equal(got, []combat.SpellEffectsRejectionKind{combat.SpellAttackNotPossible}) || len(roller.Requests()) != 0 {
		t.Errorf("reasons %v requests %v, want SpellAttackNotPossible and no die", got, roller.Requests())
	}
}

// Requirement: spell-effects.md FR-042 - the evidence of a forced failure is equal to another forced failure and not to a rolled save.
// Path: unhappy (determinism).
func TestForcedSaveEvidenceEquality(t *testing.T) {
	t.Parallel()
	forced := seTarget("a", func(p *combat.SpellTargetParams) {
		p.Saves = map[ability.Ability]d20.SaveAttempt{ability.Dexterity: d20.AutomaticFailure{}}
	})
	cmd := func(tg combat.SpellTarget) combat.SpellEffectsResolved {
		return seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), tg), dicetest.NewFixedRoller(slices.Concat(seEightD6, []int{5})...))
	}
	a, b, rolled := cmd(forced), cmd(forced), cmd(seTarget("a", nil))
	if !a.Equal(b) || a.Equal(rolled) {
		t.Error("forced failures are equal to each other and not to a rolled save")
	}
}

// differ fails the test unless a and b are unequal in both directions and each equals itself.
func differ[T interface{ Equal(T) bool }](t *testing.T, what string, a, b T) {
	t.Helper()
	if a.Equal(b) || b.Equal(a) || !a.Equal(a) || !b.Equal(b) {
		t.Errorf("%s: values that differ compare equal, or a value differs from itself", what)
	}
}

func seFireballRun(t *testing.T, saves map[ability.Ability]d20.SaveAttempt, d20face int, who string) combat.SpellEffectsResolved {
	t.Helper()
	tg := seTarget(who, func(p *combat.SpellTargetParams) {
		if saves != nil {
			p.Saves = saves
		}
	})
	faces := slices.Concat(seEightD6, []int{d20face})
	if _, forced := saves[ability.Dexterity].(d20.RollAttempt); saves != nil && !forced {
		faces = seEightD6
	}
	return seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), tg), dicetest.NewFixedRoller(faces...))
}

// Requirement: spell-effects.md FR-060 - Equal compares every field, in both directions: results that differ in the half adjustment, the Hit Kind, the part, the instance, the damage, the faces rolled, the amount at equal faces, the outcome index, the targets, the saving throw ability, a forced reason, a forced or rolled save, the modifiers of a save, the healing, the Temporary Hit Points, the conditions, the perception, the coverage, the spell, the level or the attack are not equal.
// Path: unhappy (determinism, rule interaction).
func TestEqualSeesEveryDifference(t *testing.T) {
	t.Parallel()
	failed, saved := seFireballRun(t, nil, 5, "a"), seFireballRun(t, nil, 19, "a")
	differ(t, "half against full", failed.Targets()[0], saved.Targets()[0])
	differ(t, "half result", failed, saved)
	differ(t, "other target", failed.Targets()[0], seFireballRun(t, nil, 5, "b").Targets()[0])
	differ(t, "instances of other targets", failed.Parts()[0].Instances()[0], seFireballRun(t, nil, 5, "b").Parts()[0].Instances()[0])
	otherDice := seResolved(t, seCommand(seFireball(), seReport("fireball", 3, 0), seTarget("a", nil)), dicetest.NewFixedRoller(slices.Concat([]int{4, 3, 3, 4, 3, 4, 3, 4}, []int{5})...))
	differ(t, "faces at equal totals", failed.Parts()[0].Instances()[0], otherDice.Parts()[0].Instances()[0])
	differ(t, "faces result", failed, otherDice)
	choose := map[ability.Ability]d20.SaveAttempt{ability.Dexterity: d20.ChooseToFail{}}
	auto := map[ability.Ability]d20.SaveAttempt{ability.Dexterity: d20.AutomaticFailure{}}
	forcedA, forcedB := seFireballRun(t, choose, 0, "a"), seFireballRun(t, auto, 0, "a")
	differ(t, "forced against rolled", forcedA, failed)
	differ(t, "forced reasons", forcedA, forcedB)
	plus := map[ability.Ability]d20.SaveAttempt{ability.Dexterity: d20.RollAttempt{Modifiers: d20.NewRollModifiers(modifier.New(5))}}
	differ(t, "modifiers of a save", seFireballRun(t, plus, 5, "a"), failed)

	darts := seResolved(t, withAssignments(seCommand(seMagicMissile(), seReport("magic_missile", 2, 0), seTarget("a", nil)), "a", "a", "a", "a"), dicetest.NewFixedRoller(2, 2, 2, 2))
	d := darts.Targets()[0].Damage()
	differ(t, "instance of a PlannedDamage", d[0], d[1])
	differ(t, "instance of the evidence", darts.Parts()[0].Instances()[0], darts.Parts()[0].Instances()[1])

	both := seCommand(seSpell("twin_effect", 1, seModelWithZone(), nil), seReport("twin_effect", 1, 0), seTarget("a", nil))
	both.PartIndexes = []int{0, 1}
	two := seResolved(t, both, dicetest.NewFixedRoller(3, 3))
	differ(t, "part of a PlannedDamage", two.Targets()[0].Damage()[0], two.Targets()[0].Damage()[1])
	differ(t, "part of the evidence", two.Parts()[0], two.Parts()[1])
	hold := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), dicetest.NewFixedRoller(5))
	differ(t, "ability of a save part", hold.Parts()[0], failed.Parts()[0])

	hit := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), dicetest.NewFixedRoller(15, 5))
	crit := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), dicetest.NewFixedRoller(20, 5, 5))
	miss := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), dicetest.NewFixedRoller(2))
	other := seResolved(t, seCommand(seFireBolt(), seReport("fire_bolt", 0, 1), seWithFacts("a")), dicetest.NewFixedRoller(16, 5))
	differ(t, "hit against critical", hit, crit)
	differ(t, "hit against miss", hit, miss)
	differ(t, "miss instance", hit.Parts()[0].Instances()[0], miss.Parts()[0].Instances()[0])
	differ(t, "d20 faces", hit.Parts()[0].Instances()[0], other.Parts()[0].Instances()[0])
	differ(t, "attack against save", hit.Parts()[0].Instances()[0], failed.Parts()[0].Instances()[0])
	fixedBolt := seSpell("fixed_bolt", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster),
		sePart(seAttack(content.SpellAttackRanged, []content.SpellOutcome{seDamage(seAmount(seFixedAmount(6), nil), seFire, content.AddsNothing)}, nil), seInstances(1, nil))), nil)
	n := seResolved(t, seCommand(fixedBolt, seReport("fixed_bolt", 1, 0), seWithFacts("a")), dicetest.NewFixedRoller(15))
	c := seResolved(t, seCommand(fixedBolt, seReport("fixed_bolt", 1, 0), seWithFacts("a")), dicetest.NewFixedRoller(20))
	differ(t, "hit kind", n.Targets()[0].Damage()[0], c.Targets()[0].Damage()[0])
	differ(t, "hit kind effects", n.Targets()[0], c.Targets()[0])

	h1 := seResolved(t, seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(4, 4))
	h2 := seResolved(t, seCommand(seCureWounds(), seReport("cure_wounds", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(4, 5))
	differ(t, "healing", h1.Targets()[0], h2.Targets()[0])
	tp1 := seResolved(t, seCommand(seFalseLife(), seReport("false_life", 1, 0), seTarget("caster", nil)), dicetest.NewFixedRoller(2, 2))
	tp2 := seResolved(t, seCommand(seFalseLife(), seReport("false_life", 1, 0), seTarget("caster", nil)), dicetest.NewFixedRoller(2, 3))
	differ(t, "Temporary Hit Points", tp1.Targets()[0], tp2.Targets()[0])
	held := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), dicetest.NewFixedRoller(5))
	free := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", nil)), dicetest.NewFixedRoller(19))
	differ(t, "conditions", held.Targets()[0], free.Targets()[0])
	beast := seResolved(t, seCommand(seHoldPerson(), seReport("hold_person", 2, 0), seTarget("a", func(p *combat.SpellTargetParams) { p.CreatureType = content.CreatureTypeBeast })), dicetest.NewMinimumRoller())
	differ(t, "perception", beast.Targets()[0], free.Targets()[0])

	// Amount evidence: the same faces with another flat modifier, and the same amounts at another outcome index.
	flat := func(mod string, withCondition bool) combat.SpellEffectsResolved {
		outs := []content.SpellOutcome{seDamage(seAmount(seDice("1d6"+mod), nil), seFire, content.AddsNothing)}
		if withCondition {
			outs = slices.Concat([]content.SpellOutcome{luMust(content.NewSpellAppliesCondition(content.ConditionProne, content.ForSpellDuration{}))}, outs)
		}
		sp := seSpell("flat_burst", 1, seModel(content.CoverageComplete, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster), sePart(seAuto(outs...), seInstances(1, nil))), nil)
		return seResolved(t, seCommand(sp, seReport("flat_burst", 1, 0), seTarget("a", nil)), dicetest.NewFixedRoller(3))
	}
	differ(t, "amount at equal faces", flat("+1", false).Parts()[0].Instances()[0], flat("+2", false).Parts()[0].Instances()[0])
	differ(t, "outcome index", flat("+1", false).Parts()[0].Instances()[0], flat("+1", true).Parts()[0].Instances()[0])

	// The result itself: the spell, the level and the coverage.
	mk := func(id string, level int, cov content.EffectCoverage) combat.SpellEffectsResolved {
		heal := luMust(content.NewSpellRestoresHitPoints(seAmount(seFixedAmount(4), nil), content.AddsNothing))
		sp := seSpell(id, 2, seModel(cov, seChosen(content.SubjectCreature, seExactly(1), nil, content.MayIncludeCaster), sePart(seAuto(heal), seInstances(1, nil))), nil)
		return seResolved(t, seCommand(sp, seReport(id, level, 0), seTarget("a", nil)), dicetest.NewMinimumRoller())
	}
	base := mk("mend", 2, content.CoverageComplete)
	differ(t, "spell", base, mk("mender", 2, content.CoverageComplete))
	differ(t, "level", base, mk("mend", 3, content.CoverageComplete))
	differ(t, "coverage", base, mk("mend", 2, content.CoveragePartial))
	differ(t, "perception against none", combat.SpellEffectsRejection{}, luMust(rejectionOf(t)))
}

func rejectionOf(t *testing.T) (combat.SpellEffectsRejection, error) {
	t.Helper()
	res, err := seRun(t, seCommand(seHoldPerson(), seReport("hold_person", 3, 0), seTarget("a", nil), seTarget("b", nil), seTarget("c", nil)), dicetest.NewMinimumRoller())
	return res.(combat.SpellEffectsRejected).Rejection(), err
}
