package content

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
)

// SubclassDefinition is a subclass nested in its [ClassDefinition] (SRD 5.2.1,
// "Barbarian Subclass: Path of the Berserker", p. 30). Subclasses are not
// entries and have no key: they are reached through
// [ClassDefinition.Subclass]. The zero value is invalid; build it with
// [NewSubclassDefinition]. It holds a slice, so it is not comparable with ==.
type SubclassDefinition struct {
	id       SubclassID
	name     string
	source   SourceCitation
	features []FeatureEntry
}

// NewSubclassDefinition returns the subclass id called name with its features,
// copied in order. A zero source means it cites none. That the features start
// at the class's subclass level, carry the subclass id as a prefix and hold no
// [GrantsSubclass] fact is checked when the subclass is added to a class
// ([NewClassDefinition]).
//
// It returns the *[IDError] of [ParseID] for an invalid id, then a
// *[BlockError] wrapping srderr.ErrInvalidInput: BlockInvalidName,
// BlockEmpty for no feature and BlockInvalidMember for a feature that is not
// valid. It never panics.
func NewSubclassDefinition(id SubclassID, name string, source SourceCitation, features []FeatureEntry) (SubclassDefinition, error) {
	const block = "SubclassDefinition"
	if _, err := ParseID(string(id)); err != nil {
		return SubclassDefinition{}, err
	}
	if err := checkName(block, name); err != nil {
		return SubclassDefinition{}, err
	}
	if len(features) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return SubclassDefinition{}, blockError(block, BlockEmpty)
	}
	if slices.ContainsFunc(features, func(f FeatureEntry) bool { return !f.valid() }) {
		return SubclassDefinition{}, blockError(block, BlockInvalidMember)
	}
	return SubclassDefinition{id: id, name: name, source: source, features: slices.Clone(features)}, nil
}

// ID returns the subclass id.
func (s SubclassDefinition) ID() SubclassID { return s.id }

// Name returns the subclass name.
func (s SubclassDefinition) Name() string { return s.name }

// Source returns the citation, or the zero [SourceCitation] when it cites
// none.
func (s SubclassDefinition) Source() SourceCitation { return s.source }

// Features returns the subclass features in printed order, as a new slice.
func (s SubclassDefinition) Features() []FeatureEntry { return slices.Clone(s.features) }

// FeaturesAtOrBelow returns the subclass features whose level does not exceed
// level, in printed order, and an empty slice below the subclass level.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a level outside 1 to [ClassLevels].
func (s SubclassDefinition) FeaturesAtOrBelow(level int) ([]FeatureEntry, error) {
	if err := checkLevel(level); err != nil {
		return nil, err
	}
	out := []FeatureEntry{}
	for _, f := range s.features {
		if f.level <= level {
			out = append(out, f)
		}
	}
	return out, nil
}

func checkLevel(level int) error {
	if level < 1 || level > ClassLevels {
		return blockError("ClassDefinition", BlockOutOfRange)
	}
	return nil
}

// ClassFields holds the members of a [ClassDefinition] beyond its identity, for
// [NewClassDefinition]. Construct it with a keyed literal.
type ClassFields struct {
	// PrimaryAbility is the Primary Ability (required).
	PrimaryAbility AbilityRequirement
	// HitPointDie is the Hit Point Die (required).
	HitPointDie HitPointDie
	// SavingThrowProficiencies lists one to six distinct abilities.
	SavingThrowProficiencies []ability.Ability
	// SkillChoice is the skill proficiency choice (required).
	SkillChoice SkillChoice
	// WeaponProficiencies lists at most one training per category; possibly
	// none.
	WeaponProficiencies []WeaponProficiency
	// ToolProficiencies lists the tool proficiencies; possibly none.
	ToolProficiencies []ClassToolProficiency
	// ArmorTraining lists distinct categories; none is the explicit "None".
	ArmorTraining []ArmorTrainingCategory
	// StartingEquipment is the starting equipment.
	StartingEquipment ClassStartingEquipment
	// MulticlassGrants is what a first level in the class grants as a
	// multiclass character.
	MulticlassGrants MulticlassGrants
	// Spellcasting is the spellcasting trait; nil for a class without.
	Spellcasting ClassSpellcastingTrait
	// Columns declares the class-specific columns of the Features table;
	// possibly none.
	Columns []ClassColumn
	// Progression holds exactly 20 rows, row i being level i+1.
	Progression []ProgressionRow
	// Features lists the class features ordered by level then printed order.
	Features []FeatureEntry
	// Subclasses lists the subclasses; possibly none.
	Subclasses []SubclassDefinition
}

// ClassDefinition is a character class (SRD 5.2.1, "Classes", pp. 28-82): its
// core traits, spellcasting, the 20-row Features table, the class features and
// its subclasses. It is an [Entry] of kind [KindCharacterClass] and implements
// [References]: the spells its always-prepared facts name. Immutable, with its
// collections copied on construction and on read; the zero ClassDefinition is
// invalid. It holds slices, so it is not comparable with ==; compare with
// [Set.Equal]. Safe for concurrent use.
type ClassDefinition struct {
	id          ID
	name        string
	source      SourceCitation
	f           ClassFields
	subclassLvl int
}

var (
	_ Entry      = ClassDefinition{}
	_ References = ClassDefinition{}
)

// NewClassDefinition returns the class id called name, with the members of f. A
// zero source means the entry cites none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members: BlockInvalidName; BlockMissing for a missing
// primary ability or skill choice; BlockInvalidMember for a part that is not
// valid; BlockOutOfRange, BlockDuplicate for the saving throws, armor and
// weapon sets; BlockDuplicate for a repeated column id; BlockOutOfRange for a
// progression of other than [ClassLevels] rows; then the first rule between
// parts of [ClassDefinition.problems] (a feature or subclass id without its
// owner's prefix, a row whose level is not its position, a row's spellcasting
// part or column values that do not match the class, a row naming a feature
// the class lacks or gained later, a feature no row lists at its level, not
// exactly one [GrantsSubclass], a subclass feature below the subclass level or
// no subclass feature at it, and multiclass grants beyond the core traits). It
// never panics.
func NewClassDefinition(id ClassID, name string, source SourceCitation, f ClassFields) (ClassDefinition, error) {
	const block = "ClassDefinition"
	pid, err := ParseID(string(id))
	if err != nil {
		return ClassDefinition{}, err
	}
	if err := checkName(block, name); err != nil {
		return ClassDefinition{}, err
	}
	if f.PrimaryAbility == nil || f.SkillChoice == nil {
		return ClassDefinition{}, blockError(block, BlockMissing)
	}
	if err := checkClassParts(block, f); err != nil {
		return ClassDefinition{}, err
	}
	c := ClassDefinition{id: pid, name: name, source: source, f: cloneClassFields(f)}
	if ps := c.problems(); len(ps) > 0 { // equivalent mutant: a length is never negative, so > and != agree
		return ClassDefinition{}, ps[0].blockError(block)
	}
	c.subclassLvl = c.subclassLevelOf()
	return c, nil
}

// checkClassParts validates each part on its own.
func checkClassParts(block string, f ClassFields) error {
	switch {
	case !validAbilityRequirement(f.PrimaryAbility), !f.HitPointDie.Valid(), !validSkillChoice(f.SkillChoice),
		slices.ContainsFunc(f.ToolProficiencies, func(p ClassToolProficiency) bool { return !validClassToolProficiency(p) }),
		f.Spellcasting != nil && !validSpellcastingTrait(f.Spellcasting),
		slices.ContainsFunc(f.Columns, func(c ClassColumn) bool { return !c.valid() }),
		slices.ContainsFunc(f.Progression, func(r ProgressionRow) bool { return !r.valid() }),
		slices.ContainsFunc(f.Features, func(e FeatureEntry) bool { return !e.valid() }),
		slices.ContainsFunc(f.Subclasses, func(s SubclassDefinition) bool { return !validSubclass(s) }):
		return blockError(block, BlockInvalidMember)
	}
	if err := checkSet(block, f.SavingThrowProficiencies, 1, len(ability.All()), ability.Ability.Valid); err != nil {
		return err
	}
	if err := checkWeaponProficiencies(block, f.WeaponProficiencies); err != nil {
		return err
	}
	if err := checkSet(block, f.ArmorTraining, 0, len(armorTrainingNames()), ArmorTrainingCategory.Valid); err != nil {
		return err
	}
	cols := make(map[ClassColumnID]struct{}, len(f.Columns))
	for _, c := range f.Columns {
		if _, dup := cols[c.id]; dup {
			return blockError(block, BlockDuplicate)
		}
		cols[c.id] = struct{}{}
	}
	if len(f.Progression) != ClassLevels {
		return blockError(block, BlockOutOfRange)
	}
	return nil
}

func validSubclass(s SubclassDefinition) bool {
	_, err := NewSubclassDefinition(s.id, s.name, s.source, s.features)
	return err == nil
}

func cloneClassFields(f ClassFields) ClassFields {
	f.SavingThrowProficiencies = slices.Clone(f.SavingThrowProficiencies)
	f.WeaponProficiencies = cloneOrNil(f.WeaponProficiencies)
	f.ToolProficiencies = cloneOrNil(f.ToolProficiencies)
	f.ArmorTraining = cloneOrNil(f.ArmorTraining)
	f.Columns = cloneOrNil(f.Columns)
	f.Progression = slices.Clone(f.Progression)
	f.Features = slices.Clone(f.Features)
	f.Subclasses = cloneOrNil(f.Subclasses)
	return f
}

// MustClassDefinition is like [NewClassDefinition] but panics with its error
// when the arguments are invalid.
func MustClassDefinition(id ClassID, name string, source SourceCitation, f ClassFields) ClassDefinition {
	c, err := NewClassDefinition(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return c
}

// Key returns the key of the class: kind [KindCharacterClass] and its id.
func (c ClassDefinition) Key() Key { return NewKey(KindCharacterClass, c.id) }

// ID returns the class id.
func (c ClassDefinition) ID() ClassID { return ClassID(c.id.String()) }

// Name returns the class name.
func (c ClassDefinition) Name() string { return c.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (c ClassDefinition) Source() SourceCitation { return c.source }

// PrimaryAbility returns the Primary Ability.
func (c ClassDefinition) PrimaryAbility() AbilityRequirement { return c.f.PrimaryAbility }

// HitPointDie returns the Hit Point Die.
func (c ClassDefinition) HitPointDie() HitPointDie { return c.f.HitPointDie }

// SavingThrowProficiencies returns the saving throw abilities in printed
// order, as a new slice.
func (c ClassDefinition) SavingThrowProficiencies() []ability.Ability {
	return slices.Clone(c.f.SavingThrowProficiencies)
}

// SkillChoice returns the skill proficiency choice.
func (c ClassDefinition) SkillChoice() SkillChoice { return c.f.SkillChoice }

// WeaponProficiencies returns the weapon trainings, as a new slice.
func (c ClassDefinition) WeaponProficiencies() []WeaponProficiency {
	return slices.Clone(c.f.WeaponProficiencies)
}

// ToolProficiencies returns the tool proficiencies, as a new slice.
func (c ClassDefinition) ToolProficiencies() []ClassToolProficiency {
	return slices.Clone(c.f.ToolProficiencies)
}

// ArmorTraining returns the armor training categories, as a new slice; empty
// for "None".
func (c ClassDefinition) ArmorTraining() []ArmorTrainingCategory {
	return slices.Clone(c.f.ArmorTraining)
}

// StartingEquipment returns the starting equipment.
func (c ClassDefinition) StartingEquipment() ClassStartingEquipment { return c.f.StartingEquipment }

// MulticlassGrants returns what a first level in the class grants as a
// multiclass character.
func (c ClassDefinition) MulticlassGrants() MulticlassGrants { return c.f.MulticlassGrants }

// Spellcasting returns the spellcasting trait, or nil for a class without.
func (c ClassDefinition) Spellcasting() ClassSpellcastingTrait { return c.f.Spellcasting }

// Columns returns the class-specific column declarations in printed order, as
// a new slice.
func (c ClassDefinition) Columns() []ClassColumn { return slices.Clone(c.f.Columns) }

// Progression returns the 20 rows of the Features table, as a new slice.
func (c ClassDefinition) Progression() []ProgressionRow { return slices.Clone(c.f.Progression) }

// Features returns the class features in printed order, as a new slice.
func (c ClassDefinition) Features() []FeatureEntry { return slices.Clone(c.f.Features) }

// Subclasses returns the subclasses, as a new slice.
func (c ClassDefinition) Subclasses() []SubclassDefinition { return slices.Clone(c.f.Subclasses) }

// SubclassLevel returns the level of the class feature carrying
// [GrantsSubclass].
func (c ClassDefinition) SubclassLevel() int { return c.subclassLvl }

func (c ClassDefinition) subclassLevelOf() int {
	for _, e := range c.f.Features {
		if slices.ContainsFunc(e.facts, isGrantsSubclass) {
			return e.level
		}
	}
	return 0 // equivalent mutant: replaces the literal 0 with 0, and a constructed class always has the fact
}

func isGrantsSubclass(f FeatureFact) bool { _, ok := f.(GrantsSubclass); return ok }

// RowAt returns the row of the given level.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a level outside 1 to [ClassLevels].
func (c ClassDefinition) RowAt(level int) (ProgressionRow, error) {
	if err := checkLevel(level); err != nil {
		return ProgressionRow{}, err
	}
	return c.f.Progression[level-1], nil
}

// EntriesGainedAt returns the class features whose level equals the argument,
// in printed order, and an empty slice for a level with none.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a level outside 1 to [ClassLevels].
func (c ClassDefinition) EntriesGainedAt(level int) ([]FeatureEntry, error) {
	if err := checkLevel(level); err != nil {
		return nil, err
	}
	out := []FeatureEntry{}
	for _, e := range c.f.Features {
		if e.level == level {
			out = append(out, e)
		}
	}
	return out, nil
}

// ResolvedRow is the Class Features cell of a row with its references
// resolved to feature entries. Immutable; the zero value is an empty cell.
type ResolvedRow struct {
	entries            []FeatureEntry
	hasSubclassFeature bool
}

// Entries returns the row's class features resolved to their entries in
// printed order, as a new slice; an entry may be older than the row.
func (r ResolvedRow) Entries() []FeatureEntry { return slices.Clone(r.entries) }

// HasSubclassFeature reports whether the row holds the subclass feature
// marker.
func (r ResolvedRow) HasSubclassFeature() bool { return r.hasSubclassFeature }

// RowFeatures returns the row of the given level with its feature references
// resolved.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a level outside 1 to [ClassLevels].
func (c ClassDefinition) RowFeatures(level int) (ResolvedRow, error) {
	row, err := c.RowAt(level)
	if err != nil {
		return ResolvedRow{}, err
	}
	out := ResolvedRow{entries: []FeatureEntry{}}
	for _, f := range row.features {
		switch v := f.(type) {
		case RowClassFeature:
			if lookup, ok := c.Feature(v.featureID).(FeatureEntryFound); ok {
				out.entries = append(out.entries, lookup.entry)
			}
		case RowSubclassFeature:
			out.hasSubclassFeature = true
		}
	}
	return out, nil
}

// FeatGrantsAt returns the category of every [GrantsFeat] fact among the
// features the row of the given level resolves to, in printed order.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for a level outside 1 to [ClassLevels].
func (c ClassDefinition) FeatGrantsAt(level int) ([]FeatCategory, error) {
	row, err := c.RowFeatures(level)
	if err != nil {
		return nil, err
	}
	out := []FeatCategory{}
	for _, e := range row.entries {
		for _, f := range e.facts {
			if g, ok := f.(GrantsFeat); ok {
				out = append(out, g.category)
			}
		}
	}
	return out, nil
}

// Column returns the declared class-specific column id.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an undeclared id.
func (c ClassDefinition) Column(id ClassColumnID) (ClassColumn, error) {
	for _, col := range c.f.Columns {
		if col.id == id {
			return col, nil
		}
	}
	return ClassColumn{}, blockError("ClassDefinition", BlockInvalidMember)
}

// FeatureEntryLookup is the result of [ClassDefinition.Feature]: a closed set
// with two variants, [FeatureEntryFound] and [FeatureEntryNotFound]. Consumers
// switch over it exhaustively. The variants are immutable.
type FeatureEntryLookup interface{ isFeatureEntryLookup() }

// FeatureEntryFound holds the class feature found.
type FeatureEntryFound struct{ entry FeatureEntry }

// Entry returns the feature.
func (f FeatureEntryFound) Entry() FeatureEntry { return f.entry }

func (FeatureEntryFound) isFeatureEntryLookup() {}

// FeatureEntryNotFound holds the id that names no class feature of the class.
type FeatureEntryNotFound struct{ id FeatureID }

// ID returns the id that was not found.
func (f FeatureEntryNotFound) ID() FeatureID { return f.id }

func (FeatureEntryNotFound) isFeatureEntryLookup() {}

// Feature looks up a feature of the class by id. A subclass feature id is not
// found here: subclass features are reached through [ClassDefinition.Subclass].
func (c ClassDefinition) Feature(id FeatureID) FeatureEntryLookup {
	for _, e := range c.f.Features {
		if e.id == id {
			return FeatureEntryFound{entry: e}
		}
	}
	return FeatureEntryNotFound{id: id}
}

// SubclassLookup is the result of [ClassDefinition.Subclass]: a closed set with
// two variants, [SubclassFound] and [SubclassNotFound]. Consumers switch over
// it exhaustively.
type SubclassLookup interface{ isSubclassLookup() }

// SubclassFound holds the subclass found.
type SubclassFound struct{ subclass SubclassDefinition }

// Subclass returns the subclass.
func (s SubclassFound) Subclass() SubclassDefinition { return s.subclass }

func (SubclassFound) isSubclassLookup() {}

// SubclassNotFound holds the id that names no subclass of the class.
type SubclassNotFound struct{ id SubclassID }

// ID returns the id that was not found.
func (s SubclassNotFound) ID() SubclassID { return s.id }

func (SubclassNotFound) isSubclassLookup() {}

// Subclass looks up a subclass of the class by id.
func (c ClassDefinition) Subclass(id SubclassID) SubclassLookup {
	for _, s := range c.f.Subclasses {
		if s.id == id {
			return SubclassFound{subclass: s}
		}
	}
	return SubclassNotFound{id: id}
}

// References returns the spells the always-prepared facts name, one site per
// spell id, in the order of the class features, their options, then each
// subclass's features and their options: the fields
// "features/<i>/facts/<j>/grants/<k>/spellIds/<n>" and
// "features/<i>/options/<o>/facts/<j>/grants/<k>/spellIds/<n>", and the same
// under "subclasses/<s>/". The target of each is the spell of kind
// [KindSpell] with no constraint.
func (c ClassDefinition) References() []Reference {
	var refs []Reference
	for i, e := range c.f.Features {
		refs = append(refs, featureReferences(idx("features", i), e)...)
	}
	for s, sub := range c.f.Subclasses {
		for i, e := range sub.features {
			refs = append(refs, featureReferences(idx("subclasses", s, "features", i), e)...)
		}
	}
	return refs
}

func featureReferences(base string, e FeatureEntry) []Reference {
	refs := factReferences(base, e.facts)
	for o, opt := range e.options {
		refs = append(refs, factReferences(base+"/"+idx("options", o), opt.facts)...)
	}
	return refs
}

func factReferences(base string, facts []FeatureFact) []Reference {
	var refs []Reference
	for j, f := range facts {
		a, ok := f.(AlwaysPreparedSpells)
		if !ok {
			continue
		}
		for k, g := range a.grants {
			for n, id := range g.spellIDs {
				refs = append(refs, MustReference(base+"/"+idx("facts", j, "grants", k, "spellIds", n), id.Key(), nil))
			}
		}
	}
	return refs
}
