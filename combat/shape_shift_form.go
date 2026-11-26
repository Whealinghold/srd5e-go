package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// FormFacts is what the shape-shifting spells read of the stat block of a chosen form: its
// id, creature type, size options, Challenge Rating and printed average Hit Points. It is
// an immutable value, comparable with ==. Build it with NewFormFacts; the zero value is
// invalid (see IsZero). Safe for concurrent use.
type FormFacts struct {
	id        content.MonsterID
	kind      content.CreatureType
	sizes     uint8 // bit n-1 is set for the size n
	rating    content.ChallengeRating
	hitPoints int
}

// NewFormFacts reads the facts of monster (SRD 5.2.1, "Monster Stat Blocks", p. 256).
//
// It returns an error matching srderr.ErrInvalidInput for the zero Monster, a Monster with
// no size or an invalid creature type, and an average Hit Points below 1. It never panics.
func NewFormFacts(monster content.Monster) (FormFacts, error) {
	f := FormFacts{id: monster.ID(), kind: monster.Type(), rating: monster.ChallengeRating(), hitPoints: monster.HitPoints().Average()}
	for _, z := range monster.Sizes() {
		if z.Valid() {
			f.sizes |= 1 << (z - 1)
		}
	}
	if f.id == "" || !f.kind.Valid() || f.sizes == 0 || f.hitPoints < 1 {
		return FormFacts{}, invalidInput("form facts need a monster with a type, a size and Hit Points of at least 1")
	}
	return f, nil
}

// IsZero reports whether f is the invalid zero FormFacts.
func (f FormFacts) IsZero() bool { return f.id == "" }

// ID returns the monster id of the form.
func (f FormFacts) ID() content.MonsterID { return f.id }

// Type returns the creature type of the form.
func (f FormFacts) Type() content.CreatureType { return f.kind }

// ChallengeRating returns the Challenge Rating of the form.
func (f FormFacts) ChallengeRating() content.ChallengeRating { return f.rating }

// HitPoints returns the printed average Hit Points of the form.
func (f FormFacts) HitPoints() int { return f.hitPoints }

// Sizes returns the printed size options in ascending order, as a new slice.
func (f FormFacts) Sizes() []content.CreatureSize {
	var out []content.CreatureSize
	for z := content.SizeTiny; z <= content.SizeGargantuan; z++ {
		if f.sizes&(1<<(z-1)) != 0 {
			out = append(out, z)
		}
	}
	return out
}

// AssumedForm is the form a creature takes: a creature of a stat block, or a nonmagical
// object. The zero (nil) AssumedForm is invalid. Variants: CreatureForm, ObjectForm.
type AssumedForm interface {
	// Size returns the size the host chose among the form's printed size options.
	Size() content.CreatureSize
	isAssumedForm()
}

// CreatureForm is a creature form: a stat block and the size chosen among its options. Build
// it with NewCreatureForm. Comparable and immutable.
type CreatureForm struct {
	facts FormFacts
	size  content.CreatureSize
}

// ObjectForm is a nonmagical object form of True Polymorph (SRD 5.2.1, "True Polymorph",
// p. 170), with the creature size categories of objects (shape-changing-spells.md Q-7).
// Build it with NewObjectForm. Comparable and immutable.
type ObjectForm struct{ size content.CreatureSize }

// Size returns the chosen size.
func (f CreatureForm) Size() content.CreatureSize { return f.size }

// Facts returns the facts of the form's stat block.
func (f CreatureForm) Facts() FormFacts { return f.facts }

// Size returns the size of the object.
func (f ObjectForm) Size() content.CreatureSize { return f.size }

func (CreatureForm) isAssumedForm() {}
func (ObjectForm) isAssumedForm()   {}

// SizeError reports a size that is not one of the printed options of a form. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type SizeError struct {
	size    content.CreatureSize
	allowed []content.CreatureSize
}

// Size returns the rejected size.
func (e *SizeError) Size() content.CreatureSize { return e.size }

// Allowed returns the printed size options, as a new slice.
func (e *SizeError) Allowed() []content.CreatureSize { return slices.Clone(e.allowed) }

// Error renders the error. The text is not a contract.
func (e *SizeError) Error() string { return "combat: size is not an option of the form" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *SizeError) Unwrap() error { return srderr.ErrInvalidInput }

// NewCreatureForm returns the creature form of facts in size (SRD 5.2.1, "Size", p. 254).
//
// It returns an error matching srderr.ErrInvalidInput for the zero facts, and a *SizeError
// for a size that is not one of the form's printed options. It never panics.
func NewCreatureForm(facts FormFacts, size content.CreatureSize) (CreatureForm, error) {
	if facts.IsZero() {
		return CreatureForm{}, invalidInput("a creature form needs form facts")
	}
	if !size.Valid() || facts.sizes&(1<<(size-1)) == 0 {
		return CreatureForm{}, &SizeError{size: size, allowed: facts.Sizes()}
	}
	return CreatureForm{facts, size}, nil
}

// NewObjectForm returns the object form of size. It returns an error matching
// srderr.ErrInvalidInput for an invalid size. It never panics.
func NewObjectForm(size content.CreatureSize) (ObjectForm, error) {
	if !size.Valid() {
		return ObjectForm{}, &SizeError{size: size}
	}
	return ObjectForm{size}, nil
}

// ObjectFacts is what the host knows of an object a True Polymorph targets (SRD 5.2.1,
// "True Polymorph", p. 170). Construct it with a keyed literal.
type ObjectFacts struct {
	// Size is the object's size category.
	Size content.CreatureSize
	// IsMagical is whether the object is magical.
	IsMagical bool
	// IsWornOrCarried is whether a creature wears or carries the object.
	IsWornOrCarried bool
}

// TargetRating is the Challenge Rating or level a form may not exceed: the target's (or
// the caster's, for Shapechange). The zero (nil) TargetRating is invalid. Variants:
// RatedByChallenge, RatedByLevel.
type TargetRating interface {
	// Challenge returns the Challenge Rating the rating compares as; a level n compares as
	// the Challenge Rating n.
	Challenge() content.ChallengeRating
	isTargetRating()
}

// RatedByChallenge is a rating by Challenge Rating (a monster target). Comparable and immutable.
type RatedByChallenge struct{ rating content.ChallengeRating }

// NewRatedByChallenge returns the rating of challengeRating.
func NewRatedByChallenge(challengeRating content.ChallengeRating) RatedByChallenge {
	return RatedByChallenge{challengeRating}
}

// Challenge returns the Challenge Rating.
func (r RatedByChallenge) Challenge() content.ChallengeRating { return r.rating }

// RatedByLevel is a rating by character level, 1 to 20 (shape-changing-spells.md Q-5). The
// zero value is invalid. Comparable and immutable.
type RatedByLevel struct{ level int }

// Minimum and maximum character level of a rating.
const (
	minRatingLevel = 1
	maxRatingLevel = 20
)

// LevelRangeError reports a level outside 1 to 20. It matches srderr.ErrInvalidInput with
// errors.Is and carries no display text.
type LevelRangeError struct {
	// Level is the rejected level.
	Level int
	// Min and Max are the allowed range.
	Min, Max int
}

// Error renders the error. The text is not a contract.
func (e *LevelRangeError) Error() string { return "combat: level is outside the character levels" }

// Unwrap returns srderr.ErrInvalidInput.
func (e *LevelRangeError) Unwrap() error { return srderr.ErrInvalidInput }

// NewRatedByLevel returns the rating of character level n. It returns a *LevelRangeError
// (matching srderr.ErrInvalidInput, with Min 1 and Max 20) for n below 1 or above 20. It
// never panics.
func NewRatedByLevel(n int) (RatedByLevel, error) {
	if n < minRatingLevel || n > maxRatingLevel {
		return RatedByLevel{}, &LevelRangeError{Level: n, Min: minRatingLevel, Max: maxRatingLevel}
	}
	return RatedByLevel{n}, nil
}

// Level returns the character level.
func (r RatedByLevel) Level() int { return r.level }

// Challenge returns the Challenge Rating n for a level n, and Challenge Rating 0 for the
// invalid zero value.
func (r RatedByLevel) Challenge() content.ChallengeRating {
	if r.level < minRatingLevel || r.level > maxRatingLevel {
		return content.ChallengeRating{}
	}
	return content.AllChallengeRatings()[r.level+3] // the ratings 0, 1/8, 1/4, 1/2 come first
}

func (RatedByChallenge) isTargetRating() {}
func (RatedByLevel) isTargetRating()     {}

// ratingValid reports whether r is a usable rating: not nil and not the zero level.
func ratingValid(r TargetRating) bool {
	if r == nil {
		return false
	}
	l, byLevel := r.(RatedByLevel)
	return !byLevel || l.level != 0
}
