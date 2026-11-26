package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// This file re-validates the numbers of a stat block that combat relies on (plan P13).
// Content validated them structurally, and its constructors reject the values below;
// combat checks them again so a monster built another way, or the zero Monster, is an
// invalid result and never reaches the rules. The ranges are the ones content gives
// (content-monsters FR-095); content gives no ceiling below content.MonsterMaxNumber,
// so none is invented here. The lengths of the part, spell and plan lists are capped by plan P10
// (limit_monster.go).

// monsterNumberOK reports whether v is a printed whole number a stat block may hold.
func monsterNumberOK(v int) bool {
	return v >= -content.MonsterMaxNumber && v <= content.MonsterMaxNumber
}

// monsterUsesOK reports whether n is a use count: an X/Day count, a spell frequency or
// the Legendary Actions of a stat block.
func monsterUsesOK(n int) bool { return n >= content.MonsterMinUses && n <= content.MonsterMaxNumber }

// monsterPerDayOK reports whether an X/Day count, or the Legendary Actions, and the in-Lair figure (0 for none printed) are use counts.
func monsterPerDayOK(count, inLair int) bool {
	return monsterUsesOK(count) && (inLair == 0 || monsterUsesOK(inLair))
}

// monsterSpeedFeetOK reports whether feet is a printed speed.
func monsterSpeedFeetOK(feet int) bool { return feet >= 0 && feet <= content.MonsterMaxNumber }

// monsterRechargeOK reports whether low to high is a Recharge range on a d6.
func monsterRechargeOK(low, high int) bool {
	return low >= content.MonsterMinRecharge && high <= content.MonsterMaxRecharge && low <= high
}

// monsterMultiattackCountOK reports whether n is the number of attacks of a Multiattack
// group or of the attacks one replacement rule replaces.
func monsterMultiattackCountOK(n int) bool {
	return n >= content.MonsterMinAttackCount && n <= content.MonsterMaxNumber
}

// monsterDiceOK reports whether count dice with the given sides are a stat block's dice:
// a count of 1 to dice.MaxCount and a die of content.MonsterHitPointDieFaces.
func monsterDiceOK(count, sides int) bool {
	return count >= content.MonsterMinHitDiceCount && count <= dice.MaxCount &&
		slices.Contains(content.MonsterHitPointDieFaces(), sides)
}

// monsterHitPointsOK reports whether the printed average and Hit Point Dice are usable.
func monsterHitPointsOK(average, count, sides int) bool {
	return average >= content.MonsterMinHitPointsAverage && average <= content.MonsterMaxNumber && monsterDiceOK(count, sides)
}

// monsterSpellLevelOK reports whether level is the level a spell of a stat block is
// cast at.
func monsterSpellLevelOK(level int) bool {
	return level >= content.MonsterMinSpellLevel && level <= content.MonsterMaxSpellLevel
}

// differenceFits reports whether printed minus abilityModifier fits an int.
func differenceFits(printed, abilityModifier int) bool {
	d := int64(printed) - int64(abilityModifier)
	return int64(int(d)) == d
}

// monsterNumbers is every number of a stat block that combat relies on, as plain values: the
// Hit Points and Hit Point Dice, the Initiative, the speeds, whether the save and skill
// modifiers split into an ability modifier and the rest without overflow, the Legendary
// Actions, and for each part its usage, Multiattack counts, spell frequencies and levels and
// damage dice. Content's constructors cannot build a stat block with a bad number, so the
// tests of each check build a monsterNumbers with the bad number directly.
type monsterNumbers struct {
	identified                            bool
	hitAverage, hitDiceCount, hitDieSides int
	initiativeModifier, initiativeScore   int
	speedFeet                             []int
	modifiersFit                          bool
	hasLegendary                          bool
	legendaryUses, legendaryInLair        int
	parts                                 []partNumbers
}

// partNumbers is the numbers of one part.
type partNumbers struct {
	usage            usageNumbers
	attackCounts     []int
	spellFrequencies []usageNumbers
	spellLevels      []int // the cast levels the entries print; an entry that prints none is absent
	damageDice       []diceNumbers
}

// usageKind is the shape of a usage or a spell frequency.
type usageKind int

const (
	usageInvalid  usageKind = iota // no usage, or a variant combat does not know
	usageFree                      // Unlimited, Other, Recharge after a Rest, At Will: no number
	usagePerDay                    // a use count and an in-Lair count (0: none printed)
	usageRecharge                  // a Recharge range
)

// usageNumbers is the numbers of a usage or a spell frequency: for usagePerDay a is the use
// count and b the in-Lair count, for usageRecharge a is the lowest face and b the highest.
type usageNumbers struct {
	kind usageKind
	a, b int
}

// diceNumbers is a count of dice with sides.
type diceNumbers struct{ count, sides int }

// monsterNumbersOf reads the numbers of m. The zero Monster is not identified.
func monsterNumbersOf(m content.Monster) monsterNumbers {
	if m.ID() == "" {
		return monsterNumbers{}
	}
	hp, in := m.HitPoints(), m.Initiative()
	n := monsterNumbers{identified: true, hitAverage: hp.Average(), hitDiceCount: hp.Dice().Count(), hitDieSides: hp.Dice().Die().Sides(),
		initiativeModifier: in.Modifier(), initiativeScore: in.Score(), modifiersFit: monsterModifiersFit(m)}
	for _, e := range m.Speeds() {
		n.speedFeet = append(n.speedFeet, e.Feet())
	}
	parts := slices.Concat(m.Traits(), m.Actions(), m.BonusActions(), m.Reactions())
	if l, ok := m.LegendaryActions(); ok {
		inLair, _ := l.UsesInLair()
		n.hasLegendary, n.legendaryUses, n.legendaryInLair = true, l.Uses(), inLair
		parts = slices.Concat(parts, l.Actions())
	}
	for _, p := range parts {
		n.parts = append(n.parts, partNumbersOf(p))
	}
	return n
}

// partNumbersOf reads the numbers of p.
func partNumbersOf(p content.StatBlockPart) partNumbers {
	n := partNumbers{usage: usageNumbersOf(p.Usage())}
	if plan, ok := p.Plan(); ok {
		n.attackCounts = planCounts(plan)
	}
	if sc, ok := p.Spellcasting(); ok {
		for _, g := range sc.Groups() {
			n.spellFrequencies = append(n.spellFrequencies, frequencyNumbersOf(g.Frequency()))
			for _, s := range g.Spells() {
				if level, ok := s.CastAtLevel(); ok {
					n.spellLevels = append(n.spellLevels, level)
				}
			}
		}
	}
	for _, c := range p.Clauses() {
		for _, d := range c.Damage() {
			if e, ok := d.Dice(); ok {
				n.damageDice = append(n.damageDice, diceNumbers{e.Count(), e.Die().Sides()})
			}
		}
	}
	return n
}

// usageNumbersOf reads the numbers of a part's usage.
func usageNumbersOf(u content.PartUsage) usageNumbers {
	switch v := u.(type) {
	case content.PartUnlimited, content.PartUsageOther, content.PartRechargeAfterRest:
		return usageNumbers{kind: usageFree}
	case content.PartPerDay:
		inLair, _ := v.InLairCount()
		return usageNumbers{usagePerDay, v.Count(), inLair}
	case content.PartRecharge:
		return usageNumbers{usageRecharge, v.Low(), v.High()}
	}
	return usageNumbers{}
}

// frequencyNumbersOf reads the numbers of a spell frequency.
func frequencyNumbersOf(f content.SpellFrequency) usageNumbers {
	switch v := f.(type) {
	case content.SpellFrequencyAtWill:
		return usageNumbers{kind: usageFree}
	case content.SpellFrequencyPerDay:
		return usageNumbers{usagePerDay, v.Count(), 0}
	case content.SpellFrequencyPerDayEach:
		return usageNumbers{usagePerDay, v.Count(), 0}
	}
	return usageNumbers{}
}

// monsterNumbersOK reports whether every number of m that combat relies on is in range
// (see monsterNumbers). The zero Monster is not in range.
func monsterNumbersOK(m content.Monster) bool { return monsterNumbersOf(m).ok() }

// ok reports whether every number of n is in range.
func (n monsterNumbers) ok() bool {
	return n.identified &&
		monsterHitPointsOK(n.hitAverage, n.hitDiceCount, n.hitDieSides) &&
		monsterNumberOK(n.initiativeModifier) && monsterNumberOK(n.initiativeScore) &&
		!slices.ContainsFunc(n.speedFeet, func(feet int) bool { return !monsterSpeedFeetOK(feet) }) &&
		n.modifiersFit &&
		(!n.hasLegendary || monsterPerDayOK(n.legendaryUses, n.legendaryInLair)) &&
		!slices.ContainsFunc(n.parts, func(p partNumbers) bool { return !p.ok() })
}

// ok reports whether every number of p is in range.
func (p partNumbers) ok() bool {
	return p.usage.ok() &&
		!slices.ContainsFunc(p.attackCounts, func(c int) bool { return !monsterMultiattackCountOK(c) }) &&
		!slices.ContainsFunc(p.spellFrequencies, func(f usageNumbers) bool { return !f.frequencyOK() }) &&
		!slices.ContainsFunc(p.spellLevels, func(l int) bool { return !monsterSpellLevelOK(l) }) &&
		!slices.ContainsFunc(p.damageDice, func(d diceNumbers) bool { return !monsterDiceOK(d.count, d.sides) })
}

// frequencyOK reports whether u is a spell frequency in range: At Will, or N a day with N a use count.
func (u usageNumbers) frequencyOK() bool {
	return u.kind == usageFree || (u.kind == usagePerDay && monsterUsesOK(u.a))
}

// ok reports whether the numbers of a usage are in range. A part with no usage is not.
func (u usageNumbers) ok() bool {
	switch u.kind {
	case usageInvalid:
		return false
	case usageFree:
		return true
	case usagePerDay:
		return monsterPerDayOK(u.a, u.b)
	case usageRecharge:
		return monsterRechargeOK(u.a, u.b)
	}
	return false
}

// monsterModifiersFit reports whether the ability block is present and each printed
// save and skill modifier differs from its ability modifier by an int.
func monsterModifiersFit(m content.Monster) bool {
	return modifiersFit(m.Abilities(), func(a ability.Ability) int { return m.AbilityModifier(a).Value() }, m.Skills(), differenceFits)
}

// modifiersFit is monsterModifiersFit over its parts: block must be present, and fits must hold for each
// printed save and skill modifier and its ability modifier. fits is a parameter so a test can force the
// overflow of a 32-bit int on any platform.
func modifiersFit(block content.AbilityBlock, modifier func(ability.Ability) int, skills map[content.Skill]int, fits func(printed, abilityModifier int) bool) bool {
	if block.IsZero() {
		return false
	}
	for _, a := range ability.All() {
		if !fits(block.Save(a), modifier(a)) {
			return false
		}
	}
	for sk, printed := range skills {
		if !fits(printed, modifier(sk.Ability())) {
			return false
		}
	}
	return true
}

// monsterUsageOK reports whether the numbers of a part's usage are in range. A part with
// no usage is not.
func monsterUsageOK(u content.PartUsage) bool { return usageNumbersOf(u).ok() }

// monsterPlanOK reports whether every Multiattack group and replacement count is in
// range.
func monsterPlanOK(plan content.MultiattackPlan) bool {
	return !slices.ContainsFunc(planCounts(plan), func(c int) bool { return !monsterMultiattackCountOK(c) })
}

// planCounts returns the group and replacement counts of every variant of plan.
func planCounts(plan content.MultiattackPlan) []int {
	var counts []int
	for _, v := range plan.Variants() {
		for _, g := range v.Groups() {
			counts = append(counts, g.Count())
		}
		for _, r := range v.Replacements() {
			counts = append(counts, r.Count())
		}
	}
	return counts
}
