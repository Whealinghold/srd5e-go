package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/modifier"
)

// speedPenaltyFeet is the Speed reduction of armor worn without its Strength
// requirement (SRD 5.2.1, "Armor", Strength, p. 92).
const speedPenaltyFeet = 10

// ArmorRules is the stateless service that evaluates the Armor table (SRD
// 5.2.1, "Armor", p. 92). It has no state; the zero value is ready to use.
// It evaluates a [content.BaseRule] only, so a Shield's bonus rule cannot be
// passed to it.
type ArmorRules struct{}

// ArmorBase is the base Armor Class of a body armor for one Dexterity
// modifier, with its parts: the listed base and the signed Dexterity
// contribution. Total is their sum. It is an immutable value, comparable with
// ==. The zero value is a base of 0.
type ArmorBase struct {
	listedBase int
	dexterity  int
}

// ListedBase returns the base printed on the Armor table.
func (b ArmorBase) ListedBase() int { return b.listedBase }

// DexterityContribution returns the signed part the Dexterity modifier adds;
// 0 when the armor ignores Dexterity.
func (b ArmorBase) DexterityContribution() int { return b.dexterity }

// Total returns the base Armor Class: ListedBase plus DexterityContribution.
func (b ArmorBase) Total() int { return b.listedBase + b.dexterity }

// BaseArmorClass returns the base Armor Class of rule for a Dexterity
// modifier (SRD 5.2.1, "Armor", Armor table, p. 92): the whole modifier for
// full treatment, the modifier limited to the maximum from above for a capped
// one (a negative modifier is never raised), nothing for none. A rule with no
// treatment (the zero rule) adds nothing.
func (ArmorRules) BaseArmorClass(rule content.BaseRule, dexterity ability.Modifier) ArmorBase {
	m := dexterity.Value()
	contribution := 0
	switch d := rule.Dexterity().(type) {
	case content.FullDexterity:
		contribution = m
	case content.CappedDexterity:
		contribution = min(m, d.Maximum())
	case content.NoDexterity:
	}
	return ArmorBase{listedBase: rule.Base(), dexterity: contribution}
}

// SpeedPenaltyFeet returns 10 when armor has a Strength requirement and
// strength is lower than it, otherwise 0 (SRD 5.2.1, "Armor", Strength, p.
// 92). Applying the penalty to a Speed belongs to the movement rules.
func (ArmorRules) SpeedPenaltyFeet(armor content.Armor, strength ability.Score) int {
	if need, ok := armor.StrengthRequirement(); ok && strength.Value() < need {
		return speedPenaltyFeet
	}
	return 0
}

// StealthCircumstances returns a fresh one-element slice holding a
// [modifier.Disadvantage] when any worn armor gives Disadvantage on Dexterity
// (Stealth) checks, otherwise nil (SRD 5.2.1, "Armor", Stealth, p. 92).
// Disadvantage does not stack, so there is never more than one element. It
// does not depend on Armor Training. The consumer applies it to Dexterity
// (Stealth) checks only.
func (ArmorRules) StealthCircumstances(worn []content.Armor) []modifier.RollCircumstance {
	for _, a := range worn {
		if a.StealthDisadvantage() {
			return []modifier.RollCircumstance{modifier.Disadvantage{}}
		}
	}
	return nil
}
