package content

import "slices"

// SpellEffectModel is the closed set of the structured effects a spell can
// carry beside its text (SRD 5.2.1, "Effects", p. 106): [TextOnlyEffects] and
// [ModelledEffects]. It is plain data: no function, callback or spell-specific
// type. A host reads it with an exhaustive switch; the rules that interpret it
// belong to the context that applies them. Variants may hold slices and are
// not comparable with ==; use Equal.
type SpellEffectModel interface {
	// CheckSpellLevel reports whether the model agrees with the level of the
	// spell that carries it (0 for a cantrip), taking a plain integer so this
	// package needs no Spell (FR-009).
	//
	// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
	// BlockOutOfRange for a level outside 0 to 9, and BlockConstraint for a
	// per-slot scaling on a cantrip, a character-level table on a spell of
	// level 1 or more, or a per-slot AboveLevel below the spell's level or
	// above 9. It never panics.
	CheckSpellLevel(spellLevel int) error
	// Facts answers the derived spell facts (FR-022 to FR-024).
	Facts() SpellEffectFacts
	// Equal reports whether other is the same variant with equal data.
	Equal(other SpellEffectModel) bool
	isSpellEffectModel()
}

// TextOnlyEffects is a spell with no structured effects: the description text
// is the only source, and every fact about it is unknown. The zero value is
// the model and is valid. Comparable.
type TextOnlyEffects struct{}

// CheckSpellLevel returns nil: there is nothing to contradict the level.
func (TextOnlyEffects) CheckSpellLevel(int) error { return nil }

// Facts returns the zero [SpellEffectFacts], which answers [FactUnknown] to
// every question.
func (TextOnlyEffects) Facts() SpellEffectFacts { return SpellEffectFacts{} }

// Equal reports whether other is [TextOnlyEffects].
func (TextOnlyEffects) Equal(other SpellEffectModel) bool {
	_, ok := other.(TextOnlyEffects)
	return ok
}

func (TextOnlyEffects) isSpellEffectModel() {}

// ModelledEffects is a spell with structured effects: a coverage, a targeting
// and at least one part, and optionally the facts of a lingering area. The zero
// value is invalid; build it with [NewModelledEffects]. Not comparable with
// ==; use [ModelledEffects.Equal].
type ModelledEffects struct {
	coverage  EffectCoverage
	targeting SpellTargeting
	parts     []SpellEffectPart
	zone      ZoneFacts
	hasZone   bool
}

// NewModelledEffects returns the effects with coverage, targeting and parts,
// copying the parts. A part the model does not describe exactly stays in the
// text and makes the coverage partial (FR-004, FR-005).
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid coverage, the reason of the targeting's or
// a part's own check for an invalid one, BlockEmpty for no part (FR-006), and
// BlockConstraint for more than one [ChosenDamageType] among the outcomes
// (FR-045). It never panics.
func NewModelledEffects(coverage EffectCoverage, targeting SpellTargeting, parts []SpellEffectPart) (ModelledEffects, error) {
	if !coverage.Valid() {
		return ModelledEffects{}, blockError("ModelledEffects", BlockInvalidMember)
	}
	if err := checkSpellTargeting(targeting); err != nil {
		return ModelledEffects{}, err
	}
	if len(parts) == 0 {
		return ModelledEffects{}, blockError("ModelledEffects", BlockEmpty)
	}
	chosen := 0
	for _, p := range parts {
		if _, err := NewSpellEffectPart(p.resolution, p.instances); err != nil {
			return ModelledEffects{}, err
		}
		for _, o := range p.outcomes() {
			if d, ok := o.(SpellDealsDamage); ok {
				if _, isChosen := d.damageType.(ChosenDamageType); isChosen {
					chosen++
				}
			}
		}
	}
	if chosen > 1 {
		return ModelledEffects{}, blockError("ModelledEffects", BlockConstraint)
	}
	return ModelledEffects{coverage: coverage, targeting: targeting, parts: slices.Clone(parts)}, nil
}

// WithZoneFacts returns a copy of m that also holds the facts of a lingering
// area. A rule's PartIndex is not checked against the parts here.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the zone's own check for the zero [ZoneFacts] or an invalid one, and the
// zero ModelledEffects. It never panics.
func (m ModelledEffects) WithZoneFacts(zone ZoneFacts) (ModelledEffects, error) {
	if _, err := NewZoneFacts(zone.f); err != nil {
		return ModelledEffects{}, err
	}
	m.parts = slices.Clone(m.parts)
	m.zone, m.hasZone = zone, true
	return m, nil
}

// Coverage returns how much of the effect text the model covers.
func (m ModelledEffects) Coverage() EffectCoverage { return m.coverage }

// Targeting returns how the spell picks its targets.
func (m ModelledEffects) Targeting() SpellTargeting { return m.targeting }

// Parts returns a copy of the parts in order.
func (m ModelledEffects) Parts() []SpellEffectPart { return slices.Clone(m.parts) }

// Zone returns the facts of a lingering area, and false when the spell has
// none.
func (m ModelledEffects) Zone() (ZoneFacts, bool) { return m.zone, m.hasZone }

// Facts answers the derived spell facts (FR-022, FR-023).
func (m ModelledEffects) Facts() SpellEffectFacts { return m.facts() }

// Equal reports whether other is a [ModelledEffects] with equal coverage,
// targeting, parts (in order) and zone facts.
func (m ModelledEffects) Equal(other SpellEffectModel) bool {
	o, ok := other.(ModelledEffects)
	if !ok || m.coverage != o.coverage || m.targeting != o.targeting || m.hasZone != o.hasZone ||
		!slices.EqualFunc(m.parts, o.parts, SpellEffectPart.Equal) {
		return false
	}
	return !m.hasZone || m.zone.Equal(o.zone)
}

// CheckSpellLevel applies FR-009 to every scaling of the model.
func (m ModelledEffects) CheckSpellLevel(spellLevel int) error {
	if spellLevel < 0 || spellLevel > 9 {
		return blockError("ModelledEffects", BlockOutOfRange)
	}
	var u scalingUse
	if c, ok := m.targeting.(TargetsChosen); ok {
		switch n := c.count.(type) {
		case ExactlyTargets:
			u.addCount(n.scaling)
		case UpToTargets:
			u.addCount(n.scaling)
		}
	}
	for _, p := range m.parts {
		u.addCount(p.instances.scaling)
		for _, o := range p.outcomes() {
			switch v := o.(type) {
			case SpellDealsDamage:
				u.addAmount(v.amount.scaling)
			case SpellRestoresHitPoints:
				u.addAmount(v.amount.scaling)
			case SpellGrantsTemporaryHitPoints:
				u.addAmount(v.amount.scaling)
			}
		}
	}
	if u.tables > 0 && spellLevel > 0 {
		return blockError("ModelledEffects", BlockConstraint)
	}
	for _, above := range u.above {
		if spellLevel == 0 || above < spellLevel || above > 9 {
			return blockError("ModelledEffects", BlockConstraint)
		}
	}
	return nil
}

func (ModelledEffects) isSpellEffectModel() {}

// scalingUse collects the scalings of a model that CheckSpellLevel judges:
// the AboveLevel of each per-slot scaling and the number of character-level
// tables.
type scalingUse struct {
	above  []int
	tables int
}

func (u *scalingUse) addCount(s CountScaling) {
	switch v := s.(type) {
	case PerSlotLevelCount:
		u.above = append(u.above, v.aboveLevel)
	case ByCharacterLevelCount:
		u.tables++
	}
}

func (u *scalingUse) addAmount(s AmountScaling) {
	switch v := s.(type) {
	case PerSlotLevelAmount:
		u.above = append(u.above, v.aboveLevel)
	case ByCharacterLevelAmount:
		u.tables++
	}
}
