package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// LastChargeConsequence is what becomes of an item whose last charge went. The
// zero value is invalid.
type LastChargeConsequence int

// The consequences. LastChargeDestroyed is "the item is gone", whatever the
// printed wording.
const (
	LastChargeNone LastChargeConsequence = iota + 1
	LastChargeDestroyed
	LastChargeBecomesNonmagical
	LastChargeRetainsBonusLosesOtherProperties
)

// Valid reports whether c is one of the four consequences.
func (c LastChargeConsequence) Valid() bool {
	return c >= LastChargeNone && c <= LastChargeRetainsBonusLosesOtherProperties
}

// String returns "none", "destroyed", "becomes_nonmagical", "retains_bonus" or "unknown".
func (c LastChargeConsequence) String() string {
	names := []string{"none", "destroyed", "becomes_nonmagical", "retains_bonus"}
	if !c.Valid() {
		return "unknown"
	}
	return names[c-1]
}

// The faces of the last-charge d20 (SRD 5.2.1, "Wand of Magic Missiles", p. 251).
const (
	lastChargeOneFace    = 1
	lastChargeTwentyFace = 20
)

// LastChargeRegain is whether a natural 20 on the last-charge roll gives charges
// back, and the dice for them. The zero value means no regain.
type LastChargeRegain struct{ roll dice.Expression }

// NoRegain: a 20 changes nothing.
func NoRegain() LastChargeRegain { return LastChargeRegain{} }

// RegainsDice: a 20 regains the charges rolled on e (for example 1d8 + 2).
func RegainsDice(e dice.Expression) LastChargeRegain { return LastChargeRegain{roll: e} }

// Dice returns the expression and true, or the zero Expression and false for NoRegain.
func (r LastChargeRegain) Dice() (dice.Expression, bool) { return r.roll, !r.roll.IsZero() }

// LastChargeRule is the closed set of what happens when an item's last charge is
// expended. Variants: NoLastChargeEffect, LastChargeAutomatic, LastChargeRolled.
type LastChargeRule interface{ isLastChargeRule() }

// NoLastChargeEffect: nothing happens.
type NoLastChargeEffect struct{}

// LastChargeAutomatic applies its consequence with no roll.
type LastChargeAutomatic struct{ consequence LastChargeConsequence }

// NewLastChargeAutomatic returns the rule. It returns an error matching
// srderr.ErrInvalidInput for an invalid consequence.
func NewLastChargeAutomatic(c LastChargeConsequence) (LastChargeAutomatic, error) {
	if !c.Valid() {
		return LastChargeAutomatic{}, invalidInput("last charge automatic: invalid consequence")
	}
	return LastChargeAutomatic{consequence: c}, nil
}

// Consequence returns what becomes of the item.
func (r LastChargeAutomatic) Consequence() LastChargeConsequence { return r.consequence }

// LastChargeRolled rolls 1d20: a 1 gives onOne and a 20 gives onTwenty.
type LastChargeRolled struct {
	onOne    LastChargeConsequence
	onTwenty LastChargeRegain
}

// NewLastChargeRolled returns the rule. LastChargeNone and NoRegain mean no
// effect. It returns an error matching srderr.ErrInvalidInput for an invalid
// consequence.
func NewLastChargeRolled(onOne LastChargeConsequence, onTwenty LastChargeRegain) (LastChargeRolled, error) {
	if !onOne.Valid() {
		return LastChargeRolled{}, invalidInput("last charge rolled: invalid consequence")
	}
	return LastChargeRolled{onOne: onOne, onTwenty: onTwenty}, nil
}

// OnOne returns the consequence of a natural 1.
func (r LastChargeRolled) OnOne() LastChargeConsequence { return r.onOne }

// OnTwenty returns what a natural 20 regains.
func (r LastChargeRolled) OnTwenty() LastChargeRegain { return r.onTwenty }

// ForFace returns the consequence of the d20 face: onOne for 1, LastChargeNone
// otherwise; and the regain, which is NoRegain unless the face is 20.
func (r LastChargeRolled) ForFace(face int) (LastChargeConsequence, LastChargeRegain) {
	switch face {
	case lastChargeOneFace:
		return r.onOne, NoRegain()
	case lastChargeTwentyFace:
		return LastChargeNone, r.onTwenty
	}
	return LastChargeNone, NoRegain()
}

func (NoLastChargeEffect) isLastChargeRule()  {}
func (LastChargeAutomatic) isLastChargeRule() {}
func (LastChargeRolled) isLastChargeRule()    {}

// withRegained returns c with n charges regained, capped at the maximum.
func (c ItemCharges) withRegained(n int) ItemCharges {
	return ItemCharges{current: min(c.current+max(n, 0), c.maximum), maximum: c.maximum}
}

// LastChargeRules holds table "Last charge rules" (SRD 5.2.1, pp. 223-251). The
// zero value is ready to use.
type LastChargeRules struct{}

// For returns the rule of id: the wands, rods and staffs print "destroyed" or
// "nonmagical" on a 1; Ring of Three Wishes, Gem of Brightness and Scarab of
// Protection apply theirs automatically. An id not in the table has
// NoLastChargeEffect.
func (LastChargeRules) For(id content.MagicItemID) LastChargeRule {
	rolled := func(c LastChargeConsequence, r LastChargeRegain) LastChargeRule {
		return LastChargeRolled{onOne: c, onTwenty: r}
	}
	switch id {
	case "wand_of_binding", "wand_of_enemy_detection", "wand_of_fear", "wand_of_fireballs", "wand_of_lightning_bolts", "wand_of_magic_missiles",
		"wand_of_paralysis", "wand_of_polymorph", "wand_of_web", "wand_of_wonder", "rod_of_resurrection", "staff_of_charming", "staff_of_frost",
		"staff_of_fire", "staff_of_swarming_insects", "staff_of_healing":
		return rolled(LastChargeDestroyed, NoRegain())
	case "staff_of_striking", "staff_of_the_woodlands":
		return rolled(LastChargeBecomesNonmagical, NoRegain())
	case "staff_of_power":
		return rolled(LastChargeRetainsBonusLosesOtherProperties, RegainsDice(dice.MustExpression(1, dice.D8(), 2)))
	case "staff_of_the_magi":
		return rolled(LastChargeNone, RegainsDice(dice.MustExpression(1, dice.D12(), 1)))
	case "ring_of_three_wishes", "gem_of_brightness":
		return LastChargeAutomatic{consequence: LastChargeBecomesNonmagical}
	case "scarab_of_protection":
		return LastChargeAutomatic{consequence: LastChargeDestroyed}
	}
	return NoLastChargeEffect{}
}
