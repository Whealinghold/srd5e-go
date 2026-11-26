package combat

// TargetVisibility says whether the attacker can see the target (SRD 5.2.1,
// "Unseen Attackers and Targets", p. 14). Variants: TargetSeen, TargetUnseen. A
// seen target that is not at the targeted location cannot be built. A nil
// TargetVisibility reads as TargetSeen.
type TargetVisibility interface{ isTargetVisibility() }

// TargetSeen: the attacker can see the target.
type TargetSeen struct{}

// TargetUnseen: the attacker cannot see the target (or guesses its location).
// TargetAtTargetedLocation says whether the target is where the attacker aims.
// Construct with a keyed literal.
type TargetUnseen struct{ TargetAtTargetedLocation bool }

func (TargetSeen) isTargetVisibility()   {}
func (TargetUnseen) isTargetVisibility() {}

// AttackSituation holds the facts about sight and water the host states. Construct
// with a keyed literal; the zero value is a clear, seen, dry attack. That the
// weapon deals Piercing damage is read from the DamageSpec, not stated here.
type AttackSituation struct {
	// Visibility says whether the attacker can see the target; nil reads as TargetSeen.
	Visibility TargetVisibility
	// TargetCannotSeeAttacker gives the attack Advantage.
	TargetCannotSeeAttacker bool
	// AttackerHidden says the attacker's location is given away by the attack.
	AttackerHidden bool
	// Underwater says the attacker is underwater (SRD 5.2.1, "Underwater Combat", p. 16).
	Underwater bool
	// SwimSpeed says the attacker has a Swim Speed.
	SwimSpeed bool
}

// MissOverride says what the d20 did not decide: whether the attack misses
// whatever the roll. The zero value is invalid.
type MissOverride int

// The two overrides.
const (
	// MissOverrideNone: the roll decides.
	MissOverrideNone MissOverride = iota + 1
	// MissOverrideTargetNotAtLocation: the attacker cannot see the target and it
	// is not where the attacker aims, so the attack misses.
	MissOverrideTargetNotAtLocation
)

// String returns "none", "target_not_at_location" or "unknown".
func (o MissOverride) String() string {
	return enumName([]string{"none", "target_not_at_location"}, int(o))
}

// AutomaticMissReason says why an attack missed without a roll. The zero value
// is invalid.
type AutomaticMissReason int

// AutomaticMissReasonUnderwaterBeyondNormalRange: a ranged weapon attack
// underwater against a target beyond normal range.
const AutomaticMissReasonUnderwaterBeyondNormalRange AutomaticMissReason = iota + 1

// String returns "underwater_beyond_normal_range" or "unknown".
func (r AutomaticMissReason) String() string {
	return enumName([]string{"underwater_beyond_normal_range"}, int(r))
}

// AttackNotPossibleReason says why an attack cannot be made. When several apply
// the one reported is the first in this order. The zero value is invalid.
type AttackNotPossibleReason int

// The reasons, in reporting order.
const (
	AttackNotPossibleReasonTotalCover AttackNotPossibleReason = iota + 1
	AttackNotPossibleReasonBeyondRange
	AttackNotPossibleReasonBeyondReach
	AttackNotPossibleReasonTargetTooLarge
	AttackNotPossibleReasonNoFreeHand
)

// String returns the srd5e.combat.attack.reason value: "total_cover",
// "beyond_range", "beyond_reach", "target_too_large", "no_free_hand" or "unknown".
func (r AttackNotPossibleReason) String() string {
	return enumName([]string{"total_cover", "beyond_range", "beyond_reach", "target_too_large", "no_free_hand"}, int(r))
}
