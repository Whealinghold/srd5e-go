package character_test

import (
	"testing"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
)

// spChar is a Fighter of the given level with the given Constitution score.
func spChar(t *testing.T, level, con int) character.Character {
	t.Helper()
	return charAt(t, level, 10, 10, con, 10, 10, 10)
}

// spTraits builds the traits of species id with choices.
func spTraits(t *testing.T, id string, c character.SpeciesTraitChoices) character.SpeciesTraits {
	t.Helper()
	return must(character.NewSpeciesTraits(content.SpeciesID(id), c))
}

func spDragonborn(t *testing.T, a character.DraconicAncestry) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "dragonborn", character.DragonbornChoices{Ancestry: a})
}
func spDwarf(t *testing.T) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "dwarf", character.NoSpeciesChoices{})
}
func spHalfling(t *testing.T) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "halfling", character.NoSpeciesChoices{})
}
func spOrc(t *testing.T) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "orc", character.NoSpeciesChoices{})
}
func spHuman(t *testing.T) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "human", character.HumanChoices{Skill: content.SkillStealth})
}
func spElf(t *testing.T, l character.ElvenLineage) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "elf", character.ElfChoices{Lineage: l, Ability: ability.Wisdom, KeenSensesSkill: content.SkillInsight})
}
func spGnome(t *testing.T, l character.GnomishLineage) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "gnome", character.GnomeChoices{Lineage: l, Ability: ability.Intelligence})
}
func spGoliath(t *testing.T, a character.GiantAncestry) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "goliath", character.GoliathChoices{Ancestry: a})
}
func spTiefling(t *testing.T, l character.FiendishLegacy) character.SpeciesTraits {
	t.Helper()
	return spTraits(t, "tiefling", character.TieflingChoices{Legacy: l, Ability: ability.Charisma})
}

// spExpended returns t with u expended n times.
func spExpended(t *testing.T, tr character.SpeciesTraits, u character.SpeciesUse, n int) character.SpeciesTraits {
	t.Helper()
	return must(tr.WithExpended(u, n))
}

// spRest is the RestApplied of pc-1 for a rest of kind, or, with interrupted, for a Long
// Rest cut short that earned the Short Rest benefits.
func spRest(t *testing.T, kind character.RestKind, interrupted bool) character.RestApplied {
	t.Helper()
	id := character.MustIdentifier("pc-1")
	if !interrupted {
		return fgtRest(t, id, kind)
	}
	cmd := must(character.NewTakeRestCommand(id, must(character.NewRestReport(character.LongRest, character.StandardLongRest, stopped(character.PhysicalExertion, 3, 0))), 12, 0, nil, character.SpentHitDice{}))
	return character.NewRestTaker().TakeRest(t.Context(), cmd).(character.TakeRestAppliedAfterInterruption).Applied()
}

// spCharNamed is a level 5 Fighter with the given id and flat scores of 10.
func spCharNamed(t *testing.T, id string) character.Character {
	t.Helper()
	lv := levels(character.ClassLevel{Class: "fighter", Level: 5})
	return must(character.Restore(character.Parts{
		ID: character.MustIdentifier(id), ExperiencePoints: character.MustExperiencePoints(0), ClassLevels: lv, Species: "human", Background: "soldier",
		AbilityScores: flatScores(t, 10), HitPointMaximum: character.MustHitPointMaximum(50),
		HitDicePool:       character.MustHitDicePoolFromLevels(lv, map[content.ClassID]content.HitPointDie{"fighter": dieOf("fighter")}),
		HeroicInspiration: character.NotInspired,
	}))
}
