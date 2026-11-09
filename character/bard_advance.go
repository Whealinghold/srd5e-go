package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// BardPendingChoice is a Bard-specific choice a level adds, a closed set of values (class-bard.md FR-060, FR-061, FR-066,
// FR-070, FR-076, FR-092, FR-094). The zero value is not a choice. Replacing a prepared spell, a cantrip or a Magical
// Discoveries spell is allowed at every level gained and is never owed, so it is not listed.
type BardPendingChoice int

const (
	// BardPendingExpertise: Expertise in two skills, at Bard levels 2 and 9.
	BardPendingExpertise BardPendingChoice = iota + 1
	// BardPendingSubclass: the Bard subclass, at Bard level 3.
	BardPendingSubclass
	// BardPendingBonusProficiencies: the College of Lore's three skills, at Bard level 3 once the subclass is chosen.
	BardPendingBonusProficiencies
	// BardPendingFeat: a feat, at Bard levels 4, 8, 12 and 16.
	BardPendingFeat
	// BardPendingEpicBoon: an Epic Boon feat or another feat, at Bard level 19.
	BardPendingEpicBoon
	// BardPendingCantrip: one more cantrip, at Bard levels 4 and 10.
	BardPendingCantrip
	// BardPendingMagicalDiscoveries: the College of Lore's two spells, at Bard level 6.
	BardPendingMagicalDiscoveries
)

// String returns "expertise", "subclass", "bonus_proficiencies", "feat", "epic_boon", "cantrip",
// "magical_discoveries" or "unknown".
func (c BardPendingChoice) String() string {
	names := []string{"expertise", "subclass", "bonus_proficiencies", "feat", "epic_boon", "cantrip", "magical_discoveries"}
	if c < 1 || int(c) > len(names) {
		return "unknown"
	}
	return names[c-1]
}

// BardPendingChoicesAt returns the choices that gaining Bard level l owes, in the order of BardPendingChoice, as a new
// slice. subclass is the Bard subclass the character holds when it gains the level, or empty: the College of Lore's
// level 6 spells are owed to a College of Lore Bard only, and its level 3 skills are listed when the subclass is held.
func BardPendingChoicesAt(l BardLevel, subclass content.SubclassID) []BardPendingChoice {
	out := []BardPendingChoice{}
	lore := subclass == CollegeOfLoreSubclassID
	if l.n == 2 || l.n == 9 {
		out = append(out, BardPendingExpertise)
	}
	if l.n == bardSubclassLevel {
		out = append(out, BardPendingSubclass)
		if lore {
			out = append(out, BardPendingBonusProficiencies)
		}
	}
	if l.n%4 == 0 && l.n <= 16 {
		out = append(out, BardPendingFeat)
	}
	if l.n == 19 {
		out = append(out, BardPendingEpicBoon)
	}
	if l.n == 4 || l.n == bardMagicalSecretsLevel {
		out = append(out, BardPendingCantrip)
	}
	if l.n == bardMagicalDiscoveryLevel && lore {
		out = append(out, BardPendingMagicalDiscoveries)
	}
	return out
}

// AdvanceBardResult is the closed set of outcomes of AdvanceBard. Variants: AdvanceBardApplied, AdvanceBardNotABard.
type AdvanceBardResult interface{ isAdvanceBardResult() }

// AdvanceBardApplied is the outcome of a Bard level gained. Only AdvanceBard builds one.
type AdvanceBardApplied struct {
	character Character
	level     BardLevel
	pending   []BardPendingChoice
	prepared  int
	cantrips  int
	always    []content.SpellID
}

// Character returns the Character holding the Bardic Inspiration pool: seeded full at the first Bard level, and with
// its maximum recomputed from the Charisma modifier at a later one.
func (r AdvanceBardApplied) Character() Character { return r.character }

// Level returns the new Bard level.
func (r AdvanceBardApplied) Level() BardLevel { return r.level }

// BardicDie returns the Bardic Die at the new level.
func (r AdvanceBardApplied) BardicDie() BardicDie { return BardicDieAt(r.level) }

// PendingChoices returns the Bard-specific choices the level owes, as a new slice.
func (r AdvanceBardApplied) PendingChoices() []BardPendingChoice { return slices.Clone(r.pending) }

// PreparedSpellsGained returns how many more prepared spells the Features table asks for at the new level, 0 when the
// number does not rise (class-bard.md FR-054).
func (r AdvanceBardApplied) PreparedSpellsGained() int { return r.prepared }

// CantripsGained returns how many more cantrips the Features table asks for at the new level, 0 or 1 (FR-051).
func (r AdvanceBardApplied) CantripsGained() int { return r.cantrips }

// AlwaysPreparedSpellsGranted returns Power Word Heal and Power Word Kill at level 20 and nothing at any other level,
// as a new slice (FR-095).
func (r AdvanceBardApplied) AlwaysPreparedSpellsGranted() []content.SpellID {
	return slices.Clone(r.always)
}

// AdvanceBardNotABard is the outcome for a character that holds no Bard level at all.
type AdvanceBardNotABard struct{}

func (AdvanceBardApplied) isAdvanceBardResult()  {}
func (AdvanceBardNotABard) isAdvanceBardResult() {}

// AdvanceBard applies the Bard side of a gained level (class-bard.md FR-097, FR-115, FR-121). It is called after level
// advancement has recorded the new level, so c already holds the new Bard level and previous is the Bard level before
// it (0 for the first). It gives the Character a full Bardic Inspiration pool at the first level and recomputes the
// maximum of a stored pool from the Charisma modifier of the Character's ability scores at a later one, then reports the
// choices the level owes, the prepared spells and cantrips the Features table adds and the spells always prepared. It
// loads no content and takes no spell: the choices are checked by the functions of this package and applied through the
// spellcasting state. A level gain from Bard level 20 cannot reach it, because no Character holds Bard level 21.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, a previous level that is not one below the
// new level, or another type of state under the pool key. There is no rejection.
func AdvanceBard(c Character, previous int, subclass content.SubclassID) (AdvanceBardResult, error) {
	if c.IsZero() {
		return nil, invalidInput("AdvanceBard: zero Character")
	}
	level, ok := bardLevelOf(c)
	if !ok {
		return AdvanceBardNotABard{}, nil
	}
	if previous != level.n-1 {
		return nil, invalidInput("AdvanceBard: previous Bard level " + strconv.Itoa(previous) + " is not one below " + strconv.Itoa(level.n))
	}
	charisma := c.AbilityScores().Modifier(ability.Charisma)
	stored, held, err := ReadFeatureState(c.FeatureStates(), BardicInspirationPoolKey())
	if err != nil || (!held && previous > 0) {
		return nil, firstError(err, invalidInput("a Bard gaining a level holds no Bardic Inspiration pool"))
	}
	pool := stored.WithMaximumFrom(charisma)
	if !held {
		pool = InitialBardicInspirationPool(charisma)
	}
	next, err := WithFeatureState(c, BardicInspirationPoolKey(), pool)
	if err != nil { // unreachable: the key is valid and a held state of another type was reported above
		return nil, err
	}
	out := AdvanceBardApplied{character: next, level: level, pending: BardPendingChoicesAt(level, subclass), always: []content.SpellID{},
		prepared: PreparedLimit(CasterBard, level.n) - PreparedLimit(CasterBard, previous),
		cantrips: CantripLimit(CasterBard, level.n) - CantripLimit(CasterBard, previous)}
	if level.n == bardWordsOfCreationLevel {
		out.always = wordsOfCreationSpells()
	}
	return out, nil
}
