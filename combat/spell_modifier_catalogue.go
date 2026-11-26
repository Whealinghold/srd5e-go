package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// The fourteen spells of table C (spell-granted-modifiers.md section 4.10), by id.
const (
	spellBane          content.SpellID = "bane"
	spellBarkskin      content.SpellID = "barkskin"
	spellBless         content.SpellID = "bless"
	spellBestowCurse   content.SpellID = "bestow_curse"
	spellGuidance      content.SpellID = "guidance"
	spellHaste         content.SpellID = "haste"
	spellHex           content.SpellID = "hex"
	spellHuntersMark   content.SpellID = "hunters_mark"
	spellLongstrider   content.SpellID = "longstrider"
	spellMageArmor     content.SpellID = "mage_armor"
	spellResistance    content.SpellID = "resistance"
	spellShield        content.SpellID = "shield"
	spellShieldOfFaith content.SpellID = "shield_of_faith"
	spellSlow          content.SpellID = "slow"
)

// SpellModifierEntry is one row of table C: the spell, the page of the SRD it is printed on, whether its modifiers
// belong to the caster instead of the target, and its facts. Obtain it from LookupSpellModifiers. Not comparable
// with ==; immutable.
type SpellModifierEntry struct {
	spell  content.SpellID
	page   int
	caster bool
	facts  []SpellEffectFact
	rider  SpellModifierRider
}

// SpellID returns the spell.
func (e SpellModifierEntry) SpellID() content.SpellID { return e.spell }

// Page returns the page of SRD 5.2.1 the spell is printed on (the first page of a spell on two).
func (e SpellModifierEntry) Page() int { return e.page }

// SubjectIsCaster reports whether the creature the modifiers belong to is the caster (Hunter's Mark) and not the
// target (every other spell).
func (e SpellModifierEntry) SubjectIsCaster() bool { return e.caster }

// Facts returns the non-numeric parts of the spell, as a new slice.
func (e SpellModifierEntry) Facts() []SpellEffectFact { return slices.Clone(e.facts) }

// Rider returns what the spell adds to rolls and damage on top of its numeric modifiers (the zero rider for a spell
// that adds none).
func (e SpellModifierEntry) Rider() SpellModifierRider { return e.rider }

// Coverage returns content.CoveragePartial when the entry has any fact, else content.CoverageComplete.
func (e SpellModifierEntry) Coverage() content.EffectCoverage {
	if len(e.facts) > 0 {
		return content.CoveragePartial
	}
	return content.CoverageComplete
}

// SpellModifierLookup is the closed result of LookupSpellModifiers. Variants: SpellModifiersKnown,
// SpellModifiersNotModelled.
type SpellModifierLookup interface{ isSpellModifierLookup() }

// SpellModifiersKnown is a spell of table C.
type SpellModifiersKnown struct{ entry SpellModifierEntry }

// Entry returns the catalogue entry.
func (k SpellModifiersKnown) Entry() SpellModifierEntry { return k.entry }

// SpellModifiersNotModelled is a spell that contributes nothing here: not an error, since the spell may still
// resolve its cast through ResolveSpellEffects.
type SpellModifiersNotModelled struct{ spell content.SpellID }

// SpellID returns the spell.
func (n SpellModifiersNotModelled) SpellID() content.SpellID { return n.spell }

func (SpellModifiersKnown) isSpellModifierLookup()       {}
func (SpellModifiersNotModelled) isSpellModifierLookup() {}

// LookupSpellModifiers returns the entry of table C for the spell, or SpellModifiersNotModelled for any other id
// (spell-granted-modifiers.md FR-002). It never fails and never panics.
func LookupSpellModifiers(spell content.SpellID) SpellModifierLookup {
	if e, ok := catalogueEntry(spell); ok {
		return SpellModifiersKnown{e}
	}
	return SpellModifiersNotModelled{spell}
}

// isModelled reports whether the spell is one of table C.
func isModelled(spell content.SpellID) bool {
	_, ok := catalogueEntry(spell)
	return ok
}

// catalogueEntry is table C as a switch (a package-level table would break A4).
func catalogueEntry(spell content.SpellID) (SpellModifierEntry, bool) {
	e := SpellModifierEntry{spell: spell}
	switch spell {
	case spellBane:
		e.page, e.rider = 112, SpellModifierRider{rollDie: d4Die(true)}
	case spellBarkskin:
		e.page = 112
	case spellBless:
		e.page, e.rider = 113, SpellModifierRider{rollDie: d4Die(false)}
	case spellBestowCurse:
		e.page, e.facts = 112, []SpellEffectFact{ForcedDodgeUnlessWisdomSave{}}
	case spellGuidance:
		e.page = 138
	case spellHaste:
		e.page, e.facts = 139, []SpellEffectFact{ExtraActionGranted{}, LethargyOnEnd{}}
	case spellHex:
		e.page, e.facts = 140, []SpellEffectFact{MarkMovesOnZeroHitPoints{}}
		e.rider = SpellModifierRider{extraDamage: MustDiceDamage(1, dice.D6(), content.DamageNecrotic, 0, false), checkDisadvantage: true}
	case spellHuntersMark:
		e.page, e.caster, e.facts = 141, true, []SpellEffectFact{MarkMovesOnZeroHitPoints{}}
		e.rider = SpellModifierRider{extraDamage: MustDiceDamage(1, dice.D6(), content.DamageForce, 0, false)}
	case spellLongstrider:
		e.page = 145
	case spellMageArmor:
		e.page, e.facts = 145, []SpellEffectFact{EndsIfArmorDonned{}}
	case spellResistance:
		e.page = 158
	case spellShield:
		e.page, e.facts = 161, []SpellEffectFact{NoDamageFromMagicMissile{}}
	case spellShieldOfFaith:
		e.page = 162
	case spellSlow:
		e.page, e.facts = 163, []SpellEffectFact{ReactionsForbidden{}, ActionOrBonusActionNotBoth{}, OneAttackOnly{}, SpellFailureChancePercent{}}
	default:
		return SpellModifierEntry{}, false
	}
	return e, true
}
