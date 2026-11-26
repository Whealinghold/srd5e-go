package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/character"
)

// EffectEndReason is why an effect ended (FR-040). The zero value is invalid.
type EffectEndReason int

// The reasons an effect ends.
const (
	// EffectExpired: its duration ran out.
	EffectExpired EffectEndReason = iota + 1
	// EffectRepeatSaveSucceeded: its creature succeeded on a repeated or gate save.
	EffectRepeatSaveSucceeded
	// EffectConcentrationLost: the caster's Concentration ended.
	EffectConcentrationLost
	// EffectDismissed: the caster dismissed it.
	EffectDismissed
	// EffectSubjectTookDamage: its creature took damage and the stage ends on damage.
	EffectSubjectTookDamage
	// EffectSubjectDied: its creature died.
	EffectSubjectDied
	// EffectRemovedByEffect: another effect or the host removed it.
	EffectRemovedByEffect
)

func endReasonNames() []string {
	return []string{"expired", "repeat_save_succeeded", "concentration_lost", "dismissed", "subject_took_damage",
		"subject_died", "removed_by_effect"}
}

// Valid reports whether r is one of the seven reasons.
func (r EffectEndReason) Valid() bool { return r >= EffectExpired && r <= EffectRemovedByEffect }

// String returns the snake_case name of r (the value of srd5e.combat.active_effects.end_reason), or
// "unknown".
func (r EffectEndReason) String() string {
	if !r.Valid() {
		return "unknown"
	}
	return endReasonNames()[r-1]
}

// EndingCause is what the host reports to EndActiveEffects. Variants: ConcentrationLost,
// Dismissed, SubjectTookDamage, SubjectDied, RemovedByEffect, MoveEffectSubject (a re-pointing, not
// an end).
type EndingCause interface{ isEndingCause() }

// ConcentrationLost ends every effect bound to a caster's Concentration on a subject, when the
// character rules report ConcentrationEnded (FR-041). Build it with NewConcentrationLost.
type ConcentrationLost struct {
	binding ConcentrationBinding
}

// NewConcentrationLost returns the cause for the Concentration of caster on subject. It returns an
// error matching srderr.ErrInvalidInput for a zero caster or a nil subject. It never panics.
func NewConcentrationLost(caster CombatantID, subject character.ConcentrationSubject) (ConcentrationLost, error) {
	b, err := NewConcentrationBinding(caster, subject)
	return ConcentrationLost{b}, err
}

// Dismissed is a caster dismissing Time Span effects (SRD 5.2.1, "Duration", p. 106). Build it with
// NewDismissed.
type Dismissed struct {
	caster        CombatantID
	incapacitated bool
	ids           []ActiveEffectID
}

// NewDismissed returns the cause of caster dismissing the effects ids, whom the host says has the
// Incapacitated condition or not. It copies ids. It returns an error matching
// srderr.ErrInvalidInput for a zero caster. It never panics.
func NewDismissed(caster CombatantID, casterIncapacitated bool, ids ...ActiveEffectID) (Dismissed, error) {
	if caster.IsZero() {
		return Dismissed{}, invalidInput("dismissal has a zero caster")
	}
	return Dismissed{caster, casterIncapacitated, slices.Clone(ids)}, nil
}

// SubjectTookDamage reports damage to a creature, of the total of the DamageTaken event (absorbed
// damage included). A total below 1 does nothing (FR-033). Build it with NewSubjectTookDamage.
type SubjectTookDamage struct {
	subject CombatantID
	total   int
}

// NewSubjectTookDamage returns the cause. It returns an error matching srderr.ErrInvalidInput for
// a zero creature. It never panics.
func NewSubjectTookDamage(subject CombatantID, total int) (SubjectTookDamage, error) {
	if subject.IsZero() {
		return SubjectTookDamage{}, invalidInput("damage cause has a zero creature")
	}
	return SubjectTookDamage{subject, total}, nil
}

// SubjectDied reports the death of a creature (FR-043). Build it with NewSubjectDied.
type SubjectDied struct{ subject CombatantID }

// NewSubjectDied returns the cause. It returns an error matching srderr.ErrInvalidInput for a zero
// creature. It never panics.
func NewSubjectDied(subject CombatantID) (SubjectDied, error) {
	if subject.IsZero() {
		return SubjectDied{}, invalidInput("death cause has a zero creature")
	}
	return SubjectDied{subject}, nil
}

// RemovedByEffect ends exactly the named effects: a dispel, a shift of shape, someone's action to
// shake a creature out of a spell, or any host decision (FR-044). Build it with NewRemovedByEffect.
type RemovedByEffect struct{ ids []ActiveEffectID }

// NewRemovedByEffect returns the cause, copying ids. It never panics.
func NewRemovedByEffect(ids ...ActiveEffectID) RemovedByEffect {
	return RemovedByEffect{slices.Clone(ids)}
}

// MoveEffectSubject re-points an effect to a new creature, keeping its clocks, binding, potency,
// stage and stored damage (FR-060). It is a move, not an end. Build it with NewMoveEffectSubject.
type MoveEffectSubject struct {
	id         ActiveEffectID
	newSubject EffectSubject
}

// NewMoveEffectSubject returns the cause of moving effect id to newSubject. It returns an error
// matching srderr.ErrInvalidInput for a zero id or an invalid subject. A place is a valid subject
// that the use case refuses. It never panics.
func NewMoveEffectSubject(id ActiveEffectID, newSubject EffectSubject) (MoveEffectSubject, error) {
	if id.IsZero() || !validEffectSubject(newSubject) {
		return MoveEffectSubject{}, invalidInput("move needs an effect and a subject")
	}
	return MoveEffectSubject{id, newSubject}, nil
}

func (ConcentrationLost) isEndingCause() {}
func (Dismissed) isEndingCause()         {}
func (SubjectTookDamage) isEndingCause() {}
func (SubjectDied) isEndingCause()       {}
func (RemovedByEffect) isEndingCause()   {}
func (MoveEffectSubject) isEndingCause() {}

// validEndingCause reports whether c is a variant its constructor would accept.
func validEndingCause(c EndingCause) bool {
	switch v := c.(type) {
	case ConcentrationLost:
		return !v.binding.IsZero()
	case Dismissed:
		return !v.caster.IsZero()
	case SubjectTookDamage:
		return !v.subject.IsZero()
	case SubjectDied:
		return !v.subject.IsZero()
	case RemovedByEffect:
		return true
	case MoveEffectSubject:
		return !v.id.IsZero() && validEffectSubject(v.newSubject)
	}
	return false
}
