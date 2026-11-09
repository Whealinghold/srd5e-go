package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// The Evoker rejection types follow the Wizard ones (wizard_rejection.go): each is a value
// with a Kind (the cause, in the order of the constants, which is the order several
// causes at once are reported in) and the data of that cause. String of a Kind is the
// snake_case short name that is the srd5e.character.rejection attribute value. In every
// type the first two causes are NotEvoker and FeatureNotAvailable.

// EvokerSpellbookRejectionKind is the cause of an EvokerSpellbookRejection.
type EvokerSpellbookRejectionKind int

// The causes, in reporting order.
const (
	EvokerSpellbookNotEvoker EvokerSpellbookRejectionKind = iota + 1
	EvokerSpellbookFeatureNotAvailable
	EvokerSpellbookNoNewSpellLevel
	EvokerSpellbookWrongSpellCount
	EvokerSpellbookDuplicateSpell
	EvokerSpellbookIsACantrip
	EvokerSpellbookNotOnWizardList
	EvokerSpellbookNotAnEvocationSpell
	EvokerSpellbookAboveHighestSpellLevel
	EvokerSpellbookAlreadyInSpellbook
)

// String returns the snake_case name of the cause.
func (k EvokerSpellbookRejectionKind) String() string {
	return kindName([]string{"not_evoker", "feature_not_available", "no_new_spell_level", "wrong_spell_count", "duplicate_spell",
		"is_a_cantrip", "not_on_wizard_list", "not_an_evocation_spell", "above_highest_spell_level", "already_in_spellbook"}, int(k))
}

// EvokerSpellbookRejection says why an Evoker spellbook addition was refused. Allowed is
// the number that had to be named (WrongSpellCount); Highest the highest Spell Level
// (AboveHighestSpellLevel); Spell the offender where there is one.
type EvokerSpellbookRejection struct {
	Kind    EvokerSpellbookRejectionKind
	Allowed int
	Highest int
	Spell   content.SpellID
}

// PotentCantripRejectionKind is the cause of a PotentCantripRejection.
type PotentCantripRejectionKind int

// The causes, in reporting order.
const (
	PotentCantripNotEvoker PotentCantripRejectionKind = iota + 1
	PotentCantripFeatureNotAvailable
	PotentCantripUnsupportedTestKind
	PotentCantripNotACantrip
	PotentCantripSpellDealsNoDamage
)

// String returns the snake_case name of the cause.
func (k PotentCantripRejectionKind) String() string {
	return kindName([]string{"not_evoker", "feature_not_available", "unsupported_test_kind", "not_a_cantrip", "spell_deals_no_damage"}, int(k))
}

// PotentCantripRejection says why Potent Cantrip was refused; TestKind is the kind of
// D20 Test of UnsupportedTestKind.
type PotentCantripRejection struct {
	Kind     PotentCantripRejectionKind
	TestKind d20.Kind
}

// SculptRejectionKind is the cause of a SculptRejection.
type SculptRejectionKind int

// The causes, in reporting order.
const (
	SculptNotEvoker SculptRejectionKind = iota + 1
	SculptFeatureNotAvailable
	SculptNotAnEvocationSpell
	SculptCaster
	SculptCreatureNotEligible
	SculptTooManyCreatures
)

// String returns the snake_case name of the cause.
func (k SculptRejectionKind) String() string {
	return kindName([]string{"not_evoker", "feature_not_available", "not_an_evocation_spell", "caster", "creature_not_eligible", "too_many_creatures"}, int(k))
}

// SculptRejection says why Sculpt Spells was refused. Creature is the offender of Caster
// and CreatureNotEligible; Allowed the most creatures that could be chosen (TooManyCreatures).
type SculptRejection struct {
	Kind     SculptRejectionKind
	Creature Identifier
	Allowed  int
}

// EmpoweredRejectionKind is the cause of an EmpoweredRejection.
type EmpoweredRejectionKind int

// The causes, in reporting order.
const (
	EmpoweredNotEvoker EmpoweredRejectionKind = iota + 1
	EmpoweredFeatureNotAvailable
	EmpoweredNotAWizardSpell
	EmpoweredNotAnEvocationSpell
	EmpoweredAlreadyEmpowered
)

// String returns the snake_case name of the cause.
func (k EmpoweredRejectionKind) String() string {
	return kindName([]string{"not_evoker", "feature_not_available", "not_a_wizard_spell", "not_an_evocation_spell", "already_empowered"}, int(k))
}

// EmpoweredRejection says why Empowered Evocation was refused.
type EmpoweredRejection struct{ Kind EmpoweredRejectionKind }

// OverchannelRejectionKind is the cause of an OverchannelRejection.
type OverchannelRejectionKind int

// The causes, in reporting order.
const (
	OverchannelNotEvoker OverchannelRejectionKind = iota + 1
	OverchannelFeatureNotAvailable
	OverchannelNotAWizardSpell
	OverchannelNoSpellSlotUsed
	OverchannelSlotLevelOutOfRange
	OverchannelSpellDealsNoDamage
)

// String returns the snake_case name of the cause.
func (k OverchannelRejectionKind) String() string {
	return kindName([]string{"not_evoker", "feature_not_available", "not_a_wizard_spell", "no_spell_slot_used", "slot_level_out_of_range", "spell_deals_no_damage"}, int(k))
}

// OverchannelRejection says why Overchannel was refused; Level is the slot level of
// SlotLevelOutOfRange.
type OverchannelRejection struct {
	Kind  OverchannelRejectionKind
	Level int
}
