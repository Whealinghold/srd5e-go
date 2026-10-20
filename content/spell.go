package content

import "slices"

// SpellFields holds the members of a [Spell] beyond its identity, for
// [NewSpell]. Construct it with a keyed literal.
type SpellFields struct {
	// Level is the spell level, 0 to 9 (required; the zero value is a cantrip).
	Level SpellLevel
	// School is the school of magic (required).
	School SpellSchool
	// Classes lists the classes whose spell list holds the spell, in any order;
	// it may be empty. A repeat is rejected.
	Classes []ClassID
	// CastingTimes is the casting time option or options (required).
	CastingTimes SpellCastingTimes
	// Ritual says whether the spell can be cast as a Ritual.
	Ritual bool
	// Range is the spell range (required).
	Range SpellRange
	// Components are the spell components.
	Components SpellComponents
	// Duration is the spell duration (required).
	Duration SpellDuration
	// Description is a non-empty list of text blocks in printed order.
	Description []TextBlock
	// HigherLevel is the text of "Using a Higher-Level Spell Slot"; nil for
	// none. A non-nil empty list is rejected.
	HigherLevel []TextBlock
	// CantripUpgrade is the text of "Cantrip Upgrade"; nil for none. A non-nil
	// empty list is rejected.
	CantripUpgrade []TextBlock
	// Effects is the structured effects; nil reads as [TextOnlyEffects].
	Effects SpellEffectModel
}

// Spell is a spell (SRD 5.2.1, "Spell Descriptions", pp. 107-175). It is an
// [Entry] of kind [KindSpell]. Its description stays the authoritative text;
// the structured Effects are an optional extra (spell-effect-data.md FR-003).
// Immutable, with its collections copied on construction and on read; the zero
// Spell is invalid. It holds slices, so it is not comparable with ==; compare
// with [Spell.Equal] or [Set.Equal]. Safe for concurrent use.
type Spell struct {
	id             ID
	name           string
	source         SourceCitation
	level          SpellLevel
	school         SpellSchool
	classes        []ClassID // sorted, no repeat
	castingTimes   SpellCastingTimes
	ritual         bool
	spellRange     SpellRange
	components     SpellComponents
	duration       SpellDuration
	description    []TextBlock
	higherLevel    []TextBlock
	cantripUpgrade []TextBlock
	effects        SpellEffectModel
}

var _ Entry = Spell{}

// ValidSpellName reports whether s is a spell name: the name rule of schema
// FR-027 (non-empty, no surrounding white space, clean characters).
func ValidSpellName(s string) bool { return validName(s) }

// NewSpell returns the spell id called name, with the members of f. A zero
// source means the entry cites none.
//
// It returns an error wrapping srderr.ErrInvalidInput: the *[IDError] of
// [ParseID] for an invalid id, then a *[BlockError] for the first rule broken,
// in the order of the members: BlockInvalidName; BlockOutOfRange for a level
// outside 0 to 9; BlockInvalidMember for a school that is not a member, an
// invalid class id or a text block that is not valid; BlockDuplicate for a
// repeated class; the reason of the casting times', range's, components' or
// duration's own check for an invalid one; BlockEmpty for an empty description
// or an empty non-nil higher-level or cantrip upgrade list; and the reason of
// the effects' own check, or BlockConstraint for effects that contradict the
// level (spell-effect-data.md FR-009). It never panics.
func NewSpell(id SpellID, name string, source SourceCitation, f SpellFields) (Spell, error) {
	const block = "Spell"
	pid, err := ParseID(string(id))
	if err != nil {
		return Spell{}, err
	}
	if !ValidSpellName(name) {
		return Spell{}, blockError(block, BlockInvalidName)
	}
	if !f.Level.Valid() {
		return Spell{}, blockError(block, BlockOutOfRange)
	}
	if !f.School.Valid() {
		return Spell{}, blockError(block, BlockInvalidMember)
	}
	classes, err := checkClasses(f.Classes)
	if err != nil {
		return Spell{}, err
	}
	components, err := checkSpellParts(f)
	if err != nil {
		return Spell{}, err
	}
	if len(f.Description) == 0 { // equivalent mutant: a length is never negative, so == and <= agree
		return Spell{}, blockError(block, BlockEmpty)
	}
	for _, l := range [][]TextBlock{f.Description, f.HigherLevel, f.CantripUpgrade} {
		if l != nil && len(l) == 0 {
			return Spell{}, blockError(block, BlockEmpty)
		}
		if slices.ContainsFunc(l, func(b TextBlock) bool { return !validTextBlock(b) }) {
			return Spell{}, blockError(block, BlockInvalidMember)
		}
	}
	effects, err := checkSpellEffects(f.Effects, int(f.Level))
	if err != nil {
		return Spell{}, err
	}
	return Spell{
		id: pid, name: name, source: source, level: f.Level, school: f.School, classes: classes,
		castingTimes: cloneCastingTimes(f.CastingTimes), ritual: f.Ritual, spellRange: f.Range,
		components: components, duration: f.Duration,
		description: slices.Clone(f.Description), higherLevel: cloneOrNil(f.HigherLevel),
		cantripUpgrade: cloneOrNil(f.CantripUpgrade), effects: effects,
	}, nil
}

// checkClasses returns the sorted copy of classes, or the error for an invalid
// id (BlockInvalidMember) or a repeat (BlockDuplicate).
func checkClasses(classes []ClassID) ([]ClassID, error) {
	if slices.ContainsFunc(classes, func(c ClassID) bool { return !c.Valid() }) {
		return nil, blockError("Spell", BlockInvalidMember)
	}
	out := cloneOrNil(classes)
	slices.Sort(out)
	if len(slices.Compact(slices.Clone(out))) != len(out) {
		return nil, blockError("Spell", BlockDuplicate)
	}
	return out, nil
}

// checkSpellParts applies the casting times', range's, components' and
// duration's own checks, in that order, and returns the components with their
// material copied.
func checkSpellParts(f SpellFields) (SpellComponents, error) {
	if err := checkSpellCastingTimes(f.CastingTimes); err != nil {
		return SpellComponents{}, err
	}
	if err := checkSpellRange(f.Range); err != nil {
		return SpellComponents{}, err
	}
	components := NewSpellComponents(f.Components.verbal, f.Components.somatic)
	if m, has := f.Components.Material(); has {
		var err error
		if components, err = components.WithMaterial(m); err != nil {
			return SpellComponents{}, err
		}
	}
	return components, checkSpellDuration(f.Duration)
}

func cloneCastingTimes(c SpellCastingTimes) SpellCastingTimes {
	if m, ok := c.(SpellCastingTimesModes); ok {
		return SpellCastingTimesModes{modes: slices.Clone(m.modes)}
	}
	return c
}

// checkSpellEffects returns m with nil read as [TextOnlyEffects], after
// checking it is a variant its constructors would accept and that it agrees
// with the spell level (spell-effect-data.md FR-009).
func checkSpellEffects(m SpellEffectModel, level int) (SpellEffectModel, error) {
	switch v := m.(type) {
	case nil:
		return TextOnlyEffects{}, nil
	case TextOnlyEffects:
		return v, nil
	case ModelledEffects:
		rebuilt, err := NewModelledEffects(v.coverage, v.targeting, v.parts)
		if err == nil && v.hasZone {
			rebuilt, err = rebuilt.WithZoneFacts(v.zone)
		}
		if err != nil {
			return nil, err
		}
		if err := rebuilt.CheckSpellLevel(level); err != nil {
			return nil, err
		}
		return rebuilt, nil
	}
	return nil, blockError("Spell", BlockInvalidMember)
}

// MustSpell is like [NewSpell] but panics with its error when the arguments
// are invalid.
func MustSpell(id SpellID, name string, source SourceCitation, f SpellFields) Spell {
	s, err := NewSpell(id, name, source, f)
	if err != nil {
		panic(err)
	}
	return s
}

// Key returns the key of the spell: kind [KindSpell] and its id.
func (s Spell) Key() Key { return NewKey(KindSpell, s.id) }

// ID returns the spell id.
func (s Spell) ID() SpellID { return SpellID(s.id.String()) }

// Name returns the heading exactly as printed.
func (s Spell) Name() string { return s.name }

// Source returns the citation of the entry, or the zero [SourceCitation] when
// it cites none.
func (s Spell) Source() SourceCitation { return s.source }

// Level returns the spell level.
func (s Spell) Level() SpellLevel { return s.level }

// School returns the school of magic.
func (s Spell) School() SpellSchool { return s.school }

// Classes returns the classes whose spell list holds the spell, in ordinal
// order, as a new slice, or nil when there is none.
func (s Spell) Classes() []ClassID { return slices.Clone(s.classes) }

// CastingTimes returns the casting time option or options.
func (s Spell) CastingTimes() SpellCastingTimes { return cloneCastingTimes(s.castingTimes) }

// Ritual reports whether the spell can be cast as a Ritual.
func (s Spell) Ritual() bool { return s.ritual }

// Range returns the spell range.
func (s Spell) Range() SpellRange { return s.spellRange }

// Components returns the spell components.
func (s Spell) Components() SpellComponents { return s.components }

// Duration returns the spell duration.
func (s Spell) Duration() SpellDuration { return s.duration }

// Description returns the description blocks in printed order, as a new slice.
func (s Spell) Description() []TextBlock { return slices.Clone(s.description) }

// HigherLevel returns the text of "Using a Higher-Level Spell Slot" as a new
// slice, or nil when the spell has none.
func (s Spell) HigherLevel() []TextBlock { return slices.Clone(s.higherLevel) }

// CantripUpgrade returns the text of "Cantrip Upgrade" as a new slice, or nil
// when the spell has none.
func (s Spell) CantripUpgrade() []TextBlock { return slices.Clone(s.cantripUpgrade) }

// Effects returns the structured effects; [TextOnlyEffects] when the spell has
// none.
func (s Spell) Effects() SpellEffectModel { return s.effects }

// RequiresConcentration reports whether the duration is a Time Span limited by
// Concentration (SRD 5.2.1, "Duration", p. 106).
func (s Spell) RequiresConcentration() bool {
	d, ok := s.duration.(SpellDurationTimed)
	return ok && d.limit == LimitConcentrationUpTo
}

// Equal reports whether s and o are the same spell: every member equal, the
// effects included, the lists in the same order and the classes as a set.
func (s Spell) Equal(o Spell) bool {
	if s.effects == nil || o.effects == nil {
		return s.effects == nil && o.effects == nil && s.id == o.id // the zero Spell
	}
	return s.id == o.id && s.name == o.name && s.source == o.source && s.level == o.level && s.school == o.school &&
		s.ritual == o.ritual && s.spellRange == o.spellRange && s.duration == o.duration &&
		slices.Equal(s.classes, o.classes) && SpellCastingTimesEqual(s.castingTimes, o.castingTimes) &&
		s.components.Equal(o.components) && s.effects.Equal(o.effects) &&
		slices.EqualFunc(s.description, o.description, textBlockEqual) &&
		slices.EqualFunc(s.higherLevel, o.higherLevel, textBlockEqual) &&
		slices.EqualFunc(s.cantripUpgrade, o.cantripUpgrade, textBlockEqual)
}
