package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ShoveEffect is the effect a Shove has on a target that fails its save, declared
// by the attacker in the command (SRD 5.2.1, "Unarmed Strike", p. 190). The zero
// value is invalid.
type ShoveEffect int

// The two effects.
const (
	// PushFiveFeet: the target is pushed 5 feet away.
	PushFiveFeet ShoveEffect = iota + 1
	// GiveProne: the target has the Prone condition.
	GiveProne
)

// String returns "push_five_feet", "give_prone" or "unknown".
func (e ShoveEffect) String() string {
	return enumName([]string{"push_five_feet", "give_prone"}, int(e))
}

// UnarmedStrikeOption is the one option an Unarmed Strike uses with its inputs
// (SRD 5.2.1, "Unarmed Strike", p. 190). Variants: UnarmedStrikeDamageOption, UnarmedStrikeGrappleOption,
// UnarmedStrikeShoveOption. Construct each with a keyed literal.
type UnarmedStrikeOption interface{ isUnarmedStrikeOption() }

// UnarmedStrikeDamageOption: an attack roll against ArmorClass for Bludgeoning damage.
// Circumstances are the caller's Advantage and Disadvantage; CriticalThreshold is
// 18 to 20, or 0 for the default 20.
type UnarmedStrikeDamageOption struct {
	ArmorClass        d20.ArmorClass
	Circumstances     []modifier.RollCircumstance
	CriticalThreshold int
}

// UnarmedStrikeGrappleOption: the target makes a Strength or Dexterity saving throw (SaveAbility
// is its choice) as the Attempt describes, against the Grapple DC.
type UnarmedStrikeGrappleOption struct {
	SaveAbility ability.Ability
	Attempt     d20.SaveAttempt
}

// UnarmedStrikeShoveOption: the target makes a Strength or Dexterity saving throw against the
// Shove DC; on a failure the attacker's declared Effect applies.
type UnarmedStrikeShoveOption struct {
	Effect      ShoveEffect
	SaveAbility ability.Ability
	Attempt     d20.SaveAttempt
}

func (UnarmedStrikeDamageOption) isUnarmedStrikeOption()  {}
func (UnarmedStrikeGrappleOption) isUnarmedStrikeOption() {}
func (UnarmedStrikeShoveOption) isUnarmedStrikeOption()   {}

// ResolveUnarmedStrikeField names the field of a ResolveUnarmedStrikeCommand that
// is violated.
type ResolveUnarmedStrikeField int

// The fields. Distance, CriticalThreshold and SaveAbility are those the rules
// name; the others guard the closed inputs.
const (
	ResolveUnarmedStrikeFieldDistance ResolveUnarmedStrikeField = iota + 1
	ResolveUnarmedStrikeFieldCriticalThreshold
	ResolveUnarmedStrikeFieldSaveAbility
	ResolveUnarmedStrikeFieldOption
	ResolveUnarmedStrikeFieldAttempt
	ResolveUnarmedStrikeFieldEffect
	ResolveUnarmedStrikeFieldSize
	ResolveUnarmedStrikeFieldProficiencyBonus
)

// String returns the snake_case field name or "unknown".
func (f ResolveUnarmedStrikeField) String() string {
	return enumName([]string{"distance", "critical_threshold", "save_ability", "option", "attempt", "effect", "size",
		"proficiency_bonus"}, int(f))
}

// ResolveUnarmedStrikeProblem is one violated field of
// NewResolveUnarmedStrikeCommand. It matches srderr.ErrInvalidInput with errors.Is
// and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type ResolveUnarmedStrikeProblem struct{ Field ResolveUnarmedStrikeField }

// Error renders the problem. The text is not a contract.
func (p ResolveUnarmedStrikeProblem) Error() string {
	return problemText("unarmed strike command", p.Field.String())
}

// Unwrap returns srderr.ErrInvalidInput.
func (p ResolveUnarmedStrikeProblem) Unwrap() error { return srderr.ErrInvalidInput }

// ResolveUnarmedStrikeParams are the facts of one Unarmed Strike, the input of
// NewResolveUnarmedStrikeCommand. Construct with a keyed literal.
type ResolveUnarmedStrikeParams struct {
	// Option is the option used, with its inputs.
	Option UnarmedStrikeOption
	// AbilityModifier is the ability modifier used: the Strength modifier unless the
	// caller applies a rule that uses another. Roll, damage and both DCs use this one value.
	AbilityModifier ability.Modifier
	// ProficiencyBonus is always added, without a proficiency check; it is not the zero value.
	ProficiencyBonus character.ProficiencyBonus
	// AttackerSize and TargetSize are the sizes the Grapple and Shove checks compare.
	AttackerSize, TargetSize content.CreatureSize
	// Hand says whether the attacker has a free hand, read by the Grapple option.
	Hand HandAvailability
	// Distance is the distance to the target in feet, 0 or more; an Unarmed Strike reaches 5 feet.
	Distance int
	// Cover lists the things that give cover.
	Cover []CoverSource
	// AutomaticCritical says a hit of the Damage option is a Critical Hit.
	AutomaticCritical AutomaticCritical
}

// ResolveUnarmedStrikeCommand is the input of ResolveUnarmedStrike.Run: validated
// facts, immutable. Build it with NewResolveUnarmedStrikeCommand; the zero value
// is invalid.
type ResolveUnarmedStrikeCommand struct{ p ResolveUnarmedStrikeParams }

// NewResolveUnarmedStrikeCommand returns the command for p, copying its slices.
//
// It returns the errors.Join of every ResolveUnarmedStrikeProblem, in the order of
// the field constants, matching srderr.ErrInvalidInput: a negative distance; a
// critical threshold of the Damage option outside 18 to 20; a save ability that is
// neither Strength nor Dexterity; a missing or unknown option; a missing save
// attempt; a Shove effect that is not valid; a size that is not valid for a Grapple
// or Shove; a zero Proficiency Bonus. It never panics.
func NewResolveUnarmedStrikeCommand(p ResolveUnarmedStrikeParams) (ResolveUnarmedStrikeCommand, error) {
	var ps []ResolveUnarmedStrikeProblem
	add := func(f ResolveUnarmedStrikeField) { ps = append(ps, ResolveUnarmedStrikeProblem{f}) }
	validSave := func(a ability.Ability) bool { return a == ability.Strength || a == ability.Dexterity }
	saves := func(a ability.Ability, at d20.SaveAttempt) {
		if !validSave(a) {
			add(ResolveUnarmedStrikeFieldSaveAbility)
		}
		if at == nil {
			add(ResolveUnarmedStrikeFieldAttempt)
		}
		if !p.AttackerSize.Valid() || !p.TargetSize.Valid() {
			add(ResolveUnarmedStrikeFieldSize)
		}
	}
	if p.Distance < 0 {
		add(ResolveUnarmedStrikeFieldDistance)
	}
	switch o := p.Option.(type) {
	case UnarmedStrikeDamageOption:
		if !validCriticalThreshold(o.CriticalThreshold) {
			add(ResolveUnarmedStrikeFieldCriticalThreshold)
		}
		o.Circumstances = slices.Clone(o.Circumstances)
		p.Option = o
	case UnarmedStrikeGrappleOption:
		saves(o.SaveAbility, o.Attempt)
	case UnarmedStrikeShoveOption:
		saves(o.SaveAbility, o.Attempt)
		if o.Effect != PushFiveFeet && o.Effect != GiveProne {
			add(ResolveUnarmedStrikeFieldEffect)
		}
	default:
		add(ResolveUnarmedStrikeFieldOption)
	}
	if p.ProficiencyBonus.IsZero() {
		add(ResolveUnarmedStrikeFieldProficiencyBonus)
	}
	if err := joinProblems(ps); err != nil {
		return ResolveUnarmedStrikeCommand{}, err
	}
	p.Cover = slices.Clone(p.Cover)
	return ResolveUnarmedStrikeCommand{p}, nil
}

// MustResolveUnarmedStrikeCommand is NewResolveUnarmedStrikeCommand for values
// known to be valid. It panics with the error that function returns, and only then.
func MustResolveUnarmedStrikeCommand(p ResolveUnarmedStrikeParams) ResolveUnarmedStrikeCommand {
	c, err := NewResolveUnarmedStrikeCommand(p)
	if err != nil {
		panic(err)
	}
	return c
}
