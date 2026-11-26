package combat

import "github.com/Whealinghold/srd5e-go/content"

// SpellEffectFact is a part of a modelled spell that is not a number: descriptive only, never resolved here
// (spell-granted-modifiers.md FR-070). The host or the active effects apply it. Variants: ExtraActionGranted,
// LethargyOnEnd, ReactionsForbidden, ActionOrBonusActionNotBoth, OneAttackOnly, SpellFailureChancePercent,
// MarkMovesOnZeroHitPoints, ForcedDodgeUnlessWisdomSave, EndsIfArmorDonned, NoDamageFromMagicMissile.
type SpellEffectFact interface{ isSpellEffectFact() }

// ExtraActionGranted is Haste's extra action (SRD 5.2.1, "Haste", p. 139).
type ExtraActionGranted struct{}

// Allowed returns what the extra action may be: one Attack action with one attack only, Dash, Disengage, Hide or
// Utilize.
func (ExtraActionGranted) Allowed() []StandardAction {
	return []StandardAction{StandardActionAttack, StandardActionDash, StandardActionDisengage, StandardActionHide, StandardActionUtilize}
}

// AttackLimit returns how many attacks the Attack action of the extra action allows: 1.
func (ExtraActionGranted) AttackLimit() int { return 1 }

// LethargyOnEnd is the lethargy at the end of Haste: Incapacitated and a Speed of 0 until the end of the creature's
// next turn (SRD 5.2.1, "Haste", p. 139).
type LethargyOnEnd struct{}

// Condition returns the condition: Incapacitated.
func (LethargyOnEnd) Condition() content.ConditionKind { return content.ConditionIncapacitated }

// SpeedFeet returns the Speed: 0.
func (LethargyOnEnd) SpeedFeet() int { return 0 }

// ReactionsForbidden is Slow's ban on Reactions (SRD 5.2.1, "Slow", p. 163).
type ReactionsForbidden struct{}

// ActionOrBonusActionNotBoth is Slow's limit of an action or a Bonus Action on a turn, not both.
type ActionOrBonusActionNotBoth struct{}

// OneAttackOnly is Slow's limit of one attack with the Attack action.
type OneAttackOnly struct{}

// SpellFailureChancePercent is Slow's chance that a spell with a Somatic component fails.
type SpellFailureChancePercent struct{}

// Percent returns the chance: 25.
func (SpellFailureChancePercent) Percent() int { return 25 }

// MarkMovesOnZeroHitPoints is the move of the mark of Hex and Hunter's Mark to a new creature when the marked one
// drops to 0 Hit Points.
type MarkMovesOnZeroHitPoints struct{}

// ForcedDodgeUnlessWisdomSave is the forced Dodge of Bestow Curse.
type ForcedDodgeUnlessWisdomSave struct{}

// EndsIfArmorDonned is the end of Mage Armor when the creature dons armor.
type EndsIfArmorDonned struct{}

// NoDamageFromMagicMissile is Shield's immunity to Magic Missile damage.
type NoDamageFromMagicMissile struct{}

func (ExtraActionGranted) isSpellEffectFact()          {}
func (LethargyOnEnd) isSpellEffectFact()               {}
func (ReactionsForbidden) isSpellEffectFact()          {}
func (ActionOrBonusActionNotBoth) isSpellEffectFact()  {}
func (OneAttackOnly) isSpellEffectFact()               {}
func (SpellFailureChancePercent) isSpellEffectFact()   {}
func (MarkMovesOnZeroHitPoints) isSpellEffectFact()    {}
func (ForcedDodgeUnlessWisdomSave) isSpellEffectFact() {}
func (EndsIfArmorDonned) isSpellEffectFact()           {}
func (NoDamageFromMagicMissile) isSpellEffectFact()    {}
