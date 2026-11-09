package character

// Spans of the Wish use cases WishResolver.ResolveWish and WishResolver.RollWishStressDamage. Both
// record on the shared MetricOperations counter and MetricDuration histogram.
const (
	// SpanResolveWish wraps one WishResolver.ResolveWish call.
	SpanResolveWish = "srd5e.character.resolve_wish"
	// SpanRollWishStressDamage wraps one WishResolver.RollWishStressDamage call.
	SpanRollWishStressDamage = "srd5e.character.roll_wish_stress_damage"
	// AttrWishEffect is the effect chosen: a ValueWishEffect* value; the ResolveWish span and counter, every outcome.
	AttrWishEffect = "srd5e.character.wish_effect"
	// AttrWishLockout is a bool: whether the stress locked the creature out of Wish for good; the
	// ResolveWish span and counter, applied results only.
	AttrWishLockout = "srd5e.character.wish_lockout"
)

// Attribute values of the Wish use cases. The rejection values are the String of
// ResolveWishRejectionKind. ValueResultApplied, ValueResultNone and ValueResultRejected are shared.
const (
	ValueOperationResolveWish          = "resolve_wish"
	ValueOperationRollWishStressDamage = "roll_wish_stress_damage"

	ValueResultDuplicated = "duplicated"
	ValueResultDamage     = "damage"

	ValueWishEffectDuplicate      = "duplicate"
	ValueWishEffectObjectCreation = "object_creation"
	ValueWishEffectInstantHealth  = "instant_health"
	ValueWishEffectResistance     = "resistance"
	ValueWishEffectSpellImmunity  = "spell_immunity"
	ValueWishEffectSuddenLearning = "sudden_learning"
	ValueWishEffectRollRedo       = "roll_redo"
	ValueWishEffectReshapeReality = "reshape_reality"
)

// wishEffectValue maps a choice to its ValueWishEffect* value.
func wishEffectValue(choice WishChoice) string {
	switch choice.(type) {
	case DuplicateSpell:
		return ValueWishEffectDuplicate
	case ObjectCreation:
		return ValueWishEffectObjectCreation
	case InstantHealth:
		return ValueWishEffectInstantHealth
	case ResistanceGrant:
		return ValueWishEffectResistance
	case SpellImmunityGrant:
		return ValueWishEffectSpellImmunity
	case SuddenLearning:
		return ValueWishEffectSuddenLearning
	case RollRedo:
		return ValueWishEffectRollRedo
	}
	return ValueWishEffectReshapeReality
}
