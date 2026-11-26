package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// DamageComponentField names the field of a DamageComponent that is violated.
type DamageComponentField int

// The fields.
const (
	DamageComponentFieldDiceCount DamageComponentField = iota + 1
	DamageComponentFieldDie
	DamageComponentFieldType
	DamageComponentFieldAmount
)

// String returns "dice_count", "die", "type", "amount" or "unknown".
func (f DamageComponentField) String() string {
	return enumName([]string{"dice_count", "die", "type", "amount"}, int(f))
}

// DamageComponentProblem is one violated field of NewDiceDamage or
// NewFixedDamage. It matches srderr.ErrInvalidInput with errors.Is and is
// comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type DamageComponentProblem struct{ Field DamageComponentField }

// Error renders the problem. The text is not a contract.
func (p DamageComponentProblem) Error() string {
	return problemText("damage component", p.Field.String())
}

// Unwrap returns srderr.ErrInvalidInput.
func (p DamageComponentProblem) Unwrap() error { return srderr.ErrInvalidInput }

// DamageComponent is dice or a fixed amount of damage of one Damage Type with
// its flat modifier (SRD 5.2.1, "Damage Rolls", p. 16). Variants: DiceDamage,
// FixedDamage. A component's modifier attaches to it alone, and at most one
// component of a DamageSpec adds the attack's ability modifier.
type DamageComponent interface {
	isDamageComponent()
	// Type returns the Damage Type.
	Type() content.DamageType
	// Modifier returns the flat modifier of this component, possibly negative.
	Modifier() int
	// AddsAbilityModifier reports whether this component receives the ability modifier.
	AddsAbilityModifier() bool
}

// DiceDamage is damage rolled with dice. Build it with NewDiceDamage; the zero
// value is invalid. Comparable and safe for concurrent use.
type DiceDamage struct {
	count       int
	die         dice.Die
	typ         content.DamageType
	modifier    int
	addsAbility bool
}

// NewDiceDamage returns count dice of die dealing damage of type t, plus modifier
// and, when addsAbilityModifier, the attack's ability modifier.
//
// It returns the errors.Join of every DamageComponentProblem (count outside 1 to
// half of dice.MaxCount so a Critical Hit can double it, a zero die, an invalid
// type), matching srderr.ErrInvalidInput. It never panics.
func NewDiceDamage(count int, die dice.Die, t content.DamageType, modifier int, addsAbilityModifier bool) (DiceDamage, error) {
	var ps []DamageComponentProblem
	if count < 1 || count > dice.MaxCount/2 {
		ps = append(ps, DamageComponentProblem{DamageComponentFieldDiceCount})
	}
	if _, err := dice.NewExpression(1, die, 0); err != nil {
		ps = append(ps, DamageComponentProblem{DamageComponentFieldDie})
	}
	if !t.Valid() {
		ps = append(ps, DamageComponentProblem{DamageComponentFieldType})
	}
	if err := joinProblems(ps); err != nil {
		return DiceDamage{}, err
	}
	return DiceDamage{count, die, t, modifier, addsAbilityModifier}, nil
}

// MustDiceDamage is NewDiceDamage for values known to be valid. It panics with
// the error NewDiceDamage returns, and only then.
func MustDiceDamage(count int, die dice.Die, t content.DamageType, modifier int, addsAbilityModifier bool) DiceDamage {
	d, err := NewDiceDamage(count, die, t, modifier, addsAbilityModifier)
	if err != nil {
		panic(err)
	}
	return d
}

// Count returns the number of dice rolled on a hit that is not a Critical Hit.
func (d DiceDamage) Count() int { return d.count }

// Die returns the die.
func (d DiceDamage) Die() dice.Die { return d.die }

// Type returns the Damage Type.
func (d DiceDamage) Type() content.DamageType { return d.typ }

// Modifier returns the flat modifier.
func (d DiceDamage) Modifier() int { return d.modifier }

// AddsAbilityModifier reports whether the component receives the ability modifier.
func (d DiceDamage) AddsAbilityModifier() bool { return d.addsAbility }

// FixedDamage is damage of a fixed amount, which a Critical Hit does not change.
// Build it with NewFixedDamage; the zero value is invalid. Comparable and safe
// for concurrent use.
type FixedDamage struct {
	amount      int
	typ         content.DamageType
	modifier    int
	addsAbility bool
}

// NewFixedDamage returns amount damage of type t, plus modifier and, when
// addsAbilityModifier, the attack's ability modifier.
//
// It returns the errors.Join of every DamageComponentProblem (a negative amount,
// an invalid type), matching srderr.ErrInvalidInput. It never panics.
func NewFixedDamage(amount int, t content.DamageType, modifier int, addsAbilityModifier bool) (FixedDamage, error) {
	var ps []DamageComponentProblem
	if amount < 0 {
		ps = append(ps, DamageComponentProblem{DamageComponentFieldAmount})
	}
	if !t.Valid() {
		ps = append(ps, DamageComponentProblem{DamageComponentFieldType})
	}
	if err := joinProblems(ps); err != nil {
		return FixedDamage{}, err
	}
	return FixedDamage{amount, t, modifier, addsAbilityModifier}, nil
}

// MustFixedDamage is NewFixedDamage for values known to be valid. It panics with
// the error NewFixedDamage returns, and only then.
func MustFixedDamage(amount int, t content.DamageType, modifier int, addsAbilityModifier bool) FixedDamage {
	d, err := NewFixedDamage(amount, t, modifier, addsAbilityModifier)
	if err != nil {
		panic(err)
	}
	return d
}

// Amount returns the fixed amount.
func (d FixedDamage) Amount() int { return d.amount }

// Type returns the Damage Type.
func (d FixedDamage) Type() content.DamageType { return d.typ }

// Modifier returns the flat modifier.
func (d FixedDamage) Modifier() int { return d.modifier }

// AddsAbilityModifier reports whether the component receives the ability modifier.
func (d FixedDamage) AddsAbilityModifier() bool { return d.addsAbility }

func (DiceDamage) isDamageComponent()  {}
func (FixedDamage) isDamageComponent() {}

// DamageSpecField names the field of a DamageSpec that is violated.
type DamageSpecField int

// The fields.
const (
	DamageSpecFieldComponents DamageSpecField = iota + 1
	DamageSpecFieldAddsAbilityModifier
)

// String returns "components", "adds_ability_modifier" or "unknown".
func (f DamageSpecField) String() string {
	return enumName([]string{"components", "adds_ability_modifier"}, int(f))
}

// DamageSpecProblem is one violated field of NewDamageSpec. It matches
// srderr.ErrInvalidInput with errors.Is and is comparable.
//
//nolint:errname // the typed problem named by the attacks spec
type DamageSpecProblem struct{ Field DamageSpecField }

// Error renders the problem. The text is not a contract.
func (p DamageSpecProblem) Error() string { return problemText("damage spec", p.Field.String()) }

// Unwrap returns srderr.ErrInvalidInput.
func (p DamageSpecProblem) Unwrap() error { return srderr.ErrInvalidInput }

// DamageSpec is the damage of one attack: at least one component, at most one of
// which adds the ability modifier. An attack that deals no damage has no
// DamageSpec at all, which is the zero value. Build it with NewDamageSpec. Not
// comparable with ==; the components are copied on the way in and out.
type DamageSpec struct{ components []DamageComponent }

// NewDamageSpec returns the spec of components, in order.
//
// It returns the errors.Join of every DamageSpecProblem (no component or a nil
// one; more than one component adding the ability modifier), matching
// srderr.ErrInvalidInput. More than 64 components return a LimitExceededError (plan P10) and nothing else.
// It never panics.
func NewDamageSpec(components ...DamageComponent) (DamageSpec, error) {
	if err := overCap("damage components", len(components), maxDamageParts); err != nil {
		return DamageSpec{}, err
	}
	var ps []DamageSpecProblem
	flagged := 0
	if len(components) == 0 {
		ps = append(ps, DamageSpecProblem{DamageSpecFieldComponents})
	}
	for _, c := range components {
		switch c.(type) {
		case DiceDamage, FixedDamage:
			if !c.Type().Valid() { // a zero-value component
				ps = append(ps, DamageSpecProblem{DamageSpecFieldComponents})
			} else if c.AddsAbilityModifier() {
				flagged++
			}
		default:
			ps = append(ps, DamageSpecProblem{DamageSpecFieldComponents})
		}
	}
	if flagged > 1 {
		ps = append(ps, DamageSpecProblem{DamageSpecFieldAddsAbilityModifier})
	}
	if err := joinProblems(ps); err != nil {
		return DamageSpec{}, err
	}
	return DamageSpec{slices.Clone(components)}, nil
}

// MustDamageSpec is NewDamageSpec for values known to be valid. It panics with
// the error NewDamageSpec returns, and only then.
func MustDamageSpec(components ...DamageComponent) DamageSpec {
	s, err := NewDamageSpec(components...)
	if err != nil {
		panic(err)
	}
	return s
}

// Components returns a copy of the components.
func (s DamageSpec) Components() []DamageComponent { return slices.Clone(s.components) }

// IsZero reports whether s is the zero value: an attack that deals no damage.
func (s DamageSpec) IsZero() bool { return len(s.components) == 0 }

// dealsPiercing reports whether any component is Piercing damage.
func (s DamageSpec) dealsPiercing() bool {
	return slices.ContainsFunc(s.components, func(c DamageComponent) bool { return c.Type() == content.DamagePiercing })
}
