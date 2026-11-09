package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SpellSaveDC is a spell save DC: 8 plus the spellcasting ability modifier plus
// the Proficiency Bonus (SRD 5.2.1, "Spells", "Saving Throws", p. 106). It is
// comparable. The zero SpellSaveDC is DC 0, which no valid input gives.
type SpellSaveDC struct{ dc d20.DifficultyClass }

// DifficultyClass returns the DC as the d20 type.
func (s SpellSaveDC) DifficultyClass() d20.DifficultyClass { return s.dc }

// Value returns the DC as an integer.
func (s SpellSaveDC) Value() int { return int(s.dc) }

// SpellAttackBonus is a spell attack modifier: the spellcasting ability modifier
// plus the Proficiency Bonus (SRD 5.2.1, "Spells", "Attack Rolls", p. 106). It
// is comparable. The zero SpellAttackBonus is +0.
type SpellAttackBonus struct{ n int }

// Value returns the bonus as a signed integer.
func (b SpellAttackBonus) Value() int { return b.n }

// FeatureSpellAbility is the ability a feat, species or other non-class source
// uses as its spellcasting ability (SRD 5.2.1, "Magic Initiate", p. 87). It is
// comparable. The zero FeatureSpellAbility is invalid.
type FeatureSpellAbility struct{ a ability.Ability }

// NewFeatureSpellAbility returns the value for ability a. It returns an error
// matching srderr.ErrInvalidInput for an invalid ability. It never panics.
func NewFeatureSpellAbility(a ability.Ability) (FeatureSpellAbility, error) {
	if !a.Valid() {
		return FeatureSpellAbility{}, invalidInput("feature spell ability: invalid ability")
	}
	return FeatureSpellAbility{a: a}, nil
}

// Ability returns the ability.
func (f FeatureSpellAbility) Ability() ability.Ability { return f.a }

// SpellcastingStats are the spell save DC and spell attack bonus with the terms
// that make them: the ability, its modifier and the Proficiency Bonus. It is
// comparable. The zero SpellcastingStats is not the result of any valid input.
type SpellcastingStats struct {
	ability ability.Ability
	mod     ability.Modifier
	prof    ProficiencyBonus
}

// Stats returns the stats of class c given the character's modifier in the
// class's spellcasting ability (Ability) and Proficiency Bonus. Each class
// takes its own modifier, so a multiclass character gets stats per class (SRD
// 5.2.1, "Multiclassing", "Spellcasting", p. 25). It returns the zero stats for
// an invalid c.
func Stats(c SpellcastingClass, mod ability.Modifier, pb ProficiencyBonus) SpellcastingStats {
	if !c.Valid() {
		return SpellcastingStats{}
	}
	return SpellcastingStats{ability: c.Ability(), mod: mod, prof: pb}
}

// StatsForAbility returns the stats of a spell that belongs to no class (a feat's
// or a species' spell) cast with the given ability.
func StatsForAbility(a FeatureSpellAbility, mod ability.Modifier, pb ProficiencyBonus) SpellcastingStats {
	return SpellcastingStats{ability: a.a, mod: mod, prof: pb}
}

// Ability returns the spellcasting ability used.
func (s SpellcastingStats) Ability() ability.Ability { return s.ability }

// AbilityModifier returns the ability modifier used.
func (s SpellcastingStats) AbilityModifier() ability.Modifier { return s.mod }

// ProficiencyBonus returns the Proficiency Bonus used.
func (s SpellcastingStats) ProficiencyBonus() ProficiencyBonus { return s.prof }

// SaveDC returns 8 plus the ability modifier plus the Proficiency Bonus.
func (s SpellcastingStats) SaveDC() SpellSaveDC {
	return SpellSaveDC{dc: d20.DifficultyClass(8 + s.mod.Value() + s.prof.Value())}
}

// AttackBonus returns the ability modifier plus the Proficiency Bonus.
func (s SpellcastingStats) AttackBonus() SpellAttackBonus {
	return SpellAttackBonus{n: s.mod.Value() + s.prof.Value()}
}

// AttackRollModifiers returns the modifiers of a spell attack roll: the ability
// modifier and the Proficiency Bonus as kernel modifiers, with no others. The
// sum of its parts equals AttackBonus.
func (s SpellcastingStats) AttackRollModifiers() d20.RollModifiers {
	return d20.NewRollModifiers(modifier.New(s.mod.Value())).WithProficiencyBonus(modifier.New(s.prof.Value()))
}
