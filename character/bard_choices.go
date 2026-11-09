package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BardExpertiseGain is the outcome of an accepted ChooseBardExpertise.
type BardExpertiseGain struct {
	proficiencies SkillProficiencies
	event         BardExpertiseGained
}

// Proficiencies returns the skill proficiencies with Expertise in the two chosen skills added.
func (g BardExpertiseGain) Proficiencies() SkillProficiencies { return g.proficiencies }

// Event returns the BardExpertiseGained event.
func (g BardExpertiseGain) Event() BardExpertiseGained { return g.event }

// ChooseBardExpertise validates the Expertise the Bard gains at level 2 and again at level 9 (SRD 5.2.1, "Level 2:
// Expertise", p. 32; class-bard.md FR-060 to FR-062) and adds it to proficiencies. The level at which it is owed is
// reported by BardPendingChoicesAt. Every problem is listed together, in this order: BardExpertiseCountWrong
// (not exactly two skills), BardExpertiseDuplicate, BardExpertiseNotProficient, BardExpertiseAlreadyHeld. With no
// problem the second result is empty and the first holds the new proficiencies; with any, the first is the zero
// value. proficiencies is not altered. id is the Bard's.
func ChooseBardExpertise(id Identifier, chosen []content.Skill, proficiencies SkillProficiencies) (BardExpertiseGain, []BardChoiceProblem) {
	var problems []BardChoiceProblem
	add := func(p BardChoiceProblem) { problems = appendProblem(problems, p) }
	if len(chosen) != 2 {
		add(BardExpertiseCountWrong)
	}
	for i, s := range chosen {
		switch {
		case slices.Contains(chosen[:i], s):
			add(BardExpertiseDuplicate)
		case !proficiencies.Proficient(s):
			add(BardExpertiseNotProficient)
		case proficiencies.HasExpertise(s):
			add(BardExpertiseAlreadyHeld)
		}
	}
	if len(problems) != 0 {
		return BardExpertiseGain{}, problems
	}
	for _, s := range chosen {
		proficiencies.expertise |= 1 << s
	}
	return BardExpertiseGain{proficiencies: proficiencies, event: BardExpertiseGained{id: id, skills: slices.Clone(chosen)}}, []BardChoiceProblem{}
}

// appendProblem appends p unless it is already listed.
func appendProblem(list []BardChoiceProblem, p BardChoiceProblem) []BardChoiceProblem {
	if slices.Contains(list, p) {
		return list
	}
	return append(list, p)
}

// BonusProficienciesProblems checks the College of Lore's Bonus Proficiencies (SRD 5.2.1, "Level 3: Bonus
// Proficiencies", p. 35; class-bard.md FR-070, FR-071, FR-110): exactly three distinct skills, none of them one the
// character is already proficient in. Every problem is listed together, in this order: BardSkillCountWrong,
// BardSkillDuplicate, BardSkillAlreadyProficient, BardSkillInvalid. It is empty when the choice is valid.
func BonusProficienciesProblems(chosen []content.Skill, proficiencies SkillProficiencies) []BardChoiceProblem {
	problems := []BardChoiceProblem{}
	if len(chosen) != 3 {
		problems = append(problems, BardSkillCountWrong)
	}
	for i, s := range chosen {
		if slices.Contains(chosen[:i], s) {
			problems = appendProblem(problems, BardSkillDuplicate)
		}
	}
	if slices.ContainsFunc(chosen, proficiencies.Proficient) {
		problems = append(problems, BardSkillAlreadyProficient)
	}
	if slices.ContainsFunc(chosen, func(s content.Skill) bool { return !s.Valid() }) {
		problems = append(problems, BardSkillInvalid)
	}
	return problems
}

// BardSubclassProblems checks a Bard subclass choice (SRD 5.2.1, "Level 3: Bard Subclass", p. 32; class-bard.md
// FR-066, FR-068): BardSubclassTooEarly below Bard level 3, and BardSubclassAlreadyChosen when held is not empty. It
// is empty when the choice may be made.
func BardSubclassProblems(level BardLevel, held content.SubclassID) []BardChoiceProblem {
	problems := []BardChoiceProblem{}
	if level.n < bardSubclassLevel {
		problems = append(problems, BardSubclassTooEarly)
	}
	if held != "" {
		problems = append(problems, BardSubclassAlreadyChosen)
	}
	return problems
}

// magicalDiscoveriesLists are the spell lists the College of Lore's Magical Discoveries draws on: not the Bard list
// (SRD 5.2.1, "Level 6: Magical Discoveries", p. 35; class-bard.md FR-076).
func magicalDiscoveriesLists() []content.ClassID {
	return []content.ClassID{"cleric", "druid", "wizard"}
}

// MagicalDiscoveriesProblems checks the two spells of Magical Discoveries (class-bard.md FR-076, FR-077) for a Bard of
// level: exactly two, no repeat, each on the Cleric, Druid or Wizard list (a Bard-only spell is refused, a spell on the
// Bard list and another of the three is accepted) and each a cantrip or of a level the Bard has slots for. Every
// problem is listed together, in this order: BardSpellCountWrong, BardSpellDuplicate, BardSpellNotOnList,
// BardSpellWithoutSlots. It is empty when the choice is valid. The spells themselves are looked up by the caller.
func MagicalDiscoveriesProblems(level BardLevel, chosen []content.Spell) []BardChoiceProblem {
	problems := []BardChoiceProblem{}
	if len(chosen) != 2 {
		problems = append(problems, BardSpellCountWrong)
	}
	for i, s := range chosen {
		if slices.ContainsFunc(chosen[:i], func(o content.Spell) bool { return o.ID() == s.ID() }) {
			problems = appendProblem(problems, BardSpellDuplicate)
		}
	}
	if slices.ContainsFunc(chosen, func(s content.Spell) bool { return !onAnyList(s, magicalDiscoveriesLists()) }) {
		problems = appendProblem(problems, BardSpellNotOnList)
	}
	if slices.ContainsFunc(chosen, func(s content.Spell) bool { return int(s.Level()) > MaxPreparableLevel(CasterBard, level.n) }) {
		problems = appendProblem(problems, BardSpellWithoutSlots)
	}
	return problems
}

func onAnyList(s content.Spell, lists []content.ClassID) bool {
	return slices.ContainsFunc(s.Classes(), func(c content.ClassID) bool { return slices.Contains(lists, c) })
}

// BardPreparedSpellLists returns the spell lists a Bard of level may choose a prepared spell from (SRD 5.2.1, "Level
// 10: Magical Secrets", p. 33; class-bard.md FR-088, FR-090, FR-091, FR-111): the Bard list below level 10; from level
// 10 the Bard, Cleric, Druid and Wizard lists for a replacement, and for an addition only at a level where the Prepared
// Spells number of the Features table increases (levels 10 and 11, not 12). replacing says the spell takes the place of
// one prepared before. The result is a new slice.
func BardPreparedSpellLists(level BardLevel, replacing bool) []content.ClassID {
	if level.n < bardMagicalSecretsLevel {
		return []content.ClassID{BardClassID}
	}
	if !replacing && PreparedLimit(CasterBard, level.n) <= PreparedLimit(CasterBard, level.n-1) {
		return []content.ClassID{BardClassID}
	}
	return []content.ClassID{BardClassID, "cleric", "druid", "wizard"}
}

// PreparedSpellProblems checks the prepared spells a Bard of level chooses (class-bard.md FR-055, FR-091): no repeat,
// each on a list BardPreparedSpellLists allows, and each of a level the Bard has slots for. The count against the
// Prepared Spells number is the generic spellcasting rule's. Every problem is listed together, in this order:
// BardSpellDuplicate, BardSpellNotOnList, BardSpellWithoutSlots. It is empty when the choice is valid.
func PreparedSpellProblems(level BardLevel, chosen []content.Spell, replacing bool) []BardChoiceProblem {
	problems := []BardChoiceProblem{}
	lists := BardPreparedSpellLists(level, replacing)
	for i, s := range chosen {
		if slices.ContainsFunc(chosen[:i], func(o content.Spell) bool { return o.ID() == s.ID() }) {
			problems = appendProblem(problems, BardSpellDuplicate)
		}
	}
	if slices.ContainsFunc(chosen, func(s content.Spell) bool { return !onAnyList(s, lists) }) {
		problems = appendProblem(problems, BardSpellNotOnList)
	}
	if slices.ContainsFunc(chosen, func(s content.Spell) bool { return int(s.Level()) > MaxPreparableLevel(CasterBard, level.n) }) {
		problems = appendProblem(problems, BardSpellWithoutSlots)
	}
	return problems
}

// BardAlwaysPreparedSpells returns the spells a Bard of level always has prepared, which do not count against the
// Prepared Spells number (SRD 5.2.1, "Level 6: Magical Discoveries", "Level 20: Words of Creation", pp. 33, 35;
// class-bard.md FR-057, FR-078, FR-095): the Magical Discoveries spells from level 6 (discoveries, as the College of
// Lore Bard holds them) and Power Word Heal and Power Word Kill from level 20, each once, as a new slice.
func BardAlwaysPreparedSpells(level BardLevel, discoveries []content.SpellID) []content.SpellID {
	out := []content.SpellID{}
	add := func(ids ...content.SpellID) {
		for _, id := range ids {
			if !slices.Contains(out, id) {
				out = append(out, id)
			}
		}
	}
	if level.n >= bardMagicalDiscoveryLevel {
		add(discoveries...)
	}
	if level.n >= bardWordsOfCreationLevel {
		add(wordsOfCreationSpells()...)
	}
	return out
}
