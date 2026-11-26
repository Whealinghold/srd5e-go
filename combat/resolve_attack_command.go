package combat

import (
	"errors"
	"slices"

	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ResolveAttackField names the field of a ResolveAttackCommand that is violated.
type ResolveAttackField int

// The fields. Form and Range belong to the form-and-range pair: a form that is
// not one of the five, and a ranged form without a valid AttackRange.
const (
	ResolveAttackFieldDistance ResolveAttackField = iota + 1
	ResolveAttackFieldImprovisedProficiencyBonus
	ResolveAttackFieldSpellAttackProficiencyBonus
	ResolveAttackFieldCriticalThreshold
	ResolveAttackFieldForm
	ResolveAttackFieldRange
)

// String returns "distance", "improvised_proficiency_bonus",
// "spell_attack_proficiency_bonus", "critical_threshold", "form", "range" or "unknown".
func (f ResolveAttackField) String() string {
	return enumName([]string{"distance", "improvised_proficiency_bonus", "spell_attack_proficiency_bonus",
		"critical_threshold", "form", "range"}, int(f))
}

// ResolveAttackProblem is one violated field of NewResolveAttackCommand. It
// matches srderr.ErrInvalidInput with errors.Is and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type ResolveAttackProblem struct{ Field ResolveAttackField }

// Error renders the problem. The text is not a contract.
func (p ResolveAttackProblem) Error() string { return problemText("attack command", p.Field.String()) }

// Unwrap returns srderr.ErrInvalidInput.
func (p ResolveAttackProblem) Unwrap() error { return srderr.ErrInvalidInput }

// ResolveAttackParams are the facts of one attack, the input of
// NewResolveAttackCommand. Construct with a keyed literal. The host states every
// fact (Armor Class, cover, distances, sight, the consequences of conditions as
// Circumstances and AutomaticCritical); the engine decides none of them.
type ResolveAttackParams struct {
	// Form is the attack form.
	Form AttackForm
	// ArmorClass is the target's Armor Class before cover.
	ArmorClass d20.ArmorClass
	// Modifiers are the attack roll's modifiers, each added once: the ability modifier,
	// the optional Proficiency Bonus and any others. The same ability modifier is added to damage.
	Modifiers d20.RollModifiers
	// Distance is the distance to the target in feet, 0 or more.
	Distance int
	// Range is the range of a ranged form; ignored for a melee form.
	Range AttackRange
	// Reach is the reach of a melee form; the zero value is 5 feet. Ignored for ranged forms and Unarmed Strikes.
	Reach Reach
	// Cover lists the things that give cover.
	Cover []CoverSource
	// NearbyEnemies are the attacker's enemies near it, for ranged attacks in close combat.
	NearbyEnemies []NearbyEnemy
	// Situation holds the sight and water facts.
	Situation AttackSituation
	// Circumstances are further Advantage and Disadvantage, as the caller computed them.
	Circumstances []modifier.RollCircumstance
	// AutomaticCritical says a hit is a Critical Hit; only AutomaticCriticalAutomatic counts.
	AutomaticCritical AutomaticCritical
	// CriticalThreshold is the kept d20 at or above which the attack is a Critical Hit,
	// 18 to 20; 0 is the default 20.
	CriticalThreshold int
	// Damage is the damage of a hit; the zero value deals none.
	Damage DamageSpec
	// Weapon is how the weapon is used, for a weapon form; the zero value is not supplied.
	Weapon WeaponUse
}

// ResolveAttackCommand is the input of ResolveAttack.Run: validated attack facts,
// immutable. Build it with NewResolveAttackCommand; the zero value is invalid.
type ResolveAttackCommand struct{ p ResolveAttackParams }

// NewResolveAttackCommand returns the command for p, copying its slices.
//
// It returns the errors.Join of every ResolveAttackProblem, in the order of the
// field constants, matching srderr.ErrInvalidInput: a negative distance; an
// improvised weapon use whose modifiers carry a Proficiency Bonus; a spell attack
// whose modifiers carry none; a critical threshold outside 18 to 20; a form that
// is not valid; a ranged form without a valid range. More than 256 cover sources, nearby
// enemies or circumstances return the LimitExceededError of each (plan P10) and nothing else. It never panics.
func NewResolveAttackCommand(p ResolveAttackParams) (ResolveAttackCommand, error) {
	if err := errors.Join(overCap("cover sources", len(p.Cover), maxInputList), overCap("nearby enemies", len(p.NearbyEnemies), maxInputList),
		overCap("circumstances", len(p.Circumstances), maxInputList)); err != nil {
		return ResolveAttackCommand{}, err
	}
	var ps []ResolveAttackProblem
	_, hasPB := p.Modifiers.ProficiencyBonus()
	if p.Distance < 0 {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldDistance})
	}
	if p.Form.Source() == AttackSourceWeapon && p.Weapon.IsImprovised() && hasPB {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldImprovisedProficiencyBonus})
	}
	if p.Form.Source() == AttackSourceSpell && !hasPB {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldSpellAttackProficiencyBonus})
	}
	if !validCriticalThreshold(p.CriticalThreshold) {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldCriticalThreshold})
	}
	if !p.Form.Valid() {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldForm})
	}
	if p.Form.Delivery() == AttackDeliveryRanged && p.Range.IsZero() {
		ps = append(ps, ResolveAttackProblem{ResolveAttackFieldRange})
	}
	if err := joinProblems(ps); err != nil {
		return ResolveAttackCommand{}, err
	}
	p.Cover = slices.Clone(p.Cover)
	p.NearbyEnemies = slices.Clone(p.NearbyEnemies)
	p.Circumstances = slices.Clone(p.Circumstances)
	return ResolveAttackCommand{p}, nil
}

// MustResolveAttackCommand is NewResolveAttackCommand for values known to be
// valid. It panics with the error NewResolveAttackCommand returns, and only then.
func MustResolveAttackCommand(p ResolveAttackParams) ResolveAttackCommand {
	c, err := NewResolveAttackCommand(p)
	if err != nil {
		panic(err)
	}
	return c
}

// validCriticalThreshold reports whether t is 0 (the default) or 18 to 20.
func validCriticalThreshold(t int) bool { return t == 0 || (t >= 18 && t <= 20) }

// criticalRange returns the threshold d20 takes: 20 by default.
func criticalRange(t int) int {
	if t == 0 {
		return d20.DefaultCriticalRange
	}
	return t
}

// Form returns the attack form.
func (c ResolveAttackCommand) Form() AttackForm { return c.p.Form }

// Additions are the derived inputs WithAdditions adds to a command. The zero
// value adds nothing. Construct with a keyed literal.
type Additions struct {
	// Circumstances are appended after the command's own.
	Circumstances []modifier.RollCircumstance
	// Modifiers are appended to the others of the RollModifiers.
	Modifiers []modifier.Modifier
	// AutomaticCritical makes the command's automatic when it is AutomaticCriticalAutomatic; it never clears it.
	AutomaticCritical AutomaticCritical
}

// WithAdditions returns a copy of c with a added: circumstances after the existing
// ones, modifiers among the others (the ability modifier and the Proficiency Bonus
// are never replaced), and the automatic Critical Hit when either is. Every other
// field is carried unchanged and c is not altered. The copy is validated as the
// constructor does and returns its error, matching srderr.ErrInvalidInput, for an
// invalid c (the zero command included), and a LimitExceededError for more than 256 modifiers in a (or more than 256 circumstances in the result). It never panics.
func (c ResolveAttackCommand) WithAdditions(a Additions) (ResolveAttackCommand, error) {
	if err := overCap("addition modifiers", len(a.Modifiers), maxInputList); err != nil {
		return ResolveAttackCommand{}, err
	}
	p := c.p
	p.Circumstances = slices.Concat(p.Circumstances, a.Circumstances)
	p.Modifiers = p.Modifiers.WithOthers(a.Modifiers...)
	if a.AutomaticCritical == AutomaticCriticalAutomatic {
		p.AutomaticCritical = AutomaticCriticalAutomatic
	}
	return NewResolveAttackCommand(p)
}
