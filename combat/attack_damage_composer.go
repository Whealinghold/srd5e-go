package combat

import (
	"github.com/Whealinghold/srd5e-go/dice"
)

// AttackDamageComposer is the stateless domain service that turns the dice rolled
// for a DamageSpec into Damage (SRD 5.2.1, "Damage Rolls", "Critical Hits",
// p. 16). It is not the DamageResolver, which applies Resistance and the rest.
// The zero value is ready to use. Safe for concurrent use.
type AttackDamageComposer struct{}

// Dice returns the dice to roll for c: its own count, or twice that on a Critical
// Hit, and false for a fixed amount, which has no dice to double.
func (AttackDamageComposer) Dice(c DamageComponent, critical bool) (dice.Expression, bool) {
	d, ok := c.(DiceDamage)
	if !ok {
		return dice.Expression{}, false
	}
	n := d.count
	if critical {
		n *= 2
	}
	return dice.MustExpression(n, d.die, 0), true
}

// Compose returns one Damage per component of spec, in order. rolls[i] holds the
// faces rolled for component i (none for a fixed amount). Each total is the
// faces, plus the component's modifier and, for the one flagged component, the
// ability modifier, floored at 0 per component; modifiers are never doubled.
//
// It returns an error matching srderr.ErrInvalidInput, naming no field, when rolls
// does not match what Dice asks for. It never panics.
func (c AttackDamageComposer) Compose(spec DamageSpec, rolls [][]int, abilityModifier int, critical bool) ([]Damage, error) {
	if len(rolls) != len(spec.components) {
		return nil, invalidInput("damage rolls do not match the components")
	}
	out := make([]Damage, len(spec.components))
	for i, comp := range spec.components {
		want := 0
		if e, ok := c.Dice(comp, critical); ok {
			want = e.Count()
		}
		if len(rolls[i]) != want {
			return nil, invalidInput("damage rolls do not match the dice")
		}
		total := comp.Modifier()
		if f, ok := comp.(FixedDamage); ok {
			total = satAdd(total, f.amount)
		}
		for _, face := range rolls[i] {
			total = satAdd(total, face)
		}
		if comp.AddsAbilityModifier() {
			total = satAdd(total, abilityModifier)
		}
		dmg, err := NewDamage(max(total, 0), comp.Type())
		if err != nil {
			return nil, err
		}
		out[i] = dmg
	}
	return out, nil
}
