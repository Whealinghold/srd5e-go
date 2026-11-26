package combat

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// spellcastingPartName is the name of the stat-block part that holds a monster's
// Spellcasting trait (SRD 5.2.1, "Spellcasting", p. 257).
const spellcastingPartName = "Spellcasting"

// MonsterSpellcaster is what the monster spellcasting rules read of a monster: its
// combatant, its monster id, the structured Spellcasting trait (none for a monster
// whose parts cast on their own) and the Modifier of each of its six scores (SRD
// 5.2.1, "Spellcasting", p. 257). It holds the printed numbers by copy, so no
// operation takes the Monster again. It holds a slice through the trait, so compare
// it with Equal. The zero value is invalid (see IsZero) and every operation given it
// returns an error. Safe for concurrent use.
type MonsterSpellcaster struct {
	combatant CombatantID
	monsterID content.MonsterID
	trait     content.MonsterSpellcasting
	modifiers [6]ability.Modifier
}

// NewMonsterSpellcaster returns the caster of monster m for combatant: the part
// named Spellcasting, with its trait and the Modifier of each of the monster's
// scores.
//
// It returns an error matching srderr.ErrInvalidInput for a zero m, a zero
// combatant, a monster with no part named Spellcasting that has a structured
// trait, a monster with two such parts, and a trait that is not the first Spellcasting
// of the stat block (its spell references would not match the LimitedUseState). The
// trait numbers are not checked again: content validated the same ranges when it built
// the trait (plan P13), and the per-day counts are checked by StartLimitedUse. It returns a LimitExceededError (plan P10) for a trait of more than 64 groups or 64 entries. It never panics.
func NewMonsterSpellcaster(m content.Monster, combatant CombatantID) (MonsterSpellcaster, error) {
	c, err := newCaster(m, combatant)
	if err != nil {
		return MonsterSpellcaster{}, err
	}
	var found []content.MonsterSpellcasting
	for _, list := range [][]content.StatBlockPart{m.Traits(), m.Actions()} {
		for _, p := range list {
			if sc, ok := p.Spellcasting(); ok && p.Name() == spellcastingPartName {
				found = append(found, sc)
			}
		}
	}
	if len(found) != 1 {
		return MonsterSpellcaster{}, invalidInput("a monster needs exactly one part named Spellcasting with a structured trait")
	}
	if first, _ := m.Spellcasting(); !first.Equal(found[0]) {
		return MonsterSpellcaster{}, invalidInput("the Spellcasting part is not the first Spellcasting of the stat block")
	}
	entries := 0
	for _, g := range found[0].Groups() {
		entries += len(g.Spells())
	}
	if err := overCap("spellcasting groups", len(found[0].Groups()), maxMonsterSpellGroups); err != nil {
		return MonsterSpellcaster{}, err
	}
	if err := overCap("spellcasting entries", entries, maxMonsterSpells); err != nil {
		return MonsterSpellcaster{}, err
	}
	c.trait = found[0]
	return c, nil
}

// MonsterSpellcasterForParts returns the caster of monster m for combatant with no
// trait, for a monster whose stat-block parts cast on their own (a PartCast with its
// own PartSpellcasting). A ListCast by it is refused with
// MonsterCastRefusalNotInSpellList.
//
// It returns an error matching srderr.ErrInvalidInput for a zero m or a zero
// combatant. It never panics.
func MonsterSpellcasterForParts(m content.Monster, combatant CombatantID) (MonsterSpellcaster, error) {
	return newCaster(m, combatant)
}

func newCaster(m content.Monster, combatant CombatantID) (MonsterSpellcaster, error) {
	if m.ID() == "" || combatant.IsZero() {
		return MonsterSpellcaster{}, invalidInput("a monster spellcaster needs a monster and a combatant")
	}
	c := MonsterSpellcaster{combatant: combatant, monsterID: m.ID()}
	for i, a := range ability.All() {
		c.modifiers[i] = m.AbilityModifier(a)
	}
	return c, nil
}

func saveDCInRange(dc int) bool {
	return dc >= content.MonsterMinSaveDC && dc <= content.MonsterMaxNumber
}

func bonusInRange(b int) bool { return b >= -content.MonsterMaxNumber && b <= content.MonsterMaxNumber }

// Combatants returns a new slice holding the monster combatant.
func (c MonsterSpellcaster) Combatants() []CombatantID { return []CombatantID{c.combatant} }

// MonsterID returns the monster id.
func (c MonsterSpellcaster) MonsterID() content.MonsterID { return c.monsterID }

// Spellcasting returns the Spellcasting trait, and false for a caster built with
// MonsterSpellcasterForParts.
func (c MonsterSpellcaster) Spellcasting() (content.MonsterSpellcasting, bool) {
	return c.trait, !c.trait.IsZero()
}

// AbilityModifier returns the Modifier of the monster's score in a, or the zero
// Modifier for an invalid ability.
func (c MonsterSpellcaster) AbilityModifier(a ability.Ability) ability.Modifier {
	if !a.Valid() {
		return ability.Modifier{}
	}
	return c.modifiers[a-1]
}

// IsZero reports whether c is the invalid zero value.
func (c MonsterSpellcaster) IsZero() bool { return c.combatant.IsZero() }

// Equal reports whether c and o hold the same values.
func (c MonsterSpellcaster) Equal(o MonsterSpellcaster) bool {
	return c.combatant == o.combatant && c.monsterID == o.monsterID && c.modifiers == o.modifiers && c.trait.Equal(o.trait)
}
