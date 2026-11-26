package combat

import "github.com/Whealinghold/srd5e-go/srderr"

// ItemUseField names the argument of an item-use constructor or rule that is violated. The
// zero value is not a field. String returns the snake_case name.
type ItemUseField int

// The fields.
const (
	ItemUseFieldKind ItemUseField = iota + 1
	ItemUseFieldThrower
	ItemUseFieldTarget
	ItemUseFieldDistanceFeet
	ItemUseFieldSight
	ItemUseFieldCover
	ItemUseFieldAttempt
	ItemUseFieldElapsed
	ItemUseFieldDamageTaken
	ItemUseFieldAmmunitionPieces
	ItemUseFieldUsesRemaining
	ItemUseFieldOwner
	ItemUseFieldUser
	ItemUseFieldPatient
	ItemUseFieldStatus
	ItemUseFieldPotionUse
	ItemUseFieldCoating
	ItemUseFieldEffect
	ItemUseFieldDamage
)

// String returns the snake_case name of f, or "unknown".
func (f ItemUseField) String() string {
	return enumName([]string{"kind", "thrower", "target", "distance_feet", "sight", "cover", "attempt", "elapsed",
		"damage_taken", "ammunition_pieces", "uses_remaining", "owner", "user", "patient", "status", "potion_use",
		"coating", "effect", "damage"}, int(f))
}

// ItemUseError reports an argument an item-use constructor or rule cannot use: the violated
// Field and, for a number, the Value that was given (a duration in nanoseconds; 0 for a field
// that is not a number). It matches srderr.ErrInvalidInput with errors.Is, is comparable and
// carries no display text.
type ItemUseError struct {
	Field ItemUseField
	Value int64
}

// Error renders the error. The text is not a contract.
func (e *ItemUseError) Error() string { return "combat: invalid item use value: " + e.Field.String() }

// Unwrap returns srderr.ErrInvalidInput.
func (e *ItemUseError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *ItemUseError with the same Field and Value.
func (e *ItemUseError) Is(target error) bool {
	t, ok := target.(*ItemUseError)
	return ok && t != nil && e != nil && *t == *e
}

// itemUseInvalid returns the *ItemUseError of f with the given value.
func itemUseInvalid(f ItemUseField, value int64) error { return &ItemUseError{f, value} }
