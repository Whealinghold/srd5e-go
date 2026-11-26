package combat_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/combat"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/dice/dicetest"
)

const setupSpec = "doc/specs/monster-combatant-setup.md "

// The monsters of these tests are synthetic: made-up names carrying the numbers the spec's case tables quote from
// the stat blocks of SRD 5.2.1 (Aboleth p. 258, Bat p. 345, Giant Rat p. 353 and others), never a stat block.

// msFields is a Large monster with the printed numbers of the first worked block: Initiative +7 (17), Hit Points 150
// (20d10 + 40), Speed 10 ft. and Swim 40 ft., Challenge Rating 10 (Proficiency Bonus +4), a Strength save of +5
// (not proficient), a Dexterity save of +3 (Dexterity -1, proficient), and Perception +10 and History +12 (twice the
// bonus on a +2 and a +4 ability).
func msFields(tb testing.TB) content.MonsterFields {
	tb.Helper()
	scores := map[ability.Ability]ability.Score{
		ability.Strength: ability.MustScore(21), ability.Dexterity: ability.MustScore(9), ability.Constitution: ability.MustScore(15),
		ability.Intelligence: ability.MustScore(18), ability.Wisdom: ability.MustScore(15), ability.Charisma: ability.MustScore(18),
	}
	saves := map[ability.Ability]int{
		ability.Strength: 5, ability.Dexterity: 3, ability.Constitution: 2, ability.Intelligence: 8, ability.Wisdom: 2, ability.Charisma: 4,
	}
	return content.MonsterFields{
		Sizes:      []content.CreatureSize{content.SizeLarge},
		Type:       content.CreatureTypeBeast,
		Alignment:  content.AlignmentUnaligned,
		ArmorClass: 17,
		Initiative: luMust(content.NewStatBlockInitiative(7, 17)),
		HitPoints:  luMust(content.NewHitPointsEntry(150, dice.MustExpression(20, dice.D10(), 40))),
		Speeds: []content.SpeedEntry{
			msSpeed(10, "", content.SpeedWalk), msSpeed(40, "", content.SpeedSwim),
		},
		Abilities:         luMust(content.NewAbilityBlock(luMust(ability.NewScores(scores)), saves)),
		Skills:            map[content.Skill]int{content.SkillPerception: 10, content.SkillHistory: 12},
		PassivePerception: 20,
		ChallengeRating:   content.MustChallengeRating("10"),
	}
}

func msSpeed(feet int, qualifier string, modes ...content.SpeedMode) content.SpeedEntry {
	return luMust(content.NewSpeedEntry(modes, feet, false, qualifier))
}

// msMonster builds a monster from msFields changed by f.
func msMonster(tb testing.TB, f func(*content.MonsterFields)) content.Monster {
	tb.Helper()
	fields := msFields(tb)
	if f != nil {
		f(&fields)
	}
	return luMust(content.NewMonster("fixture_beast", "Fixture Beast", content.MustSourceCitation("Fixture", content.MonsterSection, 258), fields))
}

// msHitPoints returns a monster whose printed Hit Points are average and the dice.
func msHitPoints(tb testing.TB, average int, expression dice.Expression) content.Monster {
	tb.Helper()
	return msMonster(tb, func(f *content.MonsterFields) {
		f.HitPoints = luMust(content.NewHitPointsEntry(average, expression))
	})
}

// msRolled rolls expression with the given faces and wraps the result as a Hit Points choice.
func msRolled(tb testing.TB, expression dice.Expression, faces ...int) combat.RolledMonsterHitPoints {
	tb.Helper()
	return combat.RolledMonsterHitPoints{Roll: luMust(dicetest.NewFixedRoller(faces...).Roll(expression))}
}

func msDamageDefense(t content.DamageType, qualifier string) content.DefenseEntry {
	return luMust(content.NewDamageTypeDefense(t, qualifier))
}

func msConditionDefense(c content.ConditionKind, qualifier string) content.DefenseEntry {
	return luMust(content.NewConditionDefense(c, qualifier))
}

func msDescribed(text string) content.DefenseEntry { return luMust(content.NewDescribedDefense(text)) }

func msSetup(tb testing.TB, m content.Monster, opts ...combat.SetupOption) combat.MonsterSetup {
	tb.Helper()
	return luMust(combat.NewMonsterSetup(m, combat.PrintedAverageHitPoints{}, opts...))
}
