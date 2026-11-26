package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// The named states of the grapple rules. Each is a typed enum with a String
// method; the zero value of each is invalid. All are comparable and safe for
// concurrent use.

// PartState says whether a part already holds a grappled creature (SRD 5.2.1,
// "Grappling", "One Grapple per Hand", p. 182). It is not the SRD's "a hand free
// to grab it", which the host states with HandAvailability.
type PartState int

// The two states.
const (
	PartStateHolding PartState = iota + 1
	PartStateNotHoldingGrapple
)

// String returns "holding", "not_holding_grapple" or "unknown".
func (s PartState) String() string {
	return enumName([]string{"holding", "not_holding_grapple"}, int(s))
}

// HandAvailability is the host's statement whether the attacker has a hand free
// to grab with (SRD 5.2.1, "Unarmed Strike", p. 190). The engine models no held
// items. Any value other than HandAvailabilityHandFree counts as no free hand.
type HandAvailability int

// The two statements.
const (
	HandAvailabilityHandFree HandAvailability = iota + 1
	HandAvailabilityNoHandFree
)

// String returns "hand_free", "no_hand_free" or "unknown".
func (h HandAvailability) String() string {
	return enumName([]string{"hand_free", "no_hand_free"}, int(h))
}

// GrappleHold is whether a grapple holds (SRD 5.2.1, "Grappling", "Ending a
// Grapple", p. 182).
type GrappleHold int

// The three answers.
const (
	GrappleHoldHolds GrappleHold = iota + 1
	GrappleHoldGrapplerIncapacitated
	GrappleHoldOutOfRange
)

// String returns "holds", "grappler_incapacitated", "out_of_range" or "unknown".
func (h GrappleHold) String() string {
	return enumName([]string{"holds", "grappler_incapacitated", "out_of_range"}, int(h))
}

// GrappleEndCause is how a grapple ended, one per way the SRD ends one (SRD
// 5.2.1, "Grappling", "Ending a Grapple", p. 182).
type GrappleEndCause int

// The four causes.
const (
	GrappleEndCauseEscaped GrappleEndCause = iota + 1
	GrappleEndCauseGrapplerIncapacitated
	GrappleEndCauseOutOfRange
	GrappleEndCauseReleased
)

// String returns "escaped", "grappler_incapacitated", "out_of_range", "released"
// or "unknown"; the value of srd5e.combat.grapple.end_reason.
func (c GrappleEndCause) String() string {
	return enumName([]string{"escaped", "grappler_incapacitated", "out_of_range", "released"}, int(c))
}

// Valid reports whether c is one of the four causes.
func (c GrappleEndCause) Valid() bool {
	return c >= GrappleEndCauseEscaped && c <= GrappleEndCauseReleased
}

// GrappleNotPossibleReason is why a creature cannot use the Grapple option of the
// Unarmed Strike (SRD 5.2.1, "Unarmed Strike", p. 190).
type GrappleNotPossibleReason int

// The two reasons, in the order they are checked.
const (
	GrappleNotPossibleReasonTargetTooLarge GrappleNotPossibleReason = iota + 1
	GrappleNotPossibleReasonNoFreeHand
)

// String returns "target_too_large", "no_free_hand" or "unknown".
func (r GrappleNotPossibleReason) String() string {
	return enumName([]string{"target_too_large", "no_free_hand"}, int(r))
}

// GrappleEscapeOption is the check a Grappled creature makes to escape: Strength
// (Athletics) or Dexterity (Acrobatics) (SRD 5.2.1, "Grappling", p. 182).
type GrappleEscapeOption int

// The two options.
const (
	GrappleEscapeOptionStrengthAthletics GrappleEscapeOption = iota + 1
	GrappleEscapeOptionDexterityAcrobatics
)

// String returns "strength_athletics", "dexterity_acrobatics" or "unknown".
func (o GrappleEscapeOption) String() string {
	return enumName([]string{"strength_athletics", "dexterity_acrobatics"}, int(o))
}

// Valid reports whether o is one of the two options.
func (o GrappleEscapeOption) Valid() bool {
	return o == GrappleEscapeOptionStrengthAthletics || o == GrappleEscapeOptionDexterityAcrobatics
}

// Skill returns Athletics or Acrobatics, or the zero Skill for an invalid option.
func (o GrappleEscapeOption) Skill() content.Skill {
	switch o {
	case GrappleEscapeOptionStrengthAthletics:
		return content.SkillAthletics
	case GrappleEscapeOptionDexterityAcrobatics:
		return content.SkillAcrobatics
	}
	return 0
}

// Ability returns the ability of the option's skill, Strength or Dexterity, or
// the zero Ability for an invalid option.
func (o GrappleEscapeOption) Ability() ability.Ability {
	if !o.Valid() {
		return 0
	}
	return o.Skill().Ability()
}
