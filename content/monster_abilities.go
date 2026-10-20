package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// AbilityBlock is the Ability Scores of a stat block: the six scores and the
// printed saving throw modifier of each (SRD 5.2.1, "Ability Scores", p. 255).
// The ability modifier is not stored; it comes from the score. The zero value
// is invalid. Comparable and safe for concurrent use.
type AbilityBlock struct {
	scores ability.Scores
	saves  [6]int
}

// NewAbilityBlock returns the block of scores and the printed save modifier
// of each ability in saves.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockMissing
// for the zero scores or an ability with no save, BlockInvalidMember for a
// key that is not an ability, BlockOutOfRange for a save whose magnitude
// exceeds [MonsterMaxNumber]. It never panics.
func NewAbilityBlock(scores ability.Scores, saves map[ability.Ability]int) (AbilityBlock, error) {
	const block = "AbilityBlock"
	if scores.IsZero() {
		return AbilityBlock{}, blockError(block, BlockMissing)
	}
	for a := range saves {
		if !a.Valid() {
			return AbilityBlock{}, blockError(block, BlockInvalidMember)
		}
	}
	b := AbilityBlock{scores: scores}
	for _, a := range ability.All() {
		v, ok := saves[a]
		switch {
		case !ok:
			return AbilityBlock{}, blockError(block, BlockMissing)
		case !monsterNumberOK(v):
			return AbilityBlock{}, blockError(block, BlockOutOfRange)
		}
		b.saves[a-ability.Strength] = v
	}
	return b, nil
}

// IsZero reports whether b is the invalid zero block.
func (b AbilityBlock) IsZero() bool { return b == AbilityBlock{} }

// Scores returns the six ability scores.
func (b AbilityBlock) Scores() ability.Scores { return b.scores }

// AbilityModifier returns the modifier of the score of a, from the Ability
// Modifiers table (SRD 5.2.1, p. 6); it is never stored here.
func (b AbilityBlock) AbilityModifier(a ability.Ability) ability.Modifier {
	return b.scores.Modifier(a)
}

// Save returns the printed saving throw modifier of a, or 0 for an invalid
// ability.
func (b AbilityBlock) Save(a ability.Ability) int {
	if !a.Valid() {
		return 0
	}
	return b.saves[a-ability.Strength]
}

// ProficientInSave reports whether the printed save of a differs from its
// ability modifier, which is how a stat block shows proficiency in the save
// (SRD 5.2.1, "Ability Scores", p. 255).
func (b AbilityBlock) ProficientInSave(a ability.Ability) bool {
	return a.Valid() && b.Save(a) != b.AbilityModifier(a).Value()
}

// DefenseEntry is one entry of the Vulnerabilities, Resistances or Immunities
// of a stat block (SRD 5.2.1, p. 255), a closed family with three variants:
// [DamageTypeDefense], [ConditionDefense] (Immunities only) and
// [DescribedDefense] for printed text that names neither. Consumers switch
// over it exhaustively. Every variant is immutable, comparable and safe for
// concurrent use.
type DefenseEntry interface{ isDefenseEntry() }

// DamageTypeDefense is a damage type with an optional qualifier such as
// "from weapons wielded by creatures under the effect of a Bless spell". The
// zero value is invalid; build it with [NewDamageTypeDefense].
type DamageTypeDefense struct {
	damageType DamageType
	qualifier  string
}

// NewDamageTypeDefense returns the entry for t with qualifier ("" for none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid damage type, then BlockBlankText or
// BlockInvalidCharacter for a non-empty qualifier that breaks the text rule.
// It never panics.
func NewDamageTypeDefense(t DamageType, qualifier string) (DamageTypeDefense, error) {
	if !t.Valid() {
		return DamageTypeDefense{}, blockError("DamageTypeDefense", BlockInvalidMember)
	}
	if err := optionalText("DamageTypeDefense", qualifier); err != nil {
		return DamageTypeDefense{}, err
	}
	return DamageTypeDefense{damageType: t, qualifier: qualifier}, nil
}

// DamageType returns the damage type.
func (d DamageTypeDefense) DamageType() DamageType { return d.damageType }

// Qualifier returns the qualifier, or "" when there is none.
func (d DamageTypeDefense) Qualifier() string { return d.qualifier }

func (DamageTypeDefense) isDefenseEntry() {}

// ConditionDefense is a condition with an optional qualifier such as "with
// Mind Blank". The zero value is invalid; build it with
// [NewConditionDefense].
type ConditionDefense struct {
	condition ConditionKind
	qualifier string
}

// NewConditionDefense returns the entry for c with qualifier ("" for none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid condition, then BlockBlankText or
// BlockInvalidCharacter for a non-empty qualifier that breaks the text rule.
// It never panics.
func NewConditionDefense(c ConditionKind, qualifier string) (ConditionDefense, error) {
	if !c.Valid() {
		return ConditionDefense{}, blockError("ConditionDefense", BlockInvalidMember)
	}
	if err := optionalText("ConditionDefense", qualifier); err != nil {
		return ConditionDefense{}, err
	}
	return ConditionDefense{condition: c, qualifier: qualifier}, nil
}

// Condition returns the condition.
func (c ConditionDefense) Condition() ConditionKind { return c.condition }

// Qualifier returns the qualifier, or "" when there is none.
func (c ConditionDefense) Qualifier() string { return c.qualifier }

func (ConditionDefense) isDefenseEntry() {}

// DescribedDefense is printed text that names neither a damage type nor a
// condition, such as "Damage type chosen for the Draconic Origin trait below".
// The zero value is invalid; build it with [NewDescribedDefense].
type DescribedDefense struct{ text string }

// NewDescribedDefense returns the entry holding text verbatim.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter when text breaks the text rule. It
// never panics.
func NewDescribedDefense(text string) (DescribedDefense, error) {
	if err := checkText("DescribedDefense", text); err != nil {
		return DescribedDefense{}, err
	}
	return DescribedDefense{text: text}, nil
}

// Text returns the verbatim text.
func (d DescribedDefense) Text() string { return d.text }

func (DescribedDefense) isDefenseEntry() {}

// optionalText applies the text rule to s unless it is empty, which means the
// member is absent.
func optionalText(block, s string) error {
	if s == "" {
		return nil
	}
	return checkText(block, s)
}

// validDefense reports whether e is a variant its constructor would accept.
func validDefense(e DefenseEntry) bool {
	var err error
	switch v := e.(type) {
	case DamageTypeDefense:
		_, err = NewDamageTypeDefense(v.damageType, v.qualifier)
	case ConditionDefense:
		_, err = NewConditionDefense(v.condition, v.qualifier)
	case DescribedDefense:
		_, err = NewDescribedDefense(v.text)
	default:
		return false
	}
	return err == nil
}

// defenseIssues returns the problems of one defence list, named by list (the
// path of the list): an entry that is nil or not valid on its own, or a
// condition where the list cannot hold one, is invalidValue at the entry; a
// damage type entry after a condition entry is constraintViolation at that
// entry (content-monsters FR-024, FR-025). A nil entry is skipped: the
// decoder uses it for an entry that had a problem.
func defenseIssues(list string, entries []DefenseEntry, conditionsAllowed bool) []issue {
	var out []issue
	sawCondition := false
	for i, e := range entries {
		if e == nil {
			continue
		}
		p := idx(list, i)
		_, isCondition := e.(ConditionDefense)
		_, isDamage := e.(DamageTypeDefense)
		switch {
		case !validDefense(e) || (isCondition && !conditionsAllowed):
			out = append(out, issue{p, ProblemInvalidValue})
		case isDamage && sawCondition:
			out = append(out, issue{p, ProblemConstraintViolation})
		}
		sawCondition = sawCondition || isCondition
	}
	return out
}

// SenseEntry is one special sense of a stat block: the sense, its range in
// feet and an optional qualifier such as "unimpeded by magical Darkness" (SRD
// 5.2.1, "Senses", p. 256). The zero value is invalid. Comparable and safe for
// concurrent use.
type SenseEntry struct {
	sense     SpecialSense
	feet      int
	qualifier string
}

// NewSenseEntry returns the sense s with a range of feet and qualifier ("" for
// none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockInvalidMember for an invalid sense, BlockOutOfRange for feet below 0 or
// above [MonsterMaxNumber], then BlockBlankText or BlockInvalidCharacter for
// a non-empty qualifier that breaks the text rule. It never panics.
func NewSenseEntry(s SpecialSense, feet int, qualifier string) (SenseEntry, error) {
	switch {
	case !s.Valid():
		return SenseEntry{}, blockError("SenseEntry", BlockInvalidMember)
	case feet < 0 || feet > MonsterMaxNumber:
		return SenseEntry{}, blockError("SenseEntry", BlockOutOfRange)
	}
	if err := optionalText("SenseEntry", qualifier); err != nil {
		return SenseEntry{}, err
	}
	return SenseEntry{sense: s, feet: feet, qualifier: qualifier}, nil
}

// Sense returns the special sense.
func (s SenseEntry) Sense() SpecialSense { return s.sense }

// Feet returns the range in feet.
func (s SenseEntry) Feet() int { return s.feet }

// Qualifier returns the qualifier, or "" when there is none.
func (s SenseEntry) Qualifier() string { return s.qualifier }

// LanguagesEntry is the Languages of a stat block: the printed phrases
// verbatim and an optional telepathy range with its own qualifier (SRD 5.2.1,
// "Languages", p. 256). Telepathy is never one of the phrases. The zero value
// is valid and holds no language and no telepathy ("None"). It holds a slice,
// so compare it with [LanguagesEntry.Equal].
type LanguagesEntry struct {
	phrases       []string
	hasTelepathy  bool
	telepathyFeet int
	qualifier     string
}

// NewLanguagesEntry returns the entry of a copy of phrases, each kept
// verbatim, with no telepathy. No phrase is the empty list ("None").
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockBlankText or BlockInvalidCharacter for the first phrase that breaks the
// text rule. It never panics.
func NewLanguagesEntry(phrases []string) (LanguagesEntry, error) {
	for _, p := range phrases {
		if err := checkText("LanguagesEntry", p); err != nil {
			return LanguagesEntry{}, err
		}
	}
	return LanguagesEntry{phrases: slices.Clone(phrases)}, nil
}

// WithTelepathy returns a copy of l with telepathy of feet and qualifier ("" for
// none).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput:
// BlockOutOfRange for feet below 0 or above [MonsterMaxNumber], then
// BlockBlankText or BlockInvalidCharacter for a non-empty qualifier that
// breaks the text rule. It never panics.
func (l LanguagesEntry) WithTelepathy(feet int, qualifier string) (LanguagesEntry, error) {
	if feet < 0 || feet > MonsterMaxNumber {
		return LanguagesEntry{}, blockError("LanguagesEntry", BlockOutOfRange)
	}
	if err := optionalText("LanguagesEntry", qualifier); err != nil {
		return LanguagesEntry{}, err
	}
	l.phrases = slices.Clone(l.phrases)
	l.hasTelepathy, l.telepathyFeet, l.qualifier = true, feet, qualifier
	return l, nil
}

// Phrases returns the printed phrases in order, as a new slice.
func (l LanguagesEntry) Phrases() []string { return slices.Clone(l.phrases) }

// Telepathy returns the telepathy range in feet, and false when the entry has
// none.
func (l LanguagesEntry) Telepathy() (int, bool) { return l.telepathyFeet, l.hasTelepathy }

// TelepathyQualifier returns the verbatim parenthetical after the telepathy
// range, or "" when there is none. It is never interpreted here.
func (l LanguagesEntry) TelepathyQualifier() string { return l.qualifier }

// Equal reports whether l and o hold the same phrases in order and the same
// telepathy.
func (l LanguagesEntry) Equal(o LanguagesEntry) bool {
	return slices.Equal(l.phrases, o.phrases) && l.hasTelepathy == o.hasTelepathy &&
		l.telepathyFeet == o.telepathyFeet && l.qualifier == o.qualifier
}
