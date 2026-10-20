package content

// EffectFact is the closed set of answers to a question about a spell's
// effects: [FactKnown], [FactKnownDamageTypes] and [FactUnknown]. A consumer
// MUST NOT read [FactUnknown] as "no" (spell-effect-data.md FR-024). Every
// variant is an immutable, comparable value and safe for concurrent use.
// Consumers switch over it exhaustively.
type EffectFact interface{ isEffectFact() }

// FactKnown is a known yes or no answer. Build it with a keyed literal.
type FactKnown struct{ Value bool }

func (FactKnown) isEffectFact() {}

// FactKnownDamageTypes is the known set of Damage Types of a spell. The zero
// value is the known empty set.
type FactKnownDamageTypes struct{ types uint32 }

// Types returns the Damage Types in vocabulary order, as a new slice.
func (f FactKnownDamageTypes) Types() []DamageType {
	var out []DamageType
	for _, v := range maskMembers(f.types, len(damageTypeNames())) {
		out = append(out, DamageType(v))
	}
	return out
}

func (FactKnownDamageTypes) isEffectFact() {}

// FactUnknown is no answer: the model does not show it and the text may.
type FactUnknown struct{}

func (FactUnknown) isEffectFact() {}

// SpellEffectFacts are the five facts a class spec needs about a spell
// (spell-effect-data.md FR-022). The zero value answers [FactUnknown] to every
// question. Comparable and safe for concurrent use.
type SpellEffectFacts struct {
	forcesSavingThrow, makesSpellAttackRoll, dealsDamage, damageTypes, canTargetMoreWithHigherSlot EffectFact
}

func orUnknown(f EffectFact) EffectFact {
	if f == nil {
		return FactUnknown{}
	}
	return f
}

// ForcesSavingThrow answers whether the spell forces a saving throw.
func (f SpellEffectFacts) ForcesSavingThrow() EffectFact { return orUnknown(f.forcesSavingThrow) }

// MakesSpellAttackRoll answers whether the spell makes a spell attack roll.
func (f SpellEffectFacts) MakesSpellAttackRoll() EffectFact {
	return orUnknown(f.makesSpellAttackRoll)
}

// DealsDamage answers whether the spell deals damage.
func (f SpellEffectFacts) DealsDamage() EffectFact { return orUnknown(f.dealsDamage) }

// DamageTypes answers with the Damage Types of the spell, a
// [FactKnownDamageTypes] when known.
func (f SpellEffectFacts) DamageTypes() EffectFact { return orUnknown(f.damageTypes) }

// CanTargetMoreWithHigherSlot answers whether a higher-level spell slot adds
// targets. More instances (darts, rays) are not additional targets.
func (f SpellEffectFacts) CanTargetMoreWithHigherSlot() EffectFact {
	return orUnknown(f.canTargetMoreWithHigherSlot)
}

// facts derives the five facts of a model (FR-023): a fact is known true when
// the modelled parts show it, in any coverage; known false and the damage types
// only when the coverage is complete.
func (m ModelledEffects) facts() SpellEffectFacts {
	var saves, attacks, damages, moreTargets bool
	var types uint32
	for _, p := range m.parts {
		switch p.resolution.(type) {
		case SavingThrowResolution:
			saves = true
		case AttackRollResolution:
			attacks = true
		}
		for _, o := range p.outcomes() {
			d, ok := o.(SpellDealsDamage)
			if !ok {
				continue
			}
			damages = true
			switch s := d.damageType.(type) {
			case FixedDamageType:
				types |= uint32(1) << (s.damageType - 1)
			case ChosenDamageType:
				types |= s.types
			}
		}
	}
	if c, ok := m.targeting.(TargetsChosen); ok {
		switch n := c.count.(type) {
		case ExactlyTargets:
			_, moreTargets = n.scaling.(PerSlotLevelCount)
		case UpToTargets:
			_, moreTargets = n.scaling.(PerSlotLevelCount)
		}
	}
	complete := m.coverage == CoverageComplete
	known := func(v bool) EffectFact {
		if v || complete {
			return FactKnown{Value: v}
		}
		return FactUnknown{}
	}
	var damageTypes EffectFact = FactUnknown{}
	if complete {
		damageTypes = FactKnownDamageTypes{types: types}
	}
	return SpellEffectFacts{
		forcesSavingThrow:           known(saves),
		makesSpellAttackRoll:        known(attacks),
		dealsDamage:                 known(damages),
		damageTypes:                 damageTypes,
		canTargetMoreWithHigherSlot: known(moreTargets),
	}
}
