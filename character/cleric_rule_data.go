package character

import (
	"slices"
	"strconv"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// ExtraDivineInterventionGrant is a spell a Cleric of at least MinLevel may choose with
// Divine Intervention beyond the Cleric list: the SRD grants Wish at level 20, after which
// Divine Intervention recharges only after 2d4 Long Rests (SRD 5.2.1, "Level 20: Greater
// Divine Intervention", p. 38). It is immutable and comparable with ==. The zero value is
// invalid; build it with NewExtraDivineInterventionGrant.
type ExtraDivineInterventionGrant struct {
	spell     content.SpellID
	minLevel  int
	diceCount int
	die       dice.Die
}

// NewExtraDivineInterventionGrant returns the grant. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell id, a minimum Cleric level outside 1 to 20,
// a dice count below 1 or an invalid die. It never panics.
func NewExtraDivineInterventionGrant(spell content.SpellID, minLevel, diceCount int, die dice.Die) (ExtraDivineInterventionGrant, error) {
	if !spell.Valid() || minLevel < 1 || minLevel > MaxLevel || diceCount < 1 || die.IsZero() {
		return ExtraDivineInterventionGrant{}, invalidInput("extra Divine Intervention grant: spell " + string(spell) + ", level " + strconv.Itoa(minLevel))
	}
	return ExtraDivineInterventionGrant{spell: spell, minLevel: minLevel, diceCount: diceCount, die: die}, nil
}

// Spell returns the granted spell id.
func (g ExtraDivineInterventionGrant) Spell() content.SpellID { return g.spell }

// MinLevel returns the minimum Cleric level.
func (g ExtraDivineInterventionGrant) MinLevel() int { return g.minLevel }

// RechargeDice returns the dice whose total is the number of Long Rests to recharge.
func (g ExtraDivineInterventionGrant) RechargeDice() dice.Expression {
	return dice.MustExpression(g.diceCount, g.die, 0)
}

// ClericRuleData is the Cleric rule data that the content format keeps only as feature
// text: the recommended spells and the extra Divine Intervention grants (class-cleric.md
// FR-133). DefaultClericRuleData holds the SRD 5.2.1 values; a host may pass another
// instance (house rules). It holds slices, so it is not comparable with ==; every getter
// returns a copy. The zero value has no recommendation and no grant.
type ClericRuleData struct {
	cantrips, level1 []content.SpellID
	grants           []ExtraDivineInterventionGrant
}

// DefaultClericRuleData returns the SRD values: Guidance, Sacred Flame and Thaumaturgy as
// recommended cantrips; Bless, Cure Wounds, Guiding Bolt and Shield of Faith as recommended
// level 1 spells (SRD 5.2.1, "Level 1: Spellcasting", pp. 36-37); Wish at level 20, 2d4.
func DefaultClericRuleData() ClericRuleData {
	wish := ExtraDivineInterventionGrant{spell: "wish", minLevel: 20, diceCount: 2, die: dice.D4()}
	return ClericRuleData{
		cantrips: []content.SpellID{"guidance", "sacred_flame", "thaumaturgy"},
		level1:   []content.SpellID{"bless", "cure_wounds", "guiding_bolt", "shield_of_faith"},
		grants:   []ExtraDivineInterventionGrant{wish},
	}
}

// NewClericRuleData returns rule data holding copies of the arguments. It returns an
// error matching srderr.ErrInvalidInput for an invalid spell id or an invalid grant. It
// never panics.
func NewClericRuleData(cantrips, level1 []content.SpellID, grants []ExtraDivineInterventionGrant) (ClericRuleData, error) {
	for _, id := range slices.Concat(cantrips, level1) {
		if !id.Valid() {
			return ClericRuleData{}, invalidInput("Cleric rule data: invalid recommended spell id")
		}
	}
	for _, g := range grants {
		if _, err := NewExtraDivineInterventionGrant(g.spell, g.minLevel, g.diceCount, g.die); err != nil {
			return ClericRuleData{}, err
		}
	}
	return ClericRuleData{cantrips: slices.Clone(cantrips), level1: slices.Clone(level1), grants: slices.Clone(grants)}, nil
}

// RecommendedCantrips returns the recommended cantrip ids as a new slice.
func (d ClericRuleData) RecommendedCantrips() []content.SpellID { return slices.Clone(d.cantrips) }

// RecommendedLevel1Spells returns the recommended level 1 spell ids as a new slice.
func (d ClericRuleData) RecommendedLevel1Spells() []content.SpellID { return slices.Clone(d.level1) }

// ExtraDivineInterventionGrants returns the grants as a new slice.
func (d ClericRuleData) ExtraDivineInterventionGrants() []ExtraDivineInterventionGrant {
	return slices.Clone(d.grants)
}

// grantFor returns the grant for spell that a Cleric of level has unlocked.
func (d ClericRuleData) grantFor(spell content.SpellID, level ClericLevel) (ExtraDivineInterventionGrant, bool) {
	for _, g := range d.grants {
		if g.spell == spell && level.n >= g.minLevel {
			return g, true
		}
	}
	return ExtraDivineInterventionGrant{}, false
}
