package content_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// This file holds the offline verification checks of content-monsters section
// 4.11 as pure functions from one Monster to findings, and their unit tests
// on hand-built monsters. The checks are a test file, not shipped. The error
// checks (FR-072 to FR-075, FR-078) are for the data-level test that runs
// over the bundled set (step 6, not yet built); the advisory checks (FR-076,
// FR-077, FR-079, FR-096) and the name report (FR-080) give report lines and
// never fail a suite.

// finding is one line of a check's report: the check, the monster, its page
// and the detail.
func finding(check string, m content.Monster, detail string, args ...any) string {
	return fmt.Sprintf("%s: %s p. %d: %s", check, m.Name(), m.Source().Page(), fmt.Sprintf(detail, args...))
}

// checkInitiative is FR-072: the Initiative score is 10 plus the modifier.
func checkInitiative(m content.Monster) []string {
	in := m.Initiative()
	if in.Score() != 10+in.Modifier() {
		return []string{finding("FR-072", m, "Initiative %+d (%d)", in.Modifier(), in.Score())}
	}
	return nil
}

// checkHitPoints is FR-073: the flat modifier is the dice count times the
// Constitution modifier and the printed average is the average of the dice.
func checkHitPoints(m content.Monster) []string {
	hp := m.HitPoints()
	e := hp.Dice()
	var out []string
	if want := e.Count() * m.AbilityModifier(ability.Constitution).Value(); e.Modifier() != want {
		out = append(out, finding("FR-073", m, "Hit Point modifier %d, want %d", e.Modifier(), want))
	}
	if hp.Average() != e.Average() {
		out = append(out, finding("FR-073", m, "Hit Points average %d, want %d", hp.Average(), e.Average()))
	}
	return out
}

// checkSaves is FR-074: each save is the ability modifier or the modifier plus
// the Proficiency Bonus.
func checkSaves(m content.Monster) []string {
	var out []string
	for _, a := range ability.All() {
		mod, save := m.AbilityModifier(a).Value(), m.Abilities().Save(a)
		if save != mod && save != mod+m.ProficiencyBonus() {
			out = append(out, finding("FR-074", m, "%v save %+d, modifier %+d, Proficiency Bonus %+d", a, save, mod, m.ProficiencyBonus()))
		}
	}
	return out
}

// checkPassivePerception is FR-075: 10 plus the Perception bonus when listed,
// else 10 plus the Wisdom modifier.
func checkPassivePerception(m content.Monster) []string {
	bonus, listed := m.Skills()[content.SkillPerception]
	if !listed {
		bonus = m.AbilityModifier(ability.Wisdom).Value()
	}
	if m.PassivePerception() != 10+bonus {
		return []string{finding("FR-075", m, "Passive Perception %d, want %d", m.PassivePerception(), 10+bonus)}
	}
	return nil
}

// standardXP is the XP of each Challenge Rating (table 4.5a, SRD 5.2.1, p.
// 256), in rating order; CR 0 is 0 or 10.
func standardXP(cr content.ChallengeRating) []int {
	if cr.IsZero() {
		return []int{0, 10}
	}
	table := []int{
		0, 25, 50, 100, 200, 450, 700, 1100, 1800, 2300, 2900, 3900, 5000, 5900, 7200, 8400, 10000, 11500, 13000,
		15000, 18000, 20000, 22000, 25000, 33000, 41000, 50000, 62000, 75000, 90000, 105000, 120000, 135000, 155000,
	}
	return []int{table[crIndex(cr)]}
}

func crIndex(cr content.ChallengeRating) int {
	return slices.Index(content.AllChallengeRatings(), cr)
}

// checkXP is FR-078: the printed XP is allowed by the Challenge Rating row and
// the lair XP is the standard XP of the next rating up.
func checkXP(m content.Monster) []string {
	var out []string
	if !slices.Contains(standardXP(m.ChallengeRating()), m.ExperiencePoints()) {
		out = append(out, finding("FR-078", m, "XP %d at CR %v, allowed %v", m.ExperiencePoints(), m.ChallengeRating(), standardXP(m.ChallengeRating())))
	}
	if lair, ok := m.ExperiencePointsInLair(); ok {
		all := content.AllChallengeRatings()
		next := crIndex(m.ChallengeRating()) + 1
		if next >= len(all) || lair != standardXP(all[next])[0] {
			out = append(out, finding("FR-078", m, "lair XP %d at CR %v", lair, m.ChallengeRating()))
		}
	}
	return out
}

// monsterParts returns every part of the stat block.
func monsterParts(m content.Monster) []content.StatBlockPart {
	var out []content.StatBlockPart
	out = append(out, m.Traits()...)
	out = append(out, m.Actions()...)
	out = append(out, m.BonusActions()...)
	out = append(out, m.Reactions()...)
	if l, ok := m.LegendaryActions(); ok {
		out = append(out, l.Actions()...)
	}
	return out
}

// adviseRollBonuses is FR-076: each attack bonus is expected to be the
// Proficiency Bonus plus one ability modifier and each save DC 8 plus the
// Proficiency Bonus plus one ability modifier. Advisory.
func adviseRollBonuses(m content.Monster) []string {
	matches := func(value, extra int) bool {
		for _, a := range ability.All() {
			if value == extra+m.ProficiencyBonus()+m.AbilityModifier(a).Value() {
				return true
			}
		}
		return false
	}
	var out []string
	for _, p := range monsterParts(m) {
		if a, ok := p.Attack(); ok && !matches(a.Bonus(), 0) {
			out = append(out, finding("FR-076", m, "%s attack bonus %+d", p.Name(), a.Bonus()))
		}
		if s, ok := p.SavingThrow(); ok && !matches(s.DC(), 8) {
			out = append(out, finding("FR-076", m, "%s save DC %d", p.Name(), s.DC()))
		}
		if sc, ok := p.Spellcasting(); ok {
			if dc, ok := sc.SaveDC(); ok && !matches(dc, 8) {
				out = append(out, finding("FR-076", m, "%s spell save DC %d", p.Name(), dc))
			}
			if b, ok := sc.AttackBonus(); ok && !matches(b, 0) {
				out = append(out, finding("FR-076", m, "%s spell attack bonus %+d", p.Name(), b))
			}
		}
	}
	return out
}

// adviseDamageAverages is FR-077: each damage number is expected to be the
// average of its dice. Advisory.
func adviseDamageAverages(m content.Monster) []string {
	var out []string
	for _, p := range monsterParts(m) {
		for _, c := range p.Clauses() {
			for _, d := range c.Damage() {
				if e, ok := d.Dice(); ok && d.Number() != e.Average() {
					out = append(out, finding("FR-077", m, "%s damage %d, dice %v average %d", p.Name(), d.Number(), e, e.Average()))
				}
			}
		}
	}
	return out
}

// adviseSkills is FR-079: each skill bonus is expected to be the ability
// modifier plus the Proficiency Bonus. Advisory.
func adviseSkills(m content.Monster) []string {
	var out []string
	for s, bonus := range m.Skills() {
		if want := m.AbilityModifier(s.Ability()).Value() + m.ProficiencyBonus(); bonus != want {
			out = append(out, finding("FR-079", m, "%v %+d, want %+d", s, bonus, want))
		}
	}
	slices.Sort(out)
	return out
}

// sizeHitDie is the Hit Die of a size (Hit Dice by Size, SRD 5.2.1, p. 255).
func sizeHitDie(z content.CreatureSize) int {
	return content.MonsterHitPointDieFaces()[int(z)-1]
}

// adviseHitDie is FR-096: the Hit Point die is expected to be the die of one of
// the monster's size options. Advisory.
func adviseHitDie(m content.Monster) []string {
	faces := m.HitPoints().Dice().Die().Sides()
	for _, z := range m.Sizes() {
		if sizeHitDie(z) == faces {
			return nil
		}
	}
	return []string{finding("FR-096", m, "Hit Point die d%d, sizes %v", faces, m.Sizes())}
}

// unresolvedNames is FR-080, report only: a line per Spellcasting spell not in
// spellNames and per Gear item not in equipmentNames, and one skipped line for
// a kind whose name set is nil.
func unresolvedNames(m content.Monster, spellNames, equipmentNames map[string]bool) []string {
	var out []string
	if spellNames == nil {
		out = append(out, finding("FR-080", m, "skipped spells"))
	} else if sc, ok := m.Spellcasting(); ok {
		for _, g := range sc.Groups() {
			for _, s := range g.Spells() {
				if !spellNames[s.Name()] {
					out = append(out, finding("FR-080", m, "unresolvedSpell %s", s.Name()))
				}
			}
		}
	}
	if equipmentNames == nil {
		out = append(out, finding("FR-080", m, "skipped gear"))
	} else {
		for _, g := range m.Gear() {
			if !equipmentNames[g] {
				out = append(out, finding("FR-080", m, "unresolvedGear %s", g))
			}
		}
	}
	return out
}

// chkOpt overrides one member of the conforming check fixture.
type chkOpt func(*content.MonsterFields)

// chkMonster is a conforming CR 5 stat block (Proficiency Bonus +3): Strength
// 16, Constitution 14, Wisdom 12, the rest 10; Initiative +0 (10); 4d8+8 for
// an average of 26; the Strength save proficient; Passive Perception 11; XP
// 1800 and 2300 in Lair; an attack, a save and a spell DC that follow the
// advisory rules. The numbers are synthetic.
func chkMonster(t *testing.T, opts ...chkOpt) content.Monster {
	t.Helper()
	scores := map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(16), ability.Dexterity: ability.MustScore(10), ability.Constitution: ability.MustScore(14),
		ability.Intelligence: ability.MustScore(10), ability.Wisdom: ability.MustScore(12), ability.Charisma: ability.MustScore(10),
	}
	saves := map[ability.Ability]int{
		ability.Strength: 6, ability.Dexterity: 0, ability.Constitution: 2, ability.Intelligence: 0, ability.Wisdom: 1, ability.Charisma: 0,
	}
	damage := must(content.NewStatBlockDamage(9, dice.MustExpression(2, dice.D6(), 2), must(content.NewSpecificDamageType(content.DamageFire)), false))
	spells := []content.MonsterSpellEntry{must(content.NewMonsterSpellEntry("Light", "", 0))}
	f := content.MonsterFields{
		Sizes: []content.CreatureSize{content.SizeMedium}, Type: content.CreatureTypeBeast, Alignment: content.AlignmentUnaligned, ArmorClass: 14,
		Initiative: must(content.NewStatBlockInitiative(0, 10)),
		HitPoints:  must(content.NewHitPointsEntry(26, dice.MustExpression(4, dice.D8(), 8))),
		Speeds:     []content.SpeedEntry{monSpeed(t, 30)},
		Abilities:  must(content.NewAbilityBlock(must(ability.NewScores(scores)), saves)), PassivePerception: 11,
		ChallengeRating: content.MustChallengeRating("5"), ExperiencePoints: 1800, ExperiencePointsInLair: 2300,
		Skills: map[content.Skill]int{content.SkillStealth: 3}, Gear: []string{"Item"},
		Actions: []content.StatBlockPart{monPart(t, "Bite", content.StatBlockPartFields{
			Attack:      must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: 6, Reach: 5, HasReach: true})),
			SavingThrow: must(content.NewStatBlockSavingThrow(ability.Constitution, 14, "", "")),
			Clauses:     []content.EffectClause{must(content.NewEffectClause(content.ClauseHit, "x", nil, []content.StatBlockDamage{damage}))},
		})},
		Traits: []content.StatBlockPart{monPart(t, "Spellcasting", content.StatBlockPartFields{
			Spellcasting: must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
				Ability: ability.Wisdom, SaveDC: 12, HasSaveDC: true, AttackBonus: 4, HasAttackBonus: true,
				Groups: []content.MonsterSpellGroup{must(content.NewMonsterSpellGroup(content.SpellFrequencyAtWill{}, spells))},
			})),
		})},
	}
	for _, o := range opts {
		o(&f)
	}
	return must(content.NewMonster("fixture_beast", "Fixture Beast", monSource(), f))
}

func withInitiative(mod, score int) chkOpt {
	return func(f *content.MonsterFields) { f.Initiative = must(content.NewStatBlockInitiative(mod, score)) }
}

func withHitPoints(avg, count, sides, mod int) chkOpt {
	return func(f *content.MonsterFields) {
		f.HitPoints = must(content.NewHitPointsEntry(avg, dice.MustExpression(count, dice.MustDie(sides), mod)))
	}
}

func withSave(a ability.Ability, v int) chkOpt {
	return func(f *content.MonsterFields) {
		scores := f.Abilities.Scores()
		saves := map[ability.Ability]int{}
		for _, x := range ability.All() {
			saves[x] = f.Abilities.Save(x)
		}
		saves[a] = v
		f.Abilities = must(content.NewAbilityBlock(scores, saves))
	}
}

func withXP(cr string, xp, lair int) chkOpt {
	return func(f *content.MonsterFields) {
		f.ChallengeRating, f.ExperiencePoints, f.ExperiencePointsInLair = content.MustChallengeRating(cr), xp, lair
	}
}

// Requirement: monSpec FR-072, FR-081 - the Initiative check accepts score = 10 + modifier and finds a corrupted
// monster.
// Path: per case.
func TestCheckInitiative(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		m          content.Monster
		want       int
	}{
		{"conforming", "happy", chkMonster(t), 0},
		{"negative modifier", "unhappy (boundary)", chkMonster(t, withInitiative(-2, 8)), 0},
		{"score one above", "unhappy (boundary)", chkMonster(t, withInitiative(1, 12)), 1},
		{"corrupted (a Gray Ooze style disagreement)", "unhappy (invalid input)", chkMonster(t, withInitiative(-2, 13)), 1},
	}
	for _, c := range cases {
		if got := checkInitiative(c.m); len(got) != c.want {
			t.Errorf("%s: checkInitiative = %q, want %d finding(s) [FR-072, %s]", c.name, got, c.want, c.path)
		}
	}
}

// Requirement: monSpec FR-074, FR-081 - each save is the modifier or the modifier plus the Proficiency Bonus.
// Path: per case.
func TestCheckSaves(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		m          content.Monster
		want       int
	}{
		{"conforming", "happy", chkMonster(t), 0},
		{"modifier only", "unhappy (boundary)", chkMonster(t, withSave(ability.Strength, 3)), 0},
		{"modifier plus proficiency", "unhappy (boundary)", chkMonster(t, withSave(ability.Dexterity, 3)), 0},
		{"one above proficiency", "unhappy (boundary)", chkMonster(t, withSave(ability.Dexterity, 4)), 1},
		{"between the two", "unhappy (invalid input)", chkMonster(t, withSave(ability.Dexterity, 2)), 1},
		{"printed sign omission (a Young White Dragon style disagreement)", "unhappy (invalid input)", chkMonster(t, withSave(ability.Intelligence, 2)), 1},
		{"two corrupted", "unhappy (invalid input)", chkMonster(t, withSave(ability.Intelligence, 2), withSave(ability.Charisma, -3)), 2},
	}
	for _, c := range cases {
		if got := checkSaves(c.m); len(got) != c.want {
			t.Errorf("%s: checkSaves = %q, want %d finding(s) [FR-074, %s]", c.name, got, c.want, c.path)
		}
	}
}

// Requirement: monSpec FR-075, FR-081 - Passive Perception is 10 plus the Perception bonus when listed, else the
// Wisdom modifier.
// Path: per case.
func TestCheckPassivePerception(t *testing.T) {
	t.Parallel()
	withPerception := func(bonus, passive int) chkOpt {
		return func(f *content.MonsterFields) {
			f.Skills = map[content.Skill]int{content.SkillPerception: bonus}
			f.PassivePerception = passive
		}
	}
	cases := []struct {
		name, path string
		m          content.Monster
		want       int
	}{
		{"wisdom only", "happy", chkMonster(t), 0},
		{"listed skill", "happy", chkMonster(t, withPerception(7, 17)), 0},
		{"listed skill, passive from wisdom", "unhappy (rule interaction)", chkMonster(t, withPerception(7, 11)), 1},
		{"one off", "unhappy (boundary)", chkMonster(t, func(f *content.MonsterFields) { f.PassivePerception = 12 }), 1},
		{"corrupted", "unhappy (invalid input)", chkMonster(t, func(f *content.MonsterFields) { f.PassivePerception = 0 }), 1},
	}
	for _, c := range cases {
		if got := checkPassivePerception(c.m); len(got) != c.want {
			t.Errorf("%s: checkPassivePerception = %q, want %d finding(s) [FR-075, %s]", c.name, got, c.want, c.path)
		}
	}
}

// Requirement: monSpec FR-078, FR-082, FR-032, table 4.5a - printed XP is allowed by the CR row (0 or 10 at CR 0)
// and the lair XP is the standard XP of the next rating; a printed disagreement still constructs.
// Path: per case.
func TestCheckXP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		m          content.Monster
		want       int
	}{
		{"conforming", "happy", chkMonster(t), 0},
		{"CR 0 with 0", "unhappy (boundary)", chkMonster(t, withXP("0", 0, 0)), 0},
		{"CR 0 with 10", "unhappy (boundary)", chkMonster(t, withXP("0", 10, 0)), 0},
		{"CR 0 with 25", "unhappy (boundary)", chkMonster(t, withXP("0", 25, 0)), 1},
		{"CR 1/8 lair at 1/4", "unhappy (boundary)", chkMonster(t, withXP("1/8", 25, 50)), 0},
		{"CR 1/2 lair at 1", "unhappy (boundary)", chkMonster(t, withXP("1/2", 100, 200)), 0},
		{"CR 30 with a lair figure", "unhappy (boundary)", chkMonster(t, withXP("30", 155000, 200000)), 1},
		{"a printed disagreement (an Archmage style row)", "unhappy (invalid input)", chkMonster(t, withXP("12", 8000, 0)), 1},
		{"corrupted lair", "unhappy (invalid input)", chkMonster(t, withXP("5", 1800, 2400)), 1},
		{"both wrong", "unhappy (invalid input)", chkMonster(t, withXP("5", 1700, 2400)), 2},
	}
	for _, c := range cases {
		if got := checkXP(c.m); len(got) != c.want {
			t.Errorf("%s: checkXP = %q, want %d finding(s) [FR-078, %s]", c.name, got, c.want, c.path)
		}
	}
	if len(standardXP(content.MustChallengeRating("30"))) != 1 || standardXP(content.MustChallengeRating("30"))[0] != 155000 {
		t.Error("the XP table does not end at CR 30 [table 4.5a]")
	}
}

// Requirement: monSpec FR-076, FR-077, FR-079, FR-096 - the advisory checks give report lines for a monster that
// differs and none for one that follows the expectation.
// Path: per case.
func TestAdvisoryChecks(t *testing.T) {
	t.Parallel()
	if got := slices.Concat(adviseRollBonuses(chkMonster(t)), adviseDamageAverages(chkMonster(t)), adviseSkills(chkMonster(t)), adviseHitDie(chkMonster(t))); len(got) != 0 {
		t.Errorf("conforming monster findings = %q [FR-076, FR-077, FR-079, FR-096]", got)
	}
	proficient := chkMonster(t, func(f *content.MonsterFields) {
		f.Skills = map[content.Skill]int{content.SkillStealth: 3, content.SkillAthletics: 6}
	})
	if got := adviseSkills(proficient); len(got) != 0 {
		t.Errorf("skills that are ability modifier + PB: %q [FR-079]", got)
	}
	if got := adviseSkills(chkMonster(t, func(f *content.MonsterFields) { f.Skills = map[content.Skill]int{content.SkillStealth: 9} })); len(got) != 1 {
		t.Errorf("a skill off by six: %q, want one line [FR-079]", got)
	}
	// Attack bonus: PB 3 plus Strength 3 is +6; +7 is no combination.
	off := chkMonster(t, func(f *content.MonsterFields) {
		f.Actions = []content.StatBlockPart{monPart(t, "Bite", content.StatBlockPartFields{
			Attack:      must(content.NewStatBlockAttack(content.StatBlockAttackFields{Kind: content.AttackMelee, Bonus: 7})),
			SavingThrow: must(content.NewStatBlockSavingThrow(ability.Constitution, 10, "", "")),
		})}
		f.Traits = []content.StatBlockPart{monPart(t, "Spellcasting", content.StatBlockPartFields{
			Spellcasting: must(content.NewMonsterSpellcasting(content.MonsterSpellcastingFields{
				Ability: ability.Wisdom, SaveDC: 99, HasSaveDC: true, AttackBonus: 50, HasAttackBonus: true,
				Groups: []content.MonsterSpellGroup{leveledGroup(t, must(content.NewMonsterSpellEntry("Light", "", 0)))},
			})),
		})}
	})
	if got := adviseRollBonuses(off); len(got) != 4 {
		t.Errorf("attack bonus, save DC, spell DC and spell bonus off: %q, want 4 lines [FR-076]", got)
	}
	// Damage: 2d6+2 averages 9; 10 is printed.
	dmgOff := chkMonster(t, func(f *content.MonsterFields) {
		d := must(content.NewStatBlockDamage(10, dice.MustExpression(2, dice.D6(), 2), must(content.NewSpecificDamageType(content.DamageFire)), false))
		fixed := must(content.NewStatBlockDamage(3, dice.Expression{}, must(content.NewSpecificDamageType(content.DamageFire)), false))
		f.Actions = []content.StatBlockPart{monPart(t, "Bite", content.StatBlockPartFields{
			Clauses: []content.EffectClause{must(content.NewEffectClause(content.ClauseHit, "x", nil, []content.StatBlockDamage{d, fixed}))},
		})}
	})
	if got := adviseDamageAverages(dmgOff); len(got) != 1 {
		t.Errorf("damage 10 for 2d6+2 and a fixed 3: %q, want one line [FR-077]", got)
	}
	// Hit die: Medium is d8; a Large or Huge monster with d8 differs; sizes are options.
	if got := adviseHitDie(chkMonster(t, func(f *content.MonsterFields) { f.Sizes = []content.CreatureSize{content.SizeLarge} })); len(got) != 1 {
		t.Errorf("Large with d8: %q, want one line [FR-096]", got)
	}
	if got := adviseHitDie(chkMonster(t, func(f *content.MonsterFields) {
		f.Sizes = []content.CreatureSize{content.SizeLarge, content.SizeMedium}
	})); len(got) != 0 {
		t.Errorf("Large or Medium with d8: %q, want none [FR-096]", got)
	}
}

// Requirement: monSpec FR-080 - unresolvedNames reports an unknown spell and gear item as a line each, one skipped
// line for a kind whose name set is not supplied, and never fails.
// Path: per case.
func TestUnresolvedNames(t *testing.T) {
	t.Parallel()
	m := chkMonster(t)
	if got := unresolvedNames(m, map[string]bool{"Light": true}, map[string]bool{"Item": true}); len(got) != 0 {
		t.Errorf("all names known: %q [FR-080]", got)
	}
	got := unresolvedNames(m, map[string]bool{}, map[string]bool{})
	if len(got) != 2 {
		t.Errorf("no names known: %q, want an unresolvedSpell and an unresolvedGear line [FR-080]", got)
	}
	if got := unresolvedNames(m, nil, nil); len(got) != 2 {
		t.Errorf("no name sets: %q, want two skipped lines [FR-080]", got)
	}
	if got := unresolvedNames(m, nil, map[string]bool{"Item": true}); len(got) != 1 {
		t.Errorf("no spell set: %q, want one skipped line [FR-080]", got)
	}
}

// withPlainConstitution makes every score 10, so the Constitution modifier is +0.
func withPlainConstitution(f *content.MonsterFields) {
	f.Abilities = must(content.NewAbilityBlock(monScores(10), monSaves(0)))
	f.PassivePerception = 10
}

// Requirement: monSpec FR-073, FR-081 - the Hit Points check: modifier = count x Constitution modifier, average =
// dice average rounded down; a corrupted monster is found.
// Path: per case.
func TestCheckHitPoints(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, path string
		m          content.Monster
		want       int
	}{
		{"conforming", "happy", chkMonster(t), 0},
		{"average is rounded down", "unhappy (boundary)", chkMonster(t, withPlainConstitution, withHitPoints(10, 3, 6, 0)), 0},
		{"average one above the rounded down one", "unhappy (boundary)", chkMonster(t, withPlainConstitution, withHitPoints(11, 3, 6, 0)), 1},
		{"average one below", "unhappy (boundary)", chkMonster(t, withPlainConstitution, withHitPoints(9, 3, 6, 0)), 1},
		{"wrong modifier", "unhappy (invalid input)", chkMonster(t, withHitPoints(24, 4, 8, 6)), 1},
		{"wrong average", "unhappy (invalid input)", chkMonster(t, withHitPoints(30, 4, 8, 8)), 1},
		{"both wrong", "unhappy (invalid input)", chkMonster(t, withHitPoints(1, 4, 8, 0)), 2},
	}
	for _, c := range cases {
		if got := checkHitPoints(c.m); len(got) != c.want {
			t.Errorf("%s: checkHitPoints = %q, want %d finding(s) [FR-073, %s]", c.name, got, c.want, c.path)
		}
	}
}
