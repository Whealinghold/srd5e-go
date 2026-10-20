package content

import (
	"slices"
	"strconv"
	"strings"
)

// FeatureFact is a fact other contexts read from a class or subclass feature
// as data: a closed set with three variants, [GrantsSubclass], [GrantsFeat]
// and [AlwaysPreparedSpells]. Everything else a feature does is text.
// Consumers switch over it exhaustively. The variants are immutable and safe
// for concurrent use.
type FeatureFact interface{ isFeatureFact() }

// GrantsSubclass marks the one class feature at which the character chooses a
// subclass ("Level 3: Barbarian Subclass"). The zero value is the fact.
// Comparable.
type GrantsSubclass struct{}

func (GrantsSubclass) isFeatureFact() {}

// GrantsFeat says the feature grants a feat of one category (Ability Score
// Improvement grants a General feat, SRD 5.2.1, "Level 4: Ability Score
// Improvement", p. 29). The zero value is invalid; build it with
// [NewGrantsFeat]. Comparable.
type GrantsFeat struct{ category FeatCategory }

// NewGrantsFeat returns the fact of a feat of category.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for a category that is not a member. It never panics.
func NewGrantsFeat(category FeatCategory) (GrantsFeat, error) {
	if !category.Valid() {
		return GrantsFeat{}, blockError("GrantsFeat", BlockInvalidMember)
	}
	return GrantsFeat{category: category}, nil
}

// Category returns the feat category.
func (g GrantsFeat) Category() FeatCategory { return g.category }

func (GrantsFeat) isFeatureFact() {}

// SpellGrant is the spells a character has always prepared from a class level
// on. The zero value is invalid; build it with [NewSpellGrant]. It holds a
// slice, so it is not comparable with ==.
type SpellGrant struct {
	level    int
	spellIDs []SpellID
}

// NewSpellGrant returns the grant of spellIDs (copied, in order) from class
// level.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a level outside 1 to [ClassLevels] or no spell, BlockInvalidMember for
// an invalid id and BlockDuplicate for a repeated one. It never panics.
func NewSpellGrant(level int, spellIDs []SpellID) (SpellGrant, error) {
	if level < 1 || level > ClassLevels {
		return SpellGrant{}, blockError("SpellGrant", BlockOutOfRange)
	}
	if err := checkSet("SpellGrant", spellIDs, 1, noUpperBound, SpellID.Valid); err != nil {
		return SpellGrant{}, err
	}
	return SpellGrant{level: level, spellIDs: slices.Clone(spellIDs)}, nil
}

// Level returns the class level the grant starts at.
func (g SpellGrant) Level() int { return g.level }

// SpellIDs returns the granted spell ids in printed order, as a new slice.
func (g SpellGrant) SpellIDs() []SpellID { return slices.Clone(g.spellIDs) }

func (g SpellGrant) valid() bool {
	_, err := NewSpellGrant(g.level, g.spellIDs)
	return err == nil
}

// AlwaysPreparedSpells lists spells that are always prepared, by the class
// level they start at (SRD 5.2.1, "Level 2: Paladin's Smite", p. 54). The
// grants are in strictly ascending level order. Every spell id is a reference
// to a spell entry checked by [ValidateSet]. The zero value is invalid; build
// it with [NewAlwaysPreparedSpells]. It holds a slice, so it is not comparable
// with ==.
type AlwaysPreparedSpells struct{ grants []SpellGrant }

// NewAlwaysPreparedSpells returns the fact of grants, copied in order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockEmpty for
// no grant and BlockInvalidMember for a grant that is not valid or whose level
// does not rise above the one before. It never panics.
func NewAlwaysPreparedSpells(grants []SpellGrant) (AlwaysPreparedSpells, error) {
	const block = "AlwaysPreparedSpells"
	if len(grants) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return AlwaysPreparedSpells{}, blockError(block, BlockEmpty)
	}
	for i, g := range grants {
		if !g.valid() || i > 0 && g.level <= grants[i-1].level { //nolint:gosec // i > 0 guards the index; equivalent mutant when i > 0 becomes i != 0: an index is never negative
			return AlwaysPreparedSpells{}, blockError(block, BlockInvalidMember)
		}
	}
	return AlwaysPreparedSpells{grants: slices.Clone(grants)}, nil
}

// Grants returns the grants in ascending level order, as a new slice.
func (a AlwaysPreparedSpells) Grants() []SpellGrant { return slices.Clone(a.grants) }

func (AlwaysPreparedSpells) isFeatureFact() {}

func validFeatureFact(f FeatureFact) bool {
	switch v := f.(type) {
	case GrantsSubclass:
		return true
	case GrantsFeat:
		_, err := NewGrantsFeat(v.category)
		return err == nil
	case AlwaysPreparedSpells:
		_, err := NewAlwaysPreparedSpells(v.grants)
		return err == nil
	}
	return false
}

// factKey returns a string that is equal for two facts exactly when they are
// equal, so a list is checked for repeats in one pass with a map. Spell ids
// are slugs, so the separators cannot occur inside one.
func factKey(f FeatureFact) string {
	switch v := f.(type) {
	case GrantsFeat:
		return "f" + strconv.Itoa(int(v.category))
	case AlwaysPreparedSpells:
		var b strings.Builder
		b.WriteString("p") // equivalent mutant: any prefix other than "s" keeps the key apart from GrantsSubclass, and grants are never empty
		for _, g := range v.grants {
			b.WriteString("|" + strconv.Itoa(g.level))
			for _, id := range g.spellIDs {
				b.WriteString("," + string(id))
			}
		}
		return b.String()
	}
	return "s"
}

// OptionPrerequisite is the requirement of a feature option ("Level 5+", "a
// Warlock Cantrip That Deals Damage"): a minimum class level, options of the
// same feature that must be held, and a note holding any remaining printed
// requirement. At least one part is present; the zero value is "no
// prerequisite". Build it with [NewOptionPrerequisite]. It holds a slice, so
// it is not comparable with ==.
type OptionPrerequisite struct {
	minClassLevel int
	requires      []FeatureOptionID
	note          string
}

// NewOptionPrerequisite returns the prerequisite of the given parts: a
// minClassLevel of 0 means none, a nil requires means none, a note of "" means
// none.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput: BlockOutOfRange
// for a level outside 1 to [ClassLevels] when not 0, BlockOutOfRange,
// BlockInvalidMember or BlockDuplicate for a non-nil requires that is empty,
// holds an invalid id or repeats one, BlockBlankText or
// BlockInvalidCharacter for a note that breaks the text rule, and
// BlockConstraint when no part is present. It never panics.
func NewOptionPrerequisite(minClassLevel int, requires []FeatureOptionID, note string) (OptionPrerequisite, error) {
	const block = "OptionPrerequisite"
	if minClassLevel != 0 && (minClassLevel < 1 || minClassLevel > ClassLevels) {
		return OptionPrerequisite{}, blockError(block, BlockOutOfRange)
	}
	if requires != nil {
		if err := checkSet(block, requires, 1, noUpperBound, FeatureOptionID.Valid); err != nil {
			return OptionPrerequisite{}, err
		}
	}
	if note != "" { // equivalent mutant: no string is below "", so != and > agree
		if err := checkText(block, note); err != nil {
			return OptionPrerequisite{}, err
		}
	}
	if minClassLevel == 0 && requires == nil && note == "" { // equivalent mutants: a level is never negative here and no string is below "", so == and <= agree
		return OptionPrerequisite{}, blockError(block, BlockConstraint)
	}
	return OptionPrerequisite{minClassLevel: minClassLevel, requires: slices.Clone(requires), note: note}, nil
}

// MinClassLevel returns the minimum class level, or 0 when there is none.
func (p OptionPrerequisite) MinClassLevel() int { return p.minClassLevel }

// RequiresOptionIDs returns the options that must be held, as a new slice, or
// nil when there are none.
func (p OptionPrerequisite) RequiresOptionIDs() []FeatureOptionID { return slices.Clone(p.requires) }

// Note returns the remaining printed requirement, or "" when there is none.
func (p OptionPrerequisite) Note() string { return p.note }

func (p OptionPrerequisite) zero() bool {
	return p.minClassLevel == 0 && p.requires == nil && p.note == "" // equivalent mutants: a level is never negative here and no string is below "", so == and <= agree
}

// FeatureOptionFields holds the members of a [FeatureOption] beyond its
// identity, for [NewFeatureOption]. Construct it with a keyed literal.
type FeatureOptionFields struct {
	// Text is the faithful option text.
	Text string
	// Prerequisite is the requirement; the zero value for none.
	Prerequisite OptionPrerequisite
	// PointCost is the printed cost, at least 1, or 0 when the SRD prints none.
	PointCost int
	// Repeatable is true when the SRD prints "Repeatable".
	Repeatable bool
	// Facts holds only [AlwaysPreparedSpells] facts, possibly none.
	Facts []FeatureFact
}

// FeatureOption is one named option of a class feature (a Metamagic option, an
// Eldritch Invocation). An option list is a catalogue: how many a character
// holds is stated by the feature text or a column. The zero value is invalid;
// build it with [NewFeatureOption]. It holds slices, so it is not comparable
// with ==.
type FeatureOption struct {
	id         FeatureOptionID
	name       string
	text       string
	prereq     OptionPrerequisite
	pointCost  int
	repeatable bool
	facts      []FeatureFact
}

// NewFeatureOption returns the option id called name with the members of f.
//
// It returns the *[IDError] of [ParseID] for an invalid id, then a
// *[BlockError] wrapping srderr.ErrInvalidInput in the order of the members:
// BlockInvalidName; BlockBlankText or BlockInvalidCharacter for the text;
// BlockOutOfRange for a point cost outside 0 to [ClassMaxCount]; BlockInvalidMember for a fact
// that is not valid and BlockConstraint for one that is not
// [AlwaysPreparedSpells]. It never panics.
func NewFeatureOption(id FeatureOptionID, name string, f FeatureOptionFields) (FeatureOption, error) {
	const block = "FeatureOption"
	if _, err := ParseID(string(id)); err != nil {
		return FeatureOption{}, err
	}
	if err := checkName(block, name); err != nil {
		return FeatureOption{}, err
	}
	if err := checkText(block, f.Text); err != nil {
		return FeatureOption{}, err
	}
	if f.PointCost < 0 || f.PointCost > ClassMaxCount {
		return FeatureOption{}, blockError(block, BlockOutOfRange)
	}
	for _, fact := range f.Facts {
		if !validFeatureFact(fact) {
			return FeatureOption{}, blockError(block, BlockInvalidMember)
		}
		if _, ok := fact.(AlwaysPreparedSpells); !ok {
			return FeatureOption{}, blockError(block, BlockConstraint)
		}
	}
	return FeatureOption{
		id: id, name: name, text: f.Text, prereq: f.Prerequisite, pointCost: f.PointCost,
		repeatable: f.Repeatable, facts: cloneOrNil(f.Facts),
	}, nil
}

// ID returns the option id, unique within its feature.
func (o FeatureOption) ID() FeatureOptionID { return o.id }

// Name returns the printed option name.
func (o FeatureOption) Name() string { return o.name }

// Text returns the faithful option text.
func (o FeatureOption) Text() string { return o.text }

// Prerequisite returns the requirement and true, or the zero value and false
// when the option has none.
func (o FeatureOption) Prerequisite() (OptionPrerequisite, bool) {
	return o.prereq, !o.prereq.zero()
}

// PointCost returns the printed cost, or 0 when the SRD prints none.
func (o FeatureOption) PointCost() int { return o.pointCost }

// Repeatable reports whether the SRD prints "Repeatable" for the option.
func (o FeatureOption) Repeatable() bool { return o.repeatable }

// Facts returns the option's facts, as a new slice.
func (o FeatureOption) Facts() []FeatureFact { return slices.Clone(o.facts) }

func (o FeatureOption) valid() bool {
	_, err := NewFeatureOption(o.id, o.name, FeatureOptionFields{
		Text: o.text, Prerequisite: o.prereq, PointCost: o.pointCost, Repeatable: o.repeatable, Facts: o.facts, // equivalent mutants: NewFeatureOption never checks Prerequisite or Repeatable
	})
	return err == nil
}
