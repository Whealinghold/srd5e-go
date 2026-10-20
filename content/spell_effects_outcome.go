package content

// DamageTypeSelection is the closed set of ways a damage outcome names its
// Damage Type (SRD 5.2.1, "Damage Types", p. 180): [FixedDamageType] and
// [ChosenDamageType]. Every variant is an immutable, comparable value and safe
// for concurrent use. Consumers switch over it exhaustively.
type DamageTypeSelection interface{ isDamageTypeSelection() }

// FixedDamageType is a damage outcome of one Damage Type. The zero value is
// invalid; build it with [NewFixedDamageType].
type FixedDamageType struct{ damageType DamageType }

// NewFixedDamageType returns the selection of t.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember when t is not a damage type. It never panics.
func NewFixedDamageType(t DamageType) (FixedDamageType, error) {
	if !t.Valid() {
		return FixedDamageType{}, blockError("FixedDamageType", BlockInvalidMember)
	}
	return FixedDamageType{damageType: t}, nil
}

// Type returns the Damage Type.
func (s FixedDamageType) Type() DamageType { return s.damageType }

func (FixedDamageType) isDamageTypeSelection() {}

// ChosenDamageType is a damage outcome whose Damage Type the caster chooses
// from a set of at least two (SRD 5.2.1, "Chromatic Orb", p. 115). The zero
// value is invalid; build it with [NewChosenDamageType]. The set has no order:
// two values are equal when they hold the same types.
type ChosenDamageType struct{ types uint32 }

// NewChosenDamageType returns the selection among types, copying them.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for a value that is not a damage type, BlockDuplicate for
// a repeat and BlockOutOfRange for fewer than two types. It never panics.
func NewChosenDamageType(types []DamageType) (ChosenDamageType, error) {
	vals := make([]int, len(types))
	for i, t := range types {
		vals[i] = int(t)
	}
	mask, err := enumMask("ChosenDamageType", vals, len(damageTypeNames()))
	if err != nil {
		return ChosenDamageType{}, err
	}
	if len(types) < 2 {
		return ChosenDamageType{}, blockError("ChosenDamageType", BlockOutOfRange)
	}
	return ChosenDamageType{types: mask}, nil
}

// Types returns the Damage Types in vocabulary order, as a new slice.
func (s ChosenDamageType) Types() []DamageType {
	var out []DamageType
	for _, v := range maskMembers(s.types, len(damageTypeNames())) {
		out = append(out, DamageType(v))
	}
	return out
}

func (ChosenDamageType) isDamageTypeSelection() {}

// checkDamageTypeSelection reports why s is not a selection its constructor
// would accept: BlockMissing for nil.
func checkDamageTypeSelection(s DamageTypeSelection) error {
	var err error
	switch v := s.(type) {
	case FixedDamageType:
		_, err = NewFixedDamageType(v.damageType)
	case ChosenDamageType:
		_, err = NewChosenDamageType(v.Types())
	default:
		err = blockError("DamageTypeSelection", BlockMissing)
	}
	return err
}

// ConditionDuration is the closed set of ways a condition outcome lasts:
// [ForSpellDuration], [TimedConditionDuration] and [UntilNextTurnBoundary].
// Every variant is an immutable, comparable value and safe for concurrent use.
// Consumers switch over it exhaustively.
type ConditionDuration interface{ isConditionDuration() }

// ForSpellDuration is a condition that lasts for the duration of the spell
// (SRD 5.2.1, "Hold Person", p. 141). The zero value is valid.
type ForSpellDuration struct{}

func (ForSpellDuration) isConditionDuration() {}

// TimedConditionDuration is a condition that lasts a number of rounds,
// minutes, hours or days (SRD 5.2.1, "Duration", p. 106). The zero value is
// invalid; build it with [NewTimedConditionDuration].
type TimedConditionDuration struct {
	amount int
	unit   DurationUnit
}

// NewTimedConditionDuration returns the duration of amount units.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockOutOfRange for an amount below 1 and BlockInvalidMember for an invalid
// unit. It never panics.
func NewTimedConditionDuration(amount int, unit DurationUnit) (TimedConditionDuration, error) {
	if amount < 1 {
		return TimedConditionDuration{}, blockError("TimedConditionDuration", BlockOutOfRange)
	}
	if !unit.Valid() {
		return TimedConditionDuration{}, blockError("TimedConditionDuration", BlockInvalidMember)
	}
	return TimedConditionDuration{amount: amount, unit: unit}, nil
}

// Amount returns the number of units.
func (d TimedConditionDuration) Amount() int { return d.amount }

// Unit returns the unit.
func (d TimedConditionDuration) Unit() DurationUnit { return d.unit }

func (TimedConditionDuration) isConditionDuration() {}

// UntilNextTurnBoundary is a condition that lasts until the next start or end
// of a turn (SRD 5.2.1, "Ray of Sickness", p. 158). The zero value is
// invalid; build it with [NewUntilNextTurnBoundary].
type UntilNextTurnBoundary struct {
	owner    TurnOwner
	boundary TurnBoundary
}

// NewUntilNextTurnBoundary returns the duration up to the next boundary of a
// turn of owner.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid owner or boundary. It never panics.
func NewUntilNextTurnBoundary(owner TurnOwner, boundary TurnBoundary) (UntilNextTurnBoundary, error) {
	if !owner.Valid() || !boundary.Valid() {
		return UntilNextTurnBoundary{}, blockError("UntilNextTurnBoundary", BlockInvalidMember)
	}
	return UntilNextTurnBoundary{owner: owner, boundary: boundary}, nil
}

// Owner returns whose turn it is.
func (d UntilNextTurnBoundary) Owner() TurnOwner { return d.owner }

// Boundary returns the start or the end of that turn.
func (d UntilNextTurnBoundary) Boundary() TurnBoundary { return d.boundary }

func (UntilNextTurnBoundary) isConditionDuration() {}

// checkConditionDuration reports why d is not a duration its constructor would
// accept: BlockMissing for nil.
func checkConditionDuration(d ConditionDuration) error {
	var err error
	switch v := d.(type) {
	case ForSpellDuration:
	case TimedConditionDuration:
		_, err = NewTimedConditionDuration(v.amount, v.unit)
	case UntilNextTurnBoundary:
		_, err = NewUntilNextTurnBoundary(v.owner, v.boundary)
	default:
		err = blockError("ConditionDuration", BlockMissing)
	}
	return err
}

// SpellOutcome is the closed set of effects a part of a spell has on its
// target: [SpellDealsDamage], [SpellRestoresHitPoints],
// [SpellGrantsTemporaryHitPoints] and [SpellAppliesCondition]. Every variant
// is an immutable, comparable value and safe for concurrent use. Consumers
// switch over it exhaustively.
type SpellOutcome interface{ isSpellOutcome() }

// SpellDealsDamage is damage of an amount and a Damage Type (SRD 5.2.1,
// "Damage Rolls", p. 16). The zero value is invalid; build it with
// [NewSpellDealsDamage].
type SpellDealsDamage struct {
	amount     ScalableAmount
	damageType DamageTypeSelection
	adds       ModifierAddition
}

// NewSpellDealsDamage returns the damage outcome.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the amount's or the selection's own check for an invalid one and
// BlockInvalidMember for an invalid addition. It never panics.
func NewSpellDealsDamage(amount ScalableAmount, damageType DamageTypeSelection, adds ModifierAddition) (SpellDealsDamage, error) {
	if err := checkOutcomeAmount("SpellDealsDamage", amount, adds); err != nil {
		return SpellDealsDamage{}, err
	}
	if err := checkDamageTypeSelection(damageType); err != nil {
		return SpellDealsDamage{}, err
	}
	return SpellDealsDamage{amount: amount, damageType: damageType, adds: adds}, nil
}

// Amount returns the damage amount.
func (o SpellDealsDamage) Amount() ScalableAmount { return o.amount }

// Type returns the Damage Type selection.
func (o SpellDealsDamage) Type() DamageTypeSelection { return o.damageType }

// AddsModifier returns whether the spellcasting ability modifier is added.
func (o SpellDealsDamage) AddsModifier() ModifierAddition { return o.adds }

func (SpellDealsDamage) isSpellOutcome() {}

// SpellRestoresHitPoints is a restoration of Hit Points (SRD 5.2.1,
// "Healing", p. 17). The zero value is invalid; build it with
// [NewSpellRestoresHitPoints].
type SpellRestoresHitPoints struct {
	amount ScalableAmount
	adds   ModifierAddition
}

// NewSpellRestoresHitPoints returns the healing outcome.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the amount's own check for an invalid one and BlockInvalidMember for an
// invalid addition. It never panics.
func NewSpellRestoresHitPoints(amount ScalableAmount, adds ModifierAddition) (SpellRestoresHitPoints, error) {
	if err := checkOutcomeAmount("SpellRestoresHitPoints", amount, adds); err != nil {
		return SpellRestoresHitPoints{}, err
	}
	return SpellRestoresHitPoints{amount: amount, adds: adds}, nil
}

// Amount returns the amount of Hit Points restored.
func (o SpellRestoresHitPoints) Amount() ScalableAmount { return o.amount }

// AddsModifier returns whether the spellcasting ability modifier is added.
func (o SpellRestoresHitPoints) AddsModifier() ModifierAddition { return o.adds }

func (SpellRestoresHitPoints) isSpellOutcome() {}

// SpellGrantsTemporaryHitPoints is a grant of Temporary Hit Points (SRD 5.2.1,
// "Temporary Hit Points", p. 18). The zero value is invalid; build it with
// [NewSpellGrantsTemporaryHitPoints].
type SpellGrantsTemporaryHitPoints struct {
	amount ScalableAmount
	adds   ModifierAddition
}

// NewSpellGrantsTemporaryHitPoints returns the Temporary Hit Points outcome.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with the reason
// of the amount's own check for an invalid one and BlockInvalidMember for an
// invalid addition. It never panics.
func NewSpellGrantsTemporaryHitPoints(amount ScalableAmount, adds ModifierAddition) (SpellGrantsTemporaryHitPoints, error) {
	if err := checkOutcomeAmount("SpellGrantsTemporaryHitPoints", amount, adds); err != nil {
		return SpellGrantsTemporaryHitPoints{}, err
	}
	return SpellGrantsTemporaryHitPoints{amount: amount, adds: adds}, nil
}

// Amount returns the number of Temporary Hit Points.
func (o SpellGrantsTemporaryHitPoints) Amount() ScalableAmount { return o.amount }

// AddsModifier returns whether the spellcasting ability modifier is added.
func (o SpellGrantsTemporaryHitPoints) AddsModifier() ModifierAddition { return o.adds }

func (SpellGrantsTemporaryHitPoints) isSpellOutcome() {}

// SpellAppliesCondition is the application of a Condition (SRD 5.2.1,
// "Condition", p. 179). One application of Exhaustion adds one level. The zero
// value is invalid; build it with [NewSpellAppliesCondition].
type SpellAppliesCondition struct {
	kind     ConditionKind
	duration ConditionDuration
}

// NewSpellAppliesCondition returns the condition outcome; any of the fifteen
// conditions is accepted.
//
// It returns a *[BlockError] wrapping srderr.ErrInvalidInput with
// BlockInvalidMember for an invalid condition and the reason of the
// duration's own check for an invalid one (BlockMissing for nil). It never
// panics.
func NewSpellAppliesCondition(kind ConditionKind, duration ConditionDuration) (SpellAppliesCondition, error) {
	if !kind.Valid() {
		return SpellAppliesCondition{}, blockError("SpellAppliesCondition", BlockInvalidMember)
	}
	if err := checkConditionDuration(duration); err != nil {
		return SpellAppliesCondition{}, err
	}
	return SpellAppliesCondition{kind: kind, duration: duration}, nil
}

// Kind returns the Condition.
func (o SpellAppliesCondition) Kind() ConditionKind { return o.kind }

// Duration returns how long the Condition lasts.
func (o SpellAppliesCondition) Duration() ConditionDuration { return o.duration }

func (SpellAppliesCondition) isSpellOutcome() {}

// checkOutcomeAmount checks the amount and the addition shared by the three
// amount outcomes.
func checkOutcomeAmount(block string, amount ScalableAmount, adds ModifierAddition) error {
	if _, err := NewScalableAmount(amount.base, amount.scaling); err != nil {
		return err
	}
	if !adds.Valid() {
		return blockError(block, BlockInvalidMember)
	}
	return nil
}

// checkSpellOutcome reports why o is not an outcome its constructor would
// accept: BlockMissing for nil.
func checkSpellOutcome(o SpellOutcome) error {
	var err error
	switch v := o.(type) {
	case SpellDealsDamage:
		_, err = NewSpellDealsDamage(v.amount, v.damageType, v.adds)
	case SpellRestoresHitPoints:
		_, err = NewSpellRestoresHitPoints(v.amount, v.adds)
	case SpellGrantsTemporaryHitPoints:
		_, err = NewSpellGrantsTemporaryHitPoints(v.amount, v.adds)
	case SpellAppliesCondition:
		_, err = NewSpellAppliesCondition(v.kind, v.duration)
	default:
		err = blockError("SpellOutcome", BlockMissing)
	}
	return err
}
