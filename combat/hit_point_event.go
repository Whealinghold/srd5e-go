package combat

import (
	"slices"
)

// HitPointEvent is the closed family of events the hit point rules produce.
// Each is an immutable value about one creature (its Combatants is exactly
// [subject]) with a constant snake_case Kind. Variants: DamageTaken,
// DroppedToZeroHitPoints, FellUnconscious, KnockedOut, BecameBloodied,
// HitPointsRestored, TemporaryHitPointsGained, TemporaryHitPointsEnded,
// HitPointMaximumReduced, HitPointMaximumRestored, DeathSavingThrowMade,
// BecameStable, LostStability, Recovered, Died. Each embeds hitPointEventBase, which
// carries the subject and the markers.
type HitPointEvent interface {
	Event
	hitPointEvent()
}

// hitPointEventBase carries the subject every hit point event is about and the
// markers that put the event in the Event and HitPointEvent families.
type hitPointEventBase struct{ subject CombatantID }

func (hitPointEventBase) event()         {}
func (hitPointEventBase) hitPointEvent() {}

// Combatants returns a new slice holding the subject.
func (b hitPointEventBase) Combatants() []CombatantID { return []CombatantID{b.subject} }

func newBase(subject CombatantID) (hitPointEventBase, error) {
	if subject.IsZero() {
		return hitPointEventBase{}, invalidInput("event without a subject")
	}
	return hitPointEventBase{subject}, nil
}

// DamageTaken: the creature took damage that was above 0 after the pipeline.
// It holds the resolution of each instance and the post-pipeline total split in
// the part Temporary Hit Points absorbed and the part that went against Hit
// Points ("lost", including any overkill and damage at 0 Hit Points). Its Kind
// is damage_taken. It is not comparable with ==; use Equal.
type DamageTaken struct {
	hitPointEventBase
	resolutions    []DamageResolution
	absorbed, lost int
}

// NewDamageTaken returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or a negative amount.
func NewDamageTaken(subject CombatantID, resolutions []DamageResolution, absorbedByTemporary, hitPointsLost int) (DamageTaken, error) {
	b, err := newBase(subject)
	if err != nil || absorbedByTemporary < 0 || hitPointsLost < 0 {
		return DamageTaken{}, invalidInput("damage taken needs a subject and amounts of 0 or more")
	}
	return DamageTaken{b, slices.Clone(resolutions), absorbedByTemporary, hitPointsLost}, nil
}

// Kind returns "damage_taken".
func (DamageTaken) Kind() string { return "damage_taken" }

// Resolutions returns a copy of the per-instance resolutions.
func (e DamageTaken) Resolutions() []DamageResolution { return slices.Clone(e.resolutions) }

// AbsorbedByTemporary returns the damage Temporary Hit Points took.
func (e DamageTaken) AbsorbedByTemporary() int { return e.absorbed }

// HitPointsLost returns the damage that went against Hit Points.
func (e DamageTaken) HitPointsLost() int { return e.lost }

// Total returns HitPointsLost plus AbsorbedByTemporary: the damage taken, the
// one amount a Concentration save reads.
func (e DamageTaken) Total() int { return satAdd(e.absorbed, e.lost) }

// Equal reports whether e and o are the same event by value.
func (e DamageTaken) Equal(o DamageTaken) bool {
	return e.subject == o.subject && e.absorbed == o.absorbed && e.lost == o.lost && slices.Equal(e.resolutions, o.resolutions)
}

// DroppedToZeroHitPoints: the creature dropped to 0 Hit Points. Kind
// dropped_to_zero_hit_points.
type DroppedToZeroHitPoints struct{ hitPointEventBase }

// FellUnconscious: the creature fell Unconscious at 0 Hit Points. Kind
// fell_unconscious.
type FellUnconscious struct{ hitPointEventBase }

// KnockedOut: a melee attack knocked the creature out; the consumer applies the
// Unconscious condition. Kind knocked_out.
type KnockedOut struct{ hitPointEventBase }

// BecameBloodied: damage took the creature from not Bloodied to Bloodied. Kind
// became_bloodied.
type BecameBloodied struct{ hitPointEventBase }

// TemporaryHitPointsEnded: the creature's Temporary Hit Points reached 0. Kind
// temporary_hit_points_ended.
type TemporaryHitPointsEnded struct{ hitPointEventBase }

// BecameStable: the creature became Stable. Kind became_stable.
type BecameStable struct{ hitPointEventBase }

// LostStability: damage ended a Stable creature's stability. Kind lost_stability.
type LostStability struct{ hitPointEventBase }

// Recovered: the creature regained Hit Points at 0 Hit Points and is Alive
// again. Kind recovered.
type Recovered struct{ hitPointEventBase }

// Kind returns "dropped_to_zero_hit_points".
func (DroppedToZeroHitPoints) Kind() string { return "dropped_to_zero_hit_points" }

// Kind returns "fell_unconscious".
func (FellUnconscious) Kind() string { return "fell_unconscious" }

// Kind returns "knocked_out".
func (KnockedOut) Kind() string { return "knocked_out" }

// Kind returns "became_bloodied".
func (BecameBloodied) Kind() string { return "became_bloodied" }

// Kind returns "temporary_hit_points_ended".
func (TemporaryHitPointsEnded) Kind() string { return "temporary_hit_points_ended" }

// Kind returns "became_stable".
func (BecameStable) Kind() string { return "became_stable" }

// Kind returns "lost_stability".
func (LostStability) Kind() string { return "lost_stability" }

// Kind returns "recovered".
func (Recovered) Kind() string { return "recovered" }

// NewDroppedToZeroHitPoints returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewDroppedToZeroHitPoints(subject CombatantID) (DroppedToZeroHitPoints, error) {
	b, err := newBase(subject)
	return DroppedToZeroHitPoints{b}, err
}

// NewFellUnconscious returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewFellUnconscious(subject CombatantID) (FellUnconscious, error) {
	b, err := newBase(subject)
	return FellUnconscious{b}, err
}

// NewKnockedOut returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewKnockedOut(subject CombatantID) (KnockedOut, error) {
	b, err := newBase(subject)
	return KnockedOut{b}, err
}

// NewBecameBloodied returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewBecameBloodied(subject CombatantID) (BecameBloodied, error) {
	b, err := newBase(subject)
	return BecameBloodied{b}, err
}

// NewTemporaryHitPointsEnded returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewTemporaryHitPointsEnded(subject CombatantID) (TemporaryHitPointsEnded, error) {
	b, err := newBase(subject)
	return TemporaryHitPointsEnded{b}, err
}

// NewBecameStable returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewBecameStable(subject CombatantID) (BecameStable, error) {
	b, err := newBase(subject)
	return BecameStable{b}, err
}

// NewLostStability returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewLostStability(subject CombatantID) (LostStability, error) {
	b, err := newBase(subject)
	return LostStability{b}, err
}

// NewRecovered returns the event; it returns an error matching
// srderr.ErrInvalidInput for a zero subject.
func NewRecovered(subject CombatantID) (Recovered, error) {
	b, err := newBase(subject)
	return Recovered{b}, err
}

// HitPointsRestored: the creature regained Hit Points. Kind
// hit_points_restored.
type HitPointsRestored struct {
	hitPointEventBase
	regained int
}

// NewHitPointsRestored returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or a regained amount below 1.
func NewHitPointsRestored(subject CombatantID, regained int) (HitPointsRestored, error) {
	b, err := newBase(subject)
	if err != nil || regained < 1 {
		return HitPointsRestored{}, invalidInput("restored needs a subject and at least 1 Hit Point")
	}
	return HitPointsRestored{b, regained}, nil
}

// Kind returns "hit_points_restored".
func (HitPointsRestored) Kind() string { return "hit_points_restored" }

// Regained returns the Hit Points regained.
func (e HitPointsRestored) Regained() int { return e.regained }

// TemporaryHitPointsGained: the creature gained Temporary Hit Points. Kind
// temporary_hit_points_gained.
type TemporaryHitPointsGained struct {
	hitPointEventBase
	amount int
}

// NewTemporaryHitPointsGained returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or an amount below 1.
func NewTemporaryHitPointsGained(subject CombatantID, amount int) (TemporaryHitPointsGained, error) {
	b, err := newBase(subject)
	if err != nil || amount < 1 {
		return TemporaryHitPointsGained{}, invalidInput("gain needs a subject and at least 1 Temporary Hit Point")
	}
	return TemporaryHitPointsGained{b, amount}, nil
}

// Kind returns "temporary_hit_points_gained".
func (TemporaryHitPointsGained) Kind() string { return "temporary_hit_points_gained" }

// Amount returns the Temporary Hit Points the creature now holds.
func (e TemporaryHitPointsGained) Amount() int { return e.amount }

// HitPointMaximumReduced: the creature's Hit Point maximum was lowered. Kind
// hit_point_maximum_reduced.
type HitPointMaximumReduced struct {
	hitPointEventBase
	previous, maximum int
}

// HitPointMaximumRestored: a reduced Hit Point maximum was raised back. Kind
// hit_point_maximum_restored.
type HitPointMaximumRestored struct {
	hitPointEventBase
	previous, maximum int
}

// NewHitPointMaximumReduced returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject, a negative maximum or a maximum
// not below the previous one.
func NewHitPointMaximumReduced(subject CombatantID, previous, maximum int) (HitPointMaximumReduced, error) {
	b, err := newBase(subject)
	if err != nil || maximum < 0 || maximum >= previous {
		return HitPointMaximumReduced{}, invalidInput("reduced needs a subject and a lower maximum of 0 or more")
	}
	return HitPointMaximumReduced{b, previous, maximum}, nil
}

// NewHitPointMaximumRestored returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject, a previous maximum below 0 or a
// maximum not above it.
func NewHitPointMaximumRestored(subject CombatantID, previous, maximum int) (HitPointMaximumRestored, error) {
	b, err := newBase(subject)
	if err != nil || previous < 0 || maximum <= previous {
		return HitPointMaximumRestored{}, invalidInput("restored maximum needs a subject and a higher maximum")
	}
	return HitPointMaximumRestored{b, previous, maximum}, nil
}

// Kind returns "hit_point_maximum_reduced".
func (HitPointMaximumReduced) Kind() string { return "hit_point_maximum_reduced" }

// Kind returns "hit_point_maximum_restored".
func (HitPointMaximumRestored) Kind() string { return "hit_point_maximum_restored" }

// PreviousMaximum returns the maximum before the change.
func (e HitPointMaximumReduced) PreviousMaximum() int { return e.previous }

// Maximum returns the maximum after the change.
func (e HitPointMaximumReduced) Maximum() int { return e.maximum }

// PreviousMaximum returns the maximum before the change.
func (e HitPointMaximumRestored) PreviousMaximum() int { return e.previous }

// Maximum returns the maximum after the change.
func (e HitPointMaximumRestored) Maximum() int { return e.maximum }

// DeathSavingThrowMade: the creature made a Death Saving Throw. Kind
// death_saving_throw_made.
type DeathSavingThrowMade struct {
	hitPointEventBase
	outcome DeathSavingThrowOutcome
}

// NewDeathSavingThrowMade returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or an outcome that is not valid.
func NewDeathSavingThrowMade(subject CombatantID, outcome DeathSavingThrowOutcome) (DeathSavingThrowMade, error) {
	b, err := newBase(subject)
	if err != nil || outcome.String() == "unknown" {
		return DeathSavingThrowMade{}, invalidInput("death saving throw needs a subject and an outcome")
	}
	return DeathSavingThrowMade{b, outcome}, nil
}

// Kind returns "death_saving_throw_made".
func (DeathSavingThrowMade) Kind() string { return "death_saving_throw_made" }

// Outcome returns what the throw did.
func (e DeathSavingThrowMade) Outcome() DeathSavingThrowOutcome { return e.outcome }

// Died: the creature died. Kind died.
type Died struct {
	hitPointEventBase
	cause DeathCause
}

// NewDied returns the event. It returns an error matching
// srderr.ErrInvalidInput for a zero subject or an invalid cause.
func NewDied(subject CombatantID, cause DeathCause) (Died, error) {
	b, err := newBase(subject)
	if err != nil || !cause.Valid() {
		return Died{}, invalidInput("death needs a subject and a cause")
	}
	return Died{b, cause}, nil
}

// Kind returns "died".
func (Died) Kind() string { return "died" }

// Cause returns why the creature died.
func (e Died) Cause() DeathCause { return e.cause }
