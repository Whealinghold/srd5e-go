package combat

// ResolutionEvent is the closed sub-family of Event for the facts only combat action resolution
// states (combat-action-resolution.md FR-094): immutable values equal by value, in the past tense, each
// embedding resolutionEventBase. A prompt is named for the happening (...BecameDue), never as a request.
// Variants: AttackPerformed, UnarmedStrikePerformed, ConcentrationChecked,
// MovementSpeedsReviewBecameDue, GrappleCheckBecameDue.
type ResolutionEvent interface {
	Event
	resolutionEvent()
	// EncounterID returns the encounter the fact happened in.
	EncounterID() EncounterID
	// Round returns the round it happened in.
	Round() int
}

// resolutionEventBase is the base of every ResolutionEvent: the encounter and the round.
type resolutionEventBase struct {
	id    EncounterID
	round int
}

func (resolutionEventBase) event()           {}
func (resolutionEventBase) resolutionEvent() {}

// EncounterID returns the encounter the fact happened in.
func (b resolutionEventBase) EncounterID() EncounterID { return b.id }

// Round returns the round it happened in.
func (b resolutionEventBase) Round() int { return b.round }

// AttackPerformed: an attack was made, and the outcome says how it came out. It is produced for an
// attack that was not AttackNotPossible; the damage that followed is a separate DamageTaken.
type AttackPerformed struct {
	resolutionEventBase
	attacker, target CombatantID
	outcome          AttackPerformedOutcome
}

// Kind returns "attack_performed".
func (AttackPerformed) Kind() string { return "attack_performed" }

// Combatants returns the attacker and then the target, in a new slice.
func (e AttackPerformed) Combatants() []CombatantID { return []CombatantID{e.attacker, e.target} }

// Outcome returns what the attack came to.
func (e AttackPerformed) Outcome() AttackPerformedOutcome { return e.outcome }

// UnarmedStrikePerformed: an Unarmed Strike was made.
type UnarmedStrikePerformed struct {
	resolutionEventBase
	attacker, target CombatantID
	outcome          UnarmedStrikePerformedOutcome
}

// Kind returns "unarmed_strike_performed".
func (UnarmedStrikePerformed) Kind() string { return "unarmed_strike_performed" }

// Combatants returns the attacker and then the target, in a new slice.
func (e UnarmedStrikePerformed) Combatants() []CombatantID {
	return []CombatantID{e.attacker, e.target}
}

// Outcome returns what the Unarmed Strike came to.
func (e UnarmedStrikePerformed) Outcome() UnarmedStrikePerformedOutcome { return e.outcome }

// MovementSpeedsReviewBecameDue: a combatant gained or lost a condition that changes its Speed, so the
// host recomputes its Speeds and calls UpdateMovementSpeeds (SRD 5.2.1, "Changes to Your Speeds", p. 188).
type MovementSpeedsReviewBecameDue struct {
	resolutionEventBase
	combatant CombatantID
}

// Kind returns "movement_speeds_review_became_due".
func (MovementSpeedsReviewBecameDue) Kind() string { return "movement_speeds_review_became_due" }

// Combatants returns the combatant, in a new slice.
func (e MovementSpeedsReviewBecameDue) Combatants() []CombatantID { return []CombatantID{e.combatant} }

// GrappleCheckBecameDue: a grappler became Incapacitated or died, so the host checks the grapple of the
// grappled combatant with CheckCombatantGrapple (SRD 5.2.1, "Grappling", p. 182).
type GrappleCheckBecameDue struct {
	resolutionEventBase
	grappled, grappler CombatantID
}

// Kind returns "grapple_check_became_due".
func (GrappleCheckBecameDue) Kind() string { return "grapple_check_became_due" }

// Combatants returns the grappled combatant and then the grappler, in a new slice.
func (e GrappleCheckBecameDue) Combatants() []CombatantID {
	return []CombatantID{e.grappled, e.grappler}
}

// ConcentrationChecked: the Concentration of a damaged combatant was checked, so it held, ended with a failed
// save, or ended because the combatant is Incapacitated or dead (SRD 5.2.1, "Concentration", p. 179). It is
// produced only when the outcome is not ConcentrationNotChecked, last in the stream of a call (FR-088). The
// events of the break itself are the outcome's and are `character` events, not part of the combat stream.
type ConcentrationChecked struct {
	resolutionEventBase
	combatant CombatantID
	outcome   ConcentrationCheckOutcome
}

// Kind returns "concentration_checked".
func (ConcentrationChecked) Kind() string { return "concentration_checked" }

// Combatants returns the combatant, in a new slice.
func (e ConcentrationChecked) Combatants() []CombatantID { return []CombatantID{e.combatant} }

// Outcome returns how the check came out.
func (e ConcentrationChecked) Outcome() ConcentrationCheckOutcome { return e.outcome }
