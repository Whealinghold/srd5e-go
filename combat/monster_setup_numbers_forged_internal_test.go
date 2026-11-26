package combat

import (
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// goodNumbers is a monsterNumbers every check of which passes, with one entry of each kind.
func goodNumbers() monsterNumbers {
	return monsterNumbers{
		identified: true, hitAverage: 9, hitDiceCount: 2, hitDieSides: 8,
		initiativeModifier: 1, initiativeScore: 11, speedFeet: []int{30, 40}, modifiersFit: true,
		hasLegendary: true, legendaryUses: 3, legendaryInLair: 4,
		parts: []partNumbers{{
			usage: usageNumbers{usagePerDay, 3, 4}, attackCounts: []int{2, 1},
			spellFrequencies: []usageNumbers{{kind: usageFree}, {usagePerDay, 2, 0}}, spellLevels: []int{1, 9},
			damageDice: []diceNumbers{{2, 6}, {1, 20}},
		}, {usage: usageNumbers{usageRecharge, 5, 6}}, {usage: usageNumbers{kind: usageFree}}},
	}
}

// Requirement: doc/specs/monster-combatant-setup.md section 8 (FR-052), plan-phase-4.md P13 - every branch of the stat block number check rejects its own bad value: each row forges one number outside its range (content's constructors cannot build it, so the numbers are built directly), and the unchanged base passes, so deleting a branch fails exactly one row.
// Path: per case.
func TestMonsterNumbersRejectEachBadNumber(t *testing.T) {
	t.Parallel()
	above, below := outsideNumbers()
	withPart := func(edit func(*partNumbers)) func(*monsterNumbers) {
		return func(n *monsterNumbers) {
			n.parts = slices.Clone(n.parts)
			edit(&n.parts[0])
		}
	}
	cases := []struct {
		name string
		edit func(*monsterNumbers)
	}{
		{"the zero Monster", func(n *monsterNumbers) { n.identified = false }},
		{"Hit Points average of 0", func(n *monsterNumbers) { n.hitAverage = 0 }},
		{"Hit Points average above the maximum", func(n *monsterNumbers) { n.hitAverage = above }},
		{"no Hit Point Dice", func(n *monsterNumbers) { n.hitDiceCount = 0 }},
		{"more Hit Point Dice than dice.MaxCount", func(n *monsterNumbers) { n.hitDiceCount = dice.MaxCount + 1 }},
		{"a Hit Point Die of 7 faces", func(n *monsterNumbers) { n.hitDieSides = 7 }},
		{"Initiative modifier above the maximum", func(n *monsterNumbers) { n.initiativeModifier = above }},
		{"Initiative modifier below the minimum", func(n *monsterNumbers) { n.initiativeModifier = below }},
		{"Initiative score above the maximum", func(n *monsterNumbers) { n.initiativeScore = above }},
		{"Initiative score below the minimum", func(n *monsterNumbers) { n.initiativeScore = below }},
		{"a negative speed", func(n *monsterNumbers) { n.speedFeet = []int{30, -1} }},
		{"a speed above the maximum", func(n *monsterNumbers) { n.speedFeet = []int{above} }},
		{"modifiers that overflow", func(n *monsterNumbers) { n.modifiersFit = false }},
		{"Legendary Actions of 0", func(n *monsterNumbers) { n.legendaryUses = 0 }},
		{"Legendary Actions above the maximum", func(n *monsterNumbers) { n.legendaryUses = above }},
		{"Legendary Actions in Lair negative", func(n *monsterNumbers) { n.legendaryInLair = -1 }},
		{"a part with no usage", withPart(func(p *partNumbers) { p.usage = usageNumbers{} })},
		{"an X/Day of 0", withPart(func(p *partNumbers) { p.usage = usageNumbers{usagePerDay, 0, 0} })},
		{"an X/Day in Lair above the maximum", withPart(func(p *partNumbers) { p.usage = usageNumbers{usagePerDay, 1, above} })},
		{"a Recharge of 0-6", withPart(func(p *partNumbers) { p.usage = usageNumbers{usageRecharge, 0, 6} })},
		{"a reversed Recharge", withPart(func(p *partNumbers) { p.usage = usageNumbers{usageRecharge, 6, 5} })},
		{"a Multiattack group of 0 attacks", withPart(func(p *partNumbers) { p.attackCounts = []int{2, 0} })},
		{"a Multiattack group above the maximum", withPart(func(p *partNumbers) { p.attackCounts = []int{above} })},
		{"a spell frequency of no kind", withPart(func(p *partNumbers) { p.spellFrequencies = []usageNumbers{{}} })},
		{"a spell frequency of 0 a day", withPart(func(p *partNumbers) { p.spellFrequencies = []usageNumbers{{usagePerDay, 0, 0}} })},
		{"a spell frequency above the maximum", withPart(func(p *partNumbers) { p.spellFrequencies = []usageNumbers{{usagePerDay, above, 0}} })},
		{"a Recharge as a spell frequency", withPart(func(p *partNumbers) { p.spellFrequencies = []usageNumbers{{usageRecharge, 5, 6}} })},
		{"a spell cast at level 0", withPart(func(p *partNumbers) { p.spellLevels = []int{0} })},
		{"a spell cast at level 10", withPart(func(p *partNumbers) { p.spellLevels = []int{10} })},
		{"damage of 0 dice", withPart(func(p *partNumbers) { p.damageDice = []diceNumbers{{0, 6}} })},
		{"damage of more than dice.MaxCount dice", withPart(func(p *partNumbers) { p.damageDice = []diceNumbers{{dice.MaxCount + 1, 6}} })},
		{"damage of a die of 7 faces", withPart(func(p *partNumbers) { p.damageDice = []diceNumbers{{1, 7}} })},
		{"a later part with a bad usage", func(n *monsterNumbers) {
			n.parts = slices.Clone(n.parts)
			n.parts[2].usage = usageNumbers{usageRecharge, 7, 7}
		}},
	}
	if !goodNumbers().ok() {
		t.Fatal("the base numbers are rejected")
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			n := goodNumbers()
			c.edit(&n)
			if n.ok() {
				t.Errorf("numbers with %s are accepted, want rejected (plan P13)", c.name)
			}
		})
	}
	t.Run("no Legendary Actions and no parts are in range", func(t *testing.T) {
		t.Parallel()
		n := goodNumbers()
		n.hasLegendary, n.legendaryUses, n.parts = false, 0, nil
		if !n.ok() {
			t.Error("a monster with no Legendary Actions and no parts is rejected")
		}
	})
}

// Requirement: doc/specs/monster-combatant-setup.md section 8 (FR-052), plan-phase-4.md P13 - the numbers are read from the stat block faithfully: Hit Points, Initiative, speeds, Legendary Actions, usages, Multiattack counts, spell frequencies and levels and damage dice of a built monster arrive in monsterNumbers unchanged.
// Path: happy.
func TestMonsterNumbersOfReadsTheStatBlock(t *testing.T) {
	t.Parallel()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	part := func(name string, f content.StatBlockPartFields) content.StatBlockPart {
		p, err := content.NewStatBlockPart(name, f)
		must(err)
		return p
	}
	scores := map[ability.Ability]ability.Score{}
	saves := map[ability.Ability]int{}
	for _, a := range ability.All() {
		scores[a], saves[a] = ability.MustScore(10), 0
	}
	sc, err := ability.NewScores(scores)
	must(err)
	block, err := content.NewAbilityBlock(sc, saves)
	must(err)
	initiative, err := content.NewStatBlockInitiative(1, 11)
	must(err)
	hp, err := content.NewHitPointsEntry(9, dice.MustExpression(2, dice.D8(), 0))
	must(err)
	walk, err := content.NewSpeedEntry([]content.SpeedMode{content.SpeedWalk}, 30, false, "")
	must(err)
	fly, err := content.NewSpeedEntry([]content.SpeedMode{content.SpeedFly}, 40, false, "")
	must(err)
	perDay, err := content.NewPartPerDay(3, 4)
	must(err)
	recharge, err := content.NewPartRecharge(5, 6)
	must(err)
	group, err := content.NewAttackSlotGroup(2, []string{"Claw"})
	must(err)
	variant, err := content.NewMultiattackVariant("", []content.AttackSlotGroup{group}, nil, nil)
	must(err)
	plan, err := content.NewMultiattackPlan([]content.MultiattackVariant{variant})
	must(err)
	dmg, err := content.NewStatBlockDamage(0, dice.MustExpression(2, dice.D6(), 0), mustDamageType(t), false)
	must(err)
	clause, err := content.NewEffectClause(content.ClauseHit, "T.", nil, []content.StatBlockDamage{dmg})
	must(err)
	freq, err := content.NewSpellFrequencyPerDay(2)
	must(err)
	entry, err := content.NewMonsterSpellEntry("Fixture Spell", "", 3)
	must(err)
	spellGroup, err := content.NewMonsterSpellGroup(freq, []content.MonsterSpellEntry{entry})
	must(err)
	casting, err := content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{Ability: ability.Wisdom, Groups: []content.MonsterSpellGroup{spellGroup}})
	must(err)
	legendary, err := content.NewLegendaryActions(3, 4, []content.StatBlockPart{part("Move", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T."})})
	must(err)
	m, err := content.NewMonster("fixture_beast", "Fixture Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeMedium}, Type: content.CreatureTypeBeast, Alignment: content.AlignmentUnaligned, ArmorClass: 12,
		Initiative: initiative, HitPoints: hp, Speeds: []content.SpeedEntry{walk, fly}, Abilities: block, PassivePerception: 10,
		Traits: []content.StatBlockPart{part("Spellcasting", content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Spellcasting: casting})},
		Actions: []content.StatBlockPart{part("Claw", content.StatBlockPartFields{Usage: perDay, Text: "T.", Clauses: []content.EffectClause{clause}}), part("Breath", content.StatBlockPartFields{Usage: recharge, Text: "T."}),
			part(content.MultiattackName, content.StatBlockPartFields{Usage: content.PartUnlimited{}, Text: "T.", Plan: plan})},
		LegendaryActions: legendary,
	})
	must(err)
	got := monsterNumbersOf(m)
	want := monsterNumbers{
		identified: true, hitAverage: 9, hitDiceCount: 2, hitDieSides: 8, initiativeModifier: 1, initiativeScore: 11,
		speedFeet: []int{30, 40}, modifiersFit: true, hasLegendary: true, legendaryUses: 3, legendaryInLair: 4,
		parts: []partNumbers{
			{usage: usageNumbers{kind: usageFree}, spellFrequencies: []usageNumbers{{usagePerDay, 2, 0}}, spellLevels: []int{3}},
			{usage: usageNumbers{usagePerDay, 3, 4}, damageDice: []diceNumbers{{2, 6}}},
			{usage: usageNumbers{usageRecharge, 5, 6}},
			{usage: usageNumbers{kind: usageFree}, attackCounts: []int{2}},
			{usage: usageNumbers{kind: usageFree}},
		},
	}
	if !equalNumbers(got, want) {
		t.Errorf("monsterNumbersOf = %+v, want %+v", got, want)
	}
	if !got.ok() || !monsterNumbersOK(m) || monsterNumbersOK(content.Monster{}) {
		t.Error("the built monster is not in range, or the zero Monster is")
	}
}

func equalNumbers(a, b monsterNumbers) bool {
	return a.identified == b.identified && a.hitAverage == b.hitAverage && a.hitDiceCount == b.hitDiceCount && a.hitDieSides == b.hitDieSides &&
		a.initiativeModifier == b.initiativeModifier && a.initiativeScore == b.initiativeScore && slices.Equal(a.speedFeet, b.speedFeet) &&
		a.modifiersFit == b.modifiersFit && a.hasLegendary == b.hasLegendary && a.legendaryUses == b.legendaryUses && a.legendaryInLair == b.legendaryInLair &&
		slices.EqualFunc(a.parts, b.parts, func(x, y partNumbers) bool {
			return x.usage == y.usage && slices.Equal(x.attackCounts, y.attackCounts) && slices.Equal(x.spellFrequencies, y.spellFrequencies) &&
				slices.Equal(x.spellLevels, y.spellLevels) && slices.Equal(x.damageDice, y.damageDice)
		})
}

func mustDamageType(t *testing.T) content.DamageTypeRef {
	t.Helper()
	ref, err := content.NewSpecificDamageType(content.DamageFire)
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
