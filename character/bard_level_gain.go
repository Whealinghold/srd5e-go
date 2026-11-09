package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// BardFeaturesGainedAt returns the ids of the features the Bard gains on reaching level (SRD 5.2.1, "Bard Features",
// p. 31; "College of Lore", p. 35; class-bard.md FR-069, FR-097): the class features of that level in printed order,
// then the features of that level of the subclass held (the College of Lore's Bonus Proficiencies and Cutting Words
// at 3, Magical Discoveries at 6, Peerless Skill at 14). Levels are gained one at a time (FR-115), so a feature of a
// lower level never comes owed here. held is the subclass the Bard holds once the level is gained, or empty. The result
// is a new slice.
//
// It returns an error matching srderr.ErrInvalidInput for a definition that is not the Bard's, a zero level, or a held
// subclass the definition does not list.
func BardFeaturesGainedAt(def content.ClassDefinition, held content.SubclassID, level BardLevel) ([]content.FeatureID, error) {
	if def.ID() != BardClassID || level.IsZero() {
		return nil, invalidInput("BardFeaturesGainedAt: not the Bard definition, or a zero level")
	}
	out := []content.FeatureID{}
	entries, _ := def.EntriesGainedAt(level.n) // the level is 1 to 20
	for _, e := range entries {
		out = append(out, e.ID())
	}
	if held == "" {
		return out, nil
	}
	found, ok := def.Subclass(held).(content.SubclassFound)
	if !ok {
		return nil, invalidInput("BardFeaturesGainedAt: unknown Bard subclass " + string(held))
	}
	for _, e := range found.Subclass().Features() {
		if e.Level() == level.n {
			out = append(out, e.ID())
		}
	}
	return out, nil
}

// BardChosenSubclassProblems checks the subclass a Bard chooses on gaining level (class-bard.md FR-066, FR-067,
// FR-068): the problems of BardSubclassProblems, BardSubclassMissing for an empty chosen and BardSubclassUnknown for an
// id the Bard definition does not list (never a missing-content outcome: a subclass id has no content key). held is the
// subclass already held, or empty. It is empty when the choice is valid.
func BardChosenSubclassProblems(def content.ClassDefinition, level BardLevel, held, chosen content.SubclassID) []BardChoiceProblem {
	problems := BardSubclassProblems(level, held)
	if chosen == "" {
		return append(problems, BardSubclassMissing)
	}
	if _, ok := def.Subclass(chosen).(content.SubclassFound); !ok {
		problems = append(problems, BardSubclassUnknown)
	}
	return problems
}

// MagicalDiscoveriesReplacement replaces the held Magical Discoveries spell Out with the spell In.
type MagicalDiscoveriesReplacement struct {
	Out content.SpellID
	In  content.Spell
}

// MagicalDiscoveriesReplacementProblems checks the replacement a College of Lore Bard of level makes on gaining a
// level (SRD 5.2.1, "Level 6: Magical Discoveries", p. 35; class-bard.md FR-079): at most one, of a spell held in held,
// for a spell not already held, on the Cleric, Druid or Wizard list and a cantrip or of a level the Bard has slots for.
// Every problem is listed together, in this order: BardReplacementTooMany, BardReplacedNotHeld, BardSpellDuplicate,
// BardSpellNotOnList, BardSpellWithoutSlots. No replacement is valid and gives an empty result.
func MagicalDiscoveriesReplacementProblems(level BardLevel, held []content.SpellID, replacements []MagicalDiscoveriesReplacement) []BardChoiceProblem {
	problems := []BardChoiceProblem{}
	if len(replacements) > 1 {
		problems = append(problems, BardReplacementTooMany)
	}
	any := func(bad func(MagicalDiscoveriesReplacement) bool) bool { return slices.ContainsFunc(replacements, bad) }
	if any(func(r MagicalDiscoveriesReplacement) bool { return !slices.Contains(held, r.Out) }) {
		problems = append(problems, BardReplacedNotHeld)
	}
	if any(func(r MagicalDiscoveriesReplacement) bool { return slices.Contains(held, r.In.ID()) }) {
		problems = append(problems, BardSpellDuplicate)
	}
	if any(func(r MagicalDiscoveriesReplacement) bool { return !onAnyList(r.In, magicalDiscoveriesLists()) }) {
		problems = append(problems, BardSpellNotOnList)
	}
	if any(func(r MagicalDiscoveriesReplacement) bool {
		return int(r.In.Level()) > MaxPreparableLevel(CasterBard, level.n)
	}) {
		problems = append(problems, BardSpellWithoutSlots)
	}
	return problems
}

// BardLevelChoices are the Bard choices a player supplies for one gained level. A field the level does not owe is
// ignored.
type BardLevelChoices struct {
	// Expertise: the two skills of Expertise, at levels 2 and 9.
	Expertise []content.Skill
	// Subclass: the subclass chosen at level 3.
	Subclass content.SubclassID
	// BonusSkills: the College of Lore's three skills, at level 3.
	BonusSkills []content.Skill
	// Feat: the feat of levels 4, 8, 12, 16 and 19. Whether it exists and qualifies is the feat service's.
	Feat content.FeatID
	// Discoveries: the College of Lore's two spells, at level 6.
	Discoveries []content.Spell
}

// BardLevelChoiceProblems checks every choice the Bard level owes at once (class-bard.md FR-098): the choices of
// BardPendingChoicesAt, each by the function of this package that checks it, with BardFeatMissing for an owed feat that
// is empty. The cantrip and the prepared spells are the generic spellcasting rules'. All problems are listed together,
// each once, in the order of the pending choices; the result is empty when the level's choices are complete and valid.
// held is the subclass held before the level, or empty.
func BardLevelChoiceProblems(def content.ClassDefinition, level BardLevel, held content.SubclassID, proficiencies SkillProficiencies, choices BardLevelChoices) []BardChoiceProblem {
	after := held
	if after == "" {
		after = choices.Subclass
	}
	problems := []BardChoiceProblem{}
	add := func(list []BardChoiceProblem) {
		for _, p := range list {
			problems = appendProblem(problems, p)
		}
	}
	for _, p := range BardPendingChoicesAt(level, after) {
		switch p {
		case BardPendingExpertise:
			_, list := ChooseBardExpertise(Identifier{}, choices.Expertise, proficiencies)
			add(list)
		case BardPendingSubclass:
			add(BardChosenSubclassProblems(def, level, held, choices.Subclass))
		case BardPendingBonusProficiencies:
			add(BonusProficienciesProblems(choices.BonusSkills, proficiencies))
		case BardPendingFeat, BardPendingEpicBoon:
			if choices.Feat == "" {
				add([]BardChoiceProblem{BardFeatMissing})
			}
		case BardPendingMagicalDiscoveries:
			add(MagicalDiscoveriesProblems(level, choices.Discoveries))
		case BardPendingCantrip: // the generic spellcasting rule's
		}
	}
	return problems
}
