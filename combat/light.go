package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// SpaceVisibilityReason says why NewSpaceVisibility rejected its arguments. The
// zero value is not a reason.
type SpaceVisibilityReason int

// The reasons a SpaceVisibility is rejected.
const (
	// SpaceVisibilityReasonInvalidLight: the light level is not one of the three.
	SpaceVisibilityReasonInvalidLight SpaceVisibilityReason = iota + 1
	// SpaceVisibilityReasonInvalidDarknessKind: the kind is neither the zero value nor a defined kind.
	SpaceVisibilityReasonInvalidDarknessKind
	// SpaceVisibilityReasonInvalidObscurement: the other obscurement is neither the zero value nor a defined level.
	SpaceVisibilityReasonInvalidObscurement
	// SpaceVisibilityReasonMagicalWithoutDarkness: magical Darkness was asked for in a light other than Darkness.
	SpaceVisibilityReasonMagicalWithoutDarkness
)

// String returns the snake_case name of r or "unknown".
func (r SpaceVisibilityReason) String() string {
	return enumName([]string{"invalid_light", "invalid_darkness_kind", "invalid_obscurement", "magical_without_darkness"}, int(r))
}

// SpaceVisibilityError reports a rejected SpaceVisibility. It matches
// srderr.ErrInvalidInput with errors.Is and carries no display text.
type SpaceVisibilityError struct{ Reason SpaceVisibilityReason }

// Error renders the error. The text is not a contract.
func (e *SpaceVisibilityError) Error() string {
	return "combat: invalid space visibility: " + e.Reason.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (e *SpaceVisibilityError) Unwrap() error { return srderr.ErrInvalidInput }

// Is reports whether target is a *SpaceVisibilityError with the same Reason.
func (e *SpaceVisibilityError) Is(target error) bool {
	t, ok := target.(*SpaceVisibilityError)
	return ok && t != nil && e != nil && *t == *e
}

// SpaceVisibility is what makes a space hard to see into: its light, the kind
// of Darkness and any other obscuring cause such as fog or foliage (SRD 5.2.1,
// "Obscured Areas", p. 11). Immutable and comparable; the zero SpaceVisibility
// is invalid (Valid reports false). Safe for concurrent use.
type SpaceVisibility struct {
	light content.LightLevel
	kind  content.DarknessKind
	other content.Obscurement
}

// NewSpaceVisibility returns the visibility of a space. A zero darknessKind means
// nonmagical Darkness and a zero otherObscurement means none. It returns a
// *SpaceVisibilityError (matching srderr.ErrInvalidInput) for an undefined light
// level, kind or obscurement, and for magical Darkness in a light other than
// Darkness (SRD 5.2.1, "Truesight", p. 190). It never panics.
func NewSpaceVisibility(light content.LightLevel, darknessKind content.DarknessKind, otherObscurement content.Obscurement) (SpaceVisibility, error) {
	if darknessKind == 0 {
		darknessKind = content.DarknessNonmagical
	}
	if otherObscurement == 0 {
		otherObscurement = content.ObscurementNone
	}
	switch {
	case !light.Valid():
		return SpaceVisibility{}, &SpaceVisibilityError{SpaceVisibilityReasonInvalidLight}
	case !darknessKind.Valid():
		return SpaceVisibility{}, &SpaceVisibilityError{SpaceVisibilityReasonInvalidDarknessKind}
	case !otherObscurement.Valid():
		return SpaceVisibility{}, &SpaceVisibilityError{SpaceVisibilityReasonInvalidObscurement}
	case darknessKind == content.DarknessMagical && light != content.LightDarkness:
		return SpaceVisibility{}, &SpaceVisibilityError{SpaceVisibilityReasonMagicalWithoutDarkness}
	}
	return SpaceVisibility{light, darknessKind, otherObscurement}, nil
}

// MustSpaceVisibility is NewSpaceVisibility for literals known to be valid. It
// panics with the error NewSpaceVisibility returns, and only then.
func MustSpaceVisibility(light content.LightLevel, darknessKind content.DarknessKind, otherObscurement content.Obscurement) SpaceVisibility {
	v, err := NewSpaceVisibility(light, darknessKind, otherObscurement)
	if err != nil {
		panic(err)
	}
	return v
}

// Valid reports whether v was built by NewSpaceVisibility.
func (v SpaceVisibility) Valid() bool { return v.light.Valid() }

// Light returns the light level of the space.
func (v SpaceVisibility) Light() content.LightLevel { return v.light }

// DarknessKind returns the kind of Darkness; it matters only when the light is Darkness.
func (v SpaceVisibility) DarknessKind() content.DarknessKind { return v.kind }

// OtherObscurement returns the obscurement from causes other than light.
func (v SpaceVisibility) OtherObscurement() content.Obscurement { return v.other }

// Obscurement returns the more severe of the obscurement of the light and the
// other obscurement (SRD 5.2.1, "Obscured Areas", p. 11). It is the zero
// Obscurement for an invalid SpaceVisibility.
func (v SpaceVisibility) Obscurement() content.Obscurement {
	if !v.Valid() {
		return 0
	}
	return max(LightRules{}.ObscurementOf(v.light), v.other)
}

// SightCheckKind says what a check needs in order to be affected by obscurement.
// The zero value is invalid.
type SightCheckKind int

// The kinds of check made to see.
const (
	// SightCheckKindPerceptionToSee: a Wisdom (Perception) check made to see something.
	SightCheckKindPerceptionToSee SightCheckKind = iota + 1
	// SightCheckKindOtherCheckRequiringSight: any other check that relies on sight.
	SightCheckKindOtherCheckRequiringSight
	// SightCheckKindCheckNotRequiringSight: a check that does not rely on sight.
	SightCheckKindCheckNotRequiringSight
)

// String returns the snake_case name of k or "unknown".
func (k SightCheckKind) String() string {
	return enumName([]string{"perception_to_see", "other_check_requiring_sight", "check_not_requiring_sight"}, int(k))
}

// SightCheckEffect is what obscurement does to a check. The zero value is invalid.
type SightCheckEffect int

// The effects on a check made to see.
const (
	// SightCheckEffectUnaffected: the check is rolled normally.
	SightCheckEffectUnaffected SightCheckEffect = iota + 1
	// SightCheckEffectDisadvantage: the check is rolled with Disadvantage.
	SightCheckEffectDisadvantage
	// SightCheckEffectAutomaticFailure: the check fails without a roll.
	SightCheckEffectAutomaticFailure
)

// String returns the snake_case name of e or "unknown".
func (e SightCheckEffect) String() string {
	return enumName([]string{"unaffected", "disadvantage", "automatic_failure"}, int(e))
}

// LightRules is the stateless domain service for light and obscurement (SRD
// 5.2.1, "Vision and Light", "Obscured Areas", p. 11). Its methods are pure,
// synchronous and silent. The zero value is ready to use. Safe for concurrent use.
type LightRules struct{}

// ObscurementOf returns the obscurement a light level causes: none for Bright
// Light, Lightly Obscured for Dim Light, Heavily Obscured for Darkness (SRD 5.2.1,
// "Bright Light", "Dim Light", "Darkness", pp. 178, 180, 181). It returns the zero
// Obscurement for an undefined level.
func (LightRules) ObscurementOf(level content.LightLevel) content.Obscurement {
	switch level {
	case content.LightBright:
		return content.ObscurementNone
	case content.LightDim:
		return content.ObscurementLightly
	case content.LightDarkness:
		return content.ObscurementHeavily
	}
	return 0
}

// SightCheckEffect returns what the obscurement of the space of the thing seen
// does to a check: Disadvantage on a Wisdom (Perception) check to see in a
// Lightly Obscured space, an automatic failure for a check that relies on sight in
// a Heavily Obscured one, and nothing otherwise (SRD 5.2.1, "Lightly Obscured",
// p. 184; "Heavily Obscured", p. 182). An undefined obscurement or kind is
// Unaffected.
func (LightRules) SightCheckEffect(o content.Obscurement, kind SightCheckKind) SightCheckEffect {
	switch {
	case o == content.ObscurementHeavily && (kind == SightCheckKindPerceptionToSee || kind == SightCheckKindOtherCheckRequiringSight):
		return SightCheckEffectAutomaticFailure
	case o == content.ObscurementLightly && kind == SightCheckKindPerceptionToSee:
		return SightCheckEffectDisadvantage
	}
	return SightCheckEffectUnaffected
}

// AttemptConditions returns a new slice of the conditions a creature has while it
// tries to see in a space: Blinded for Heavily Obscured, none otherwise (SRD 5.2.1,
// "Heavily Obscured", p. 182). It is derived per attempt and changes no ConditionSet.
func (LightRules) AttemptConditions(o content.Obscurement) []content.ConditionKind {
	if o == content.ObscurementHeavily {
		return []content.ConditionKind{content.ConditionBlinded}
	}
	return []content.ConditionKind{}
}

// Circumstances returns a new slice holding modifier.Disadvantage for
// SightCheckEffectDisadvantage and nothing for the other effects, for an active
// check and Passive Perception alike (SRD 5.2.1, "Passive Perception", p. 186).
func (LightRules) Circumstances(e SightCheckEffect) []modifier.RollCircumstance {
	if e == SightCheckEffectDisadvantage {
		return []modifier.RollCircumstance{modifier.Disadvantage{}}
	}
	return []modifier.RollCircumstance{}
}
