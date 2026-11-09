package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

const (
	blessedStrikesLevel         = 7
	improvedBlessedStrikesLevel = 14
	// potentSpellcastingRangeFeet is the range of the Temporary Hit Points of Improved Blessed Strikes.
	potentSpellcastingRangeFeet = 60
)

// ChooseBlessedStrikesResult is the closed set of outcomes of ChooseBlessedStrikes. Variants:
// ChooseBlessedStrikesChosen, ChooseBlessedStrikesRejected, ChooseBlessedStrikesNotACleric.
type ChooseBlessedStrikesResult interface{ isChooseBlessedStrikesResult() }

// ChooseBlessedStrikesChosen is the outcome of a valid choice. Only ChooseBlessedStrikes builds one.
type ChooseBlessedStrikesChosen struct {
	character Character
	choices   ClericChoices
	events    []Event
}

// Character returns the new Character, holding the choices under ClericChoicesKey.
func (r ChooseBlessedStrikesChosen) Character() Character { return r.character }

// Choices returns the new choices.
func (r ChooseBlessedStrikesChosen) Choices() ClericChoices { return r.choices }

// Events returns a new slice holding the ClericOptionChosen event.
func (r ChooseBlessedStrikesChosen) Events() []Event { return slices.Clone(r.events) }

// ChooseBlessedStrikesRejected is the outcome of a refused choice: nothing changed and there are no
// events. Construct it with a keyed literal.
type ChooseBlessedStrikesRejected struct{ Reason ClericReason }

// ChooseBlessedStrikesNotACleric is the outcome for a character with no Cleric levels.
type ChooseBlessedStrikesNotACleric struct{}

func (ChooseBlessedStrikesChosen) isChooseBlessedStrikesResult()     {}
func (ChooseBlessedStrikesRejected) isChooseBlessedStrikesResult()   {}
func (ChooseBlessedStrikesNotACleric) isChooseBlessedStrikesResult() {}

// ChooseBlessedStrikes chooses the Blessed Strikes option of a Cleric of level 7 or higher (SRD
// 5.2.1, "Level 7: Blessed Strikes", p. 38). names are the requested options ("divine_strike",
// "potent_spellcasting"). The first reason that holds is reported: below_level_7, already_chosen,
// option_missing, both_options, unknown_option.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character or another type of state
// under ClericChoicesKey. A refused choice is a ChooseBlessedStrikesRejected result.
func ChooseBlessedStrikes(c Character, names []string) (ChooseBlessedStrikesResult, error) {
	if c.IsZero() {
		return nil, invalidInput("ChooseBlessedStrikes: zero Character")
	}
	level, ok := clericLevelOf(c)
	if !ok {
		return ChooseBlessedStrikesNotACleric{}, nil
	}
	held, err := readClericChoices(c)
	if err != nil {
		return nil, err
	}
	var option BlessedStrikesOption
	var reason ClericReason
	switch {
	case level.n < blessedStrikesLevel:
		reason = ClericBelowLevel7
	case held.strikes != 0:
		reason = ClericAlreadyChosen
	case len(names) == 0:
		reason = ClericOptionMissing
	case len(names) > 1:
		reason = ClericBothOptions
	case names[0] == BlessedStrikesDivineStrike.String():
		option = BlessedStrikesDivineStrike
	case names[0] == BlessedStrikesPotentSpellcasting.String():
		option = BlessedStrikesPotentSpellcasting
	default:
		reason = ClericUnknownOption
	}
	if reason != 0 {
		return ChooseBlessedStrikesRejected{Reason: reason}, nil
	}
	held.strikes = option
	next, err := WithFeatureState(c, ClericChoicesKey(), held)
	if err != nil { // unreachable: the key is valid and the state is read from the same key
		return nil, err
	}
	return ChooseBlessedStrikesChosen{character: next, choices: held,
		events: []Event{ClericOptionChosen{id: c.ID(), feature: "blessed_strikes", name: option.String()}}}, nil
}

// BlessedStrikesFacts are the host's facts about the hit or the cast Blessed Strikes is resolved for, a
// closed set. Variants: DivineStrikeFacts, PotentSpellcastingFacts.
type BlessedStrikesFacts interface{ isBlessedStrikesFacts() }

// DivineStrikeFacts are the facts of a hit: whether the attack roll used a weapon, whether Divine Strike
// was already used this turn, and the damage type the Cleric chose (Necrotic or Radiant). Construct it
// with a keyed literal.
type DivineStrikeFacts struct {
	WeaponAttack bool
	TurnSpent    bool
	DamageType   content.DamageType
}

// PotentSpellcastingFacts are the facts of a cast: the spell, whether it dealt damage to a creature, the
// Wisdom modifier, and how far from the Cleric the Temporary Hit Points recipient is (0 for the Cleric
// itself). Construct it with a keyed literal.
type PotentSpellcastingFacts struct {
	Spell                 content.Spell
	DamageDealt           bool
	Wisdom                ability.Modifier
	RecipientDistanceFeet int
}

func (DivineStrikeFacts) isBlessedStrikesFacts()       {}
func (PotentSpellcastingFacts) isBlessedStrikesFacts() {}

// ResolveBlessedStrikesResult is the closed set of outcomes of ResolveBlessedStrikes. Variants:
// ResolveBlessedStrikesResolved, ResolveBlessedStrikesNotApplicable, ResolveBlessedStrikesNotACleric.
type ResolveBlessedStrikesResult interface{ isResolveBlessedStrikesResult() }

// ResolveBlessedStrikesResolved is the outcome of a Blessed Strikes effect that applies. Only
// ResolveBlessedStrikes builds one.
type ResolveBlessedStrikesResolved struct {
	extra          dice.Expression
	extraType      content.DamageType
	bonus          int
	hasBonus       bool
	temp           int
	tempReason     ClericReason
	hasTempOffered bool
}

// ExtraDamage returns the Divine Strike extra damage as a dice expression (1d8, 2d8 from level 14) and the
// damage type the Cleric chose; false for Potent Spellcasting. The host rolls it with the weapon's damage.
func (r ResolveBlessedStrikesResolved) ExtraDamage() (dice.Expression, content.DamageType, bool) {
	return r.extra, r.extraType, !r.extra.IsZero()
}

// BonusDamage returns the Wisdom modifier added to the cantrip's damage (the literal modifier; the host's
// damage rules floor the total at 0); false for Divine Strike.
func (r ResolveBlessedStrikesResolved) BonusDamage() (int, bool) { return r.bonus, r.hasBonus }

// TemporaryHitPoints returns the Temporary Hit Points the Cleric may grant (twice the Wisdom modifier, 0 for
// a modifier of 0 or less) and true when they are offered.
func (r ResolveBlessedStrikesResolved) TemporaryHitPoints() (int, bool) {
	return r.temp, r.hasTempOffered
}

// TemporaryHitPointsReason returns why the Temporary Hit Points are not offered (below_level_14 or
// recipient_out_of_range) for a Potent Spellcasting that is resolved; 0 when they are offered or for Divine Strike.
func (r ResolveBlessedStrikesResolved) TemporaryHitPointsReason() ClericReason { return r.tempReason }

// ResolveBlessedStrikesNotApplicable is the outcome of a Blessed Strikes effect that does not apply. It is
// never an error. Construct it with a keyed literal.
type ResolveBlessedStrikesNotApplicable struct{ Reason ClericReason }

// ResolveBlessedStrikesNotACleric is the outcome for a character with no Cleric levels.
type ResolveBlessedStrikesNotACleric struct{}

func (ResolveBlessedStrikesResolved) isResolveBlessedStrikesResult()      {}
func (ResolveBlessedStrikesNotApplicable) isResolveBlessedStrikesResult() {}
func (ResolveBlessedStrikesNotACleric) isResolveBlessedStrikesResult()    {}

// ResolveBlessedStrikes decides what Blessed Strikes adds to a hit (DivineStrikeFacts) or a cast
// (PotentSpellcastingFacts) of the character (SRD 5.2.1, "Level 7: Blessed Strikes", "Level 14: Improved
// Blessed Strikes", p. 38). The reasons it reports, in order, are below_level_7, no_option_chosen,
// other_option_chosen, then for Divine Strike turn_spent, not_weapon_attack and damage_type_not_allowed, and
// for Potent Spellcasting not_cleric_cantrip and no_damage_dealt; a cantrip that dealt damage always gets the
// bonus, and its Temporary Hit Points are offered from level 14 within 60 feet of the Cleric.
//
// It returns an error matching srderr.ErrInvalidInput for a zero Character, nil facts, another type of state
// under ClericChoicesKey or a negative recipient distance. It never panics.
func ResolveBlessedStrikes(c Character, facts BlessedStrikesFacts) (ResolveBlessedStrikesResult, error) {
	if c.IsZero() || facts == nil {
		return nil, invalidInput("ResolveBlessedStrikes: zero Character or nil facts")
	}
	level, ok := clericLevelOf(c)
	if !ok {
		return ResolveBlessedStrikesNotACleric{}, nil
	}
	held, err := readClericChoices(c)
	if err != nil {
		return nil, err
	}
	no := func(r ClericReason) (ResolveBlessedStrikesResult, error) {
		return ResolveBlessedStrikesNotApplicable{Reason: r}, nil
	}
	switch {
	case level.n < blessedStrikesLevel:
		return no(ClericBelowLevel7)
	case held.strikes == 0:
		return no(ClericNoOptionChosen)
	}
	switch f := facts.(type) {
	case DivineStrikeFacts:
		switch {
		case held.strikes != BlessedStrikesDivineStrike:
			return no(ClericOtherOptionChosen)
		case f.TurnSpent:
			return no(ClericTurnSpent)
		case !f.WeaponAttack:
			return no(ClericNotWeaponAttack)
		case f.DamageType != content.DamageNecrotic && f.DamageType != content.DamageRadiant:
			return no(ClericDamageTypeNotAllowed)
		}
		count := 1
		if level.n >= improvedBlessedStrikesLevel {
			count = 2
		}
		return ResolveBlessedStrikesResolved{extra: dice.MustExpression(count, dice.D8(), 0), extraType: f.DamageType}, nil
	case PotentSpellcastingFacts:
		if f.RecipientDistanceFeet < 0 {
			return nil, invalidInput("Potent Spellcasting recipient at a negative distance")
		}
		switch {
		case held.strikes != BlessedStrikesPotentSpellcasting:
			return no(ClericOtherOptionChosen)
		case f.Spell.Level() != 0 || !slices.Contains(f.Spell.Classes(), ClericClassID):
			return no(ClericNotClericCantrip)
		case !f.DamageDealt:
			return no(ClericNoDamageDealt)
		}
		res := ResolveBlessedStrikesResolved{bonus: f.Wisdom.Value(), hasBonus: true}
		switch {
		case level.n < improvedBlessedStrikesLevel:
			res.tempReason = ClericBelowLevel14
		case f.RecipientDistanceFeet > potentSpellcastingRangeFeet:
			res.tempReason = ClericRecipientOutOfRange
		default:
			res.temp, res.hasTempOffered = max(2*f.Wisdom.Value(), 0), true
		}
		return res, nil
	}
	return nil, invalidInput("ResolveBlessedStrikes: facts of an unknown kind")
}
