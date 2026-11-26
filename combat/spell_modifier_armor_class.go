package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/character"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// SpellArmorClassInputs is what the active spells on one creature give its Armor Class (spell-granted-modifiers.md
// section 4.4): flat bonuses in ascending CastOrder, an optional base other than 10 (Mage Armor's 13) and an optional
// floor (Barkskin's 17). The host passes them to character.CalculateDerivedStats; nothing here computes an Armor
// Class. The zero value changes nothing. Not comparable with ==; compare with Equal.
type SpellArmorClassInputs struct {
	bonuses []character.FlatBonus
	base    int
	floor   int
}

// FlatBonuses returns the bonuses, as a new slice.
func (a SpellArmorClassInputs) FlatBonuses() []character.FlatBonus { return slices.Clone(a.bonuses) }

// AlternativeBase returns the base value of the Armor Class calculation a spell gives, and whether one applies.
func (a SpellArmorClassInputs) AlternativeBase() (int, bool) { return a.base, a.base != 0 }

// AlternativeBasis returns the base as a basis for character.CalculateDerivedStats: the base value plus the
// Dexterity modifier, a Shield allowed, under the given feature id. The bool is false when no base applies. It
// returns an error matching srderr.ErrInvalidInput for an id that is not valid.
func (a SpellArmorClassInputs) AlternativeBasis(id content.FeatureID) (character.AbilityBasedArmorClassBasis, bool, error) {
	if a.base == 0 {
		return character.AbilityBasedArmorClassBasis{}, false, nil
	}
	b, err := character.NewAbilityBasedArmorClassBasis(id, []ability.Ability{ability.Dexterity}, true)
	if err == nil {
		b, err = b.WithBaseValue(a.base)
	}
	return b, err == nil, err
}

// Floor returns the least Armor Class a spell gives, and whether one applies.
func (a SpellArmorClassInputs) Floor() (int, bool) { return a.floor, a.floor != 0 }

// ApplyFloor returns the larger of ac and the floor, or ac when there is no floor: a minimum on the finished Armor
// Class, applied after the base, the Shield and every flat bonus (FR-042, FR-043).
func (a SpellArmorClassInputs) ApplyFloor(ac int) int {
	if f, ok := a.Floor(); ok {
		return max(ac, f)
	}
	return ac
}

// Equal reports whether a and o hold the same inputs in the same order.
func (a SpellArmorClassInputs) Equal(o SpellArmorClassInputs) bool {
	return a.base == o.base && a.floor == o.floor && slices.Equal(a.bonuses, o.bonuses)
}

// QueryArmorClass returns what the active spells in sources on creature give its Armor Class: Shield +5, Shield of
// Faith +2, Haste +2, Slow -2, Mage Armor's base 13 (only when wearsArmor is false) and Barkskin's floor 17. The
// bonuses are not clamped (FR-045). A source on another creature, and one for a spell that is not modelled, is
// skipped. The input is not altered.
//
// It returns an error matching ErrDuplicateSource (and srderr.ErrInvalidInput) for two sources with one spell and
// target, and one matching srderr.ErrInvalidInput for a zero source or a zero creature. It never panics.
func QueryArmorClass(sources []SpellModifierSource, creature CombatantID, wearsArmor bool) (SpellArmorClassInputs, error) {
	if err := checkSources(sources); err != nil {
		return SpellArmorClassInputs{}, err
	}
	if creature.IsZero() {
		return SpellArmorClassInputs{}, invalidInput("armor class query without a creature")
	}
	var out SpellArmorClassInputs
	for _, s := range ordered(sources, func(s SpellModifierSource) bool { return s.TargetIs(creature) }) {
		switch s.spell {
		case spellShield:
			out.addBonus(s.spell, 5)
		case spellShieldOfFaith, spellHaste:
			out.addBonus(s.spell, 2)
		case spellSlow:
			out.addBonus(s.spell, -2)
		case spellMageArmor:
			if !wearsArmor {
				out.base = 13
			}
		case spellBarkskin:
			out.floor = 17
		}
	}
	return out, nil
}

// addBonus adds a flat bonus of a spell. The source is a valid spell, so NewFlatBonus cannot fail.
func (a *SpellArmorClassInputs) addBonus(spell content.SpellID, v int) {
	b, _ := character.NewFlatBonus(modifier.New(v), character.SpellFlatBonusSource{Spell: spell})
	a.bonuses = append(a.bonuses, b)
}

// ReevaluateAttack returns the attack roll r judged against a new Armor Class, with the same roll, modifiers and
// critical range and no die rolled: the Shield spell's +5 applies "including against the triggering attack"
// (spell-granted-modifiers.md FR-044). A natural 20 stays a hit and a Critical Hit; a total that now misses is a
// miss. It returns the error of d20.ResolveAttack unchanged, for a zero result. It never panics.
func ReevaluateAttack(r d20.AttackRollResult, ac d20.ArmorClass) (d20.AttackRollResult, error) {
	return d20.ResolveAttack(r.Roll(), r.Modifiers(), ac, r.CriticalRange())
}
