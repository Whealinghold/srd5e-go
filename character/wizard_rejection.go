package character

import (
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// The Wizard rejection types. Each is a value with a Kind (the cause, in the order
// of the type's constants, which is the order several causes at once are reported
// in) and the data of that cause. String of a Kind is the snake_case short name that
// is the srd5e.character.rejection attribute value. The zero value of a Kind is not a
// cause. Every type is comparable except where it carries a slice.

func kindName(names []string, k int) string {
	if k < 1 || k > len(names) {
		return "unknown"
	}
	return names[k-1]
}

// SkillsRejectionKind is the cause of a SkillsRejection.
type SkillsRejectionKind int

// The causes, in reporting order.
const (
	SkillsWrongSkillCount SkillsRejectionKind = iota + 1
	SkillsDuplicateSkill
	SkillsSkillNotAllowed
)

func skillsKindNames() []string {
	return []string{"wrong_skill_count", "duplicate_skill", "skill_not_allowed"}
}

// String returns the snake_case name of the cause.
func (k SkillsRejectionKind) String() string { return kindName(skillsKindNames(), int(k)) }

// SkillsRejection says why the Wizard skills were refused. Count is the number
// named for SkillsWrongSkillCount; Skill is the offender of the other two.
type SkillsRejection struct {
	Kind  SkillsRejectionKind
	Count int
	Skill content.Skill
}

// EquipmentRejectionKind is the cause of an EquipmentRejection.
type EquipmentRejectionKind int

// EquipmentUnknownEquipmentOption: the option is neither "A" nor "B".
const EquipmentUnknownEquipmentOption EquipmentRejectionKind = 1

// String returns the snake_case name of the cause.
func (k EquipmentRejectionKind) String() string {
	return kindName([]string{"unknown_equipment_option"}, int(k))
}

// EquipmentRejection says why the Starting Equipment option was refused; Option is the raw value.
type EquipmentRejection struct {
	Kind   EquipmentRejectionKind
	Option string
}

// CantripChoiceRejectionKind is the cause of a CantripChoiceRejection.
type CantripChoiceRejectionKind int

// The causes, in reporting order.
const (
	CantripChoiceWrongCantripCount CantripChoiceRejectionKind = iota + 1
	CantripChoiceDuplicateCantrip
	CantripChoiceNotACantrip
	CantripChoiceNotOnWizardList
)

func cantripKindNames() []string {
	return []string{"wrong_cantrip_count", "duplicate_cantrip", "not_a_cantrip", "not_on_wizard_list"}
}

// String returns the snake_case name of the cause.
func (k CantripChoiceRejectionKind) String() string { return kindName(cantripKindNames(), int(k)) }

// CantripChoiceRejection says why a cantrip choice was refused. Allowed is the
// number the choice had to name (WrongCantripCount); Spell is the offender of the others.
type CantripChoiceRejection struct {
	Kind    CantripChoiceRejectionKind
	Allowed int
	Spell   content.SpellID
}

// SpellbookRejectionKind is the cause of a SpellbookRejection.
type SpellbookRejectionKind int

// The causes, in reporting order.
const (
	SpellbookFeatureNotAvailable SpellbookRejectionKind = iota + 1
	SpellbookWrongSpellCount
	SpellbookDuplicateSpell
	SpellbookIsACantrip
	SpellbookNotOnWizardList
	SpellbookAboveHighestSpellLevel
	SpellbookAlreadyInSpellbook
	SpellbookNotInSourceSpellbook
)

func spellbookKindNames() []string {
	return []string{"feature_not_available", "wrong_spell_count", "duplicate_spell", "is_a_cantrip", "not_on_wizard_list",
		"above_highest_spell_level", "already_in_spellbook", "not_in_source_spellbook"}
}

// String returns the snake_case name of the cause.
func (k SpellbookRejectionKind) String() string { return kindName(spellbookKindNames(), int(k)) }

// SpellbookRejection says why a spellbook operation was refused. Allowed is the
// number of spells that had to be named (WrongSpellCount); Highest is the highest
// Spell Level (AboveHighestSpellLevel); Spell is the offender where there is one.
type SpellbookRejection struct {
	Kind    SpellbookRejectionKind
	Allowed int
	Highest int
	Spell   content.SpellID
}

// ArcaneRecoveryRejectionKind is the cause of an ArcaneRecoveryRejection.
type ArcaneRecoveryRejectionKind int

// The causes, in reporting order.
const (
	ArcaneRecoveryNotAfterShortRest ArcaneRecoveryRejectionKind = iota + 1
	ArcaneRecoveryAlreadyUsed
	ArcaneRecoverySlotLevelTooHigh
	ArcaneRecoverySlotNotExpended
	ArcaneRecoveryExceedsMaximum
)

func arcaneKindNames() []string {
	return []string{"not_after_short_rest", "already_used", "slot_level_too_high", "slot_not_expended", "exceeds_maximum"}
}

// String returns the snake_case name of the cause.
func (k ArcaneRecoveryRejectionKind) String() string { return kindName(arcaneKindNames(), int(k)) }

// ArcaneRecoveryRejection says why Arcane Recovery was refused. Level is the slot
// level of SlotLevelTooHigh and SlotNotExpended; Maximum is the most levels of
// ExceedsMaximum.
type ArcaneRecoveryRejection struct {
	Kind    ArcaneRecoveryRejectionKind
	Level   int
	Maximum int
}

// ScholarRejectionKind is the cause of a ScholarRejection.
type ScholarRejectionKind int

// The causes, in reporting order.
const (
	ScholarFeatureNotAvailable ScholarRejectionKind = iota + 1
	ScholarAlready
	ScholarSkillNotEligible
	ScholarNotProficient
)

func scholarKindNames() []string {
	return []string{"feature_not_available", "already", "skill_not_eligible", "not_proficient"}
}

// String returns the snake_case name of the cause.
func (k ScholarRejectionKind) String() string { return kindName(scholarKindNames(), int(k)) }

// ScholarRejection says why a Scholar choice was refused; Skill is the offender of
// SkillNotEligible and NotProficient.
type ScholarRejection struct {
	Kind  ScholarRejectionKind
	Skill content.Skill
}

// MemorizeRejectionKind is the cause of a MemorizeRejection.
type MemorizeRejectionKind int

// The causes, in reporting order.
const (
	MemorizeFeatureNotAvailable MemorizeRejectionKind = iota + 1
	MemorizeNotAfterShortRest
	MemorizeNotPrepared
	MemorizeAlwaysPreparedNotReplaceable
	MemorizeNewSpellIsCantrip
	MemorizeNewSpellNotInSpellbook
	MemorizeNewSpellAboveHighestLevel
	MemorizeNewSpellAlreadyPrepared
)

func memorizeKindNames() []string {
	return []string{"feature_not_available", "not_after_short_rest", "not_prepared", "always_prepared_not_replaceable",
		"new_spell_is_cantrip", "new_spell_not_in_spellbook", "new_spell_above_highest_level", "new_spell_already_prepared"}
}

// String returns the snake_case name of the cause.
func (k MemorizeRejectionKind) String() string { return kindName(memorizeKindNames(), int(k)) }

// MemorizeRejection says why Memorize Spell was refused; Spell is the offender and
// Highest the highest Spell Level (NewSpellAboveHighestLevel).
type MemorizeRejection struct {
	Kind    MemorizeRejectionKind
	Spell   content.SpellID
	Highest int
}

// SpellMasteryRejectionKind is the cause of a SpellMasteryRejection.
type SpellMasteryRejectionKind int

// The causes, in reporting order.
const (
	SpellMasteryFeatureNotAvailable SpellMasteryRejectionKind = iota + 1
	SpellMasteryNotAfterLongRest
	SpellMasteryNotHeld
	SpellMasteryAlready
	SpellMasteryNotAMasterySpell
	SpellMasteryNotInSpellbook
	SpellMasteryDuplicateSpell
	SpellMasteryWrongSpellLevels
	SpellMasteryCastingTimeNotAction
)

func masteryKindNames() []string {
	return []string{"feature_not_available", "not_after_long_rest", "not_held", "already", "not_a_mastery_spell",
		"not_in_spellbook", "duplicate_spell", "wrong_spell_levels", "casting_time_not_action"}
}

// String returns the snake_case name of the cause.
func (k SpellMasteryRejectionKind) String() string { return kindName(masteryKindNames(), int(k)) }

// SpellMasteryRejection says why Spell Mastery was refused; Spell is the offender
// and Levels the levels of the spells named (WrongSpellLevels).
type SpellMasteryRejection struct {
	Kind   SpellMasteryRejectionKind
	Spell  content.SpellID
	Levels []int
}

// SignatureSpellsRejectionKind is the cause of a SignatureSpellsRejection.
type SignatureSpellsRejectionKind int

// The causes, in reporting order.
const (
	SignatureSpellsFeatureNotAvailable SignatureSpellsRejectionKind = iota + 1
	SignatureSpellsAlready
	SignatureSpellsWrongSpellCount
	SignatureSpellsDuplicateSpell
	SignatureSpellsNotLevelThree
	SignatureSpellsNotInSpellbook
)

func signatureKindNames() []string {
	return []string{"feature_not_available", "already", "wrong_spell_count", "duplicate_spell", "not_level_three", "not_in_spellbook"}
}

// String returns the snake_case name of the cause.
func (k SignatureSpellsRejectionKind) String() string { return kindName(signatureKindNames(), int(k)) }

// SignatureSpellsRejection says why Signature Spells were refused; Allowed is the
// count that had to be named (WrongSpellCount), Spell the offender of the others.
type SignatureSpellsRejection struct {
	Kind    SignatureSpellsRejectionKind
	Allowed int
	Spell   content.SpellID
}

// RitualAdeptRejectionKind is the cause of a RitualAdeptRejection.
type RitualAdeptRejectionKind int

// The causes, in reporting order.
const (
	RitualAdeptSpellbookUnavailable RitualAdeptRejectionKind = iota + 1
	RitualAdeptNotInSpellbook
	RitualAdeptNotARitualSpell
)

func ritualKindNames() []string {
	return []string{"spellbook_unavailable", "not_in_spellbook", "not_a_ritual_spell"}
}

// String returns the snake_case name of the cause.
func (k RitualAdeptRejectionKind) String() string { return kindName(ritualKindNames(), int(k)) }

// RitualAdeptRejection says why a spell cannot be cast as a Ritual with Ritual Adept.
type RitualAdeptRejection struct{ Kind RitualAdeptRejectionKind }

// WizardCreationRejectionKind is the part of the level 1 choices that failed first.
type WizardCreationRejectionKind int

// The parts, in reporting order.
const (
	WizardCreationSkillsInvalid WizardCreationRejectionKind = iota + 1
	WizardCreationEquipmentInvalid
	WizardCreationCantripsInvalid
	WizardCreationSpellbookInvalid
)

func creationKindNames() []string {
	return []string{"skills_invalid", "equipment_invalid", "cantrips_invalid", "spellbook_invalid"}
}

// String returns the snake_case name of the part.
func (k WizardCreationRejectionKind) String() string { return kindName(creationKindNames(), int(k)) }

// WizardCreationRejection names a part of the level 1 choices that failed and that
// part's rejection (only the one matching Kind is set). WizardCreationError carries it in the error NewWizardCreationChoices returns.
type WizardCreationRejection struct {
	Kind      WizardCreationRejectionKind
	Skills    SkillsRejection
	Equipment EquipmentRejection
	Cantrips  CantripChoiceRejection
	Spellbook SpellbookRejection
}

// String returns the snake_case name of the failed part followed by its cause.
func (r WizardCreationRejection) String() string {
	switch r.Kind {
	case WizardCreationSkillsInvalid:
		return r.Kind.String() + ":" + r.Skills.Kind.String()
	case WizardCreationEquipmentInvalid:
		return r.Kind.String() + ":" + r.Equipment.Kind.String()
	case WizardCreationCantripsInvalid:
		return r.Kind.String() + ":" + r.Cantrips.Kind.String()
	case WizardCreationSpellbookInvalid:
		return r.Kind.String() + ":" + r.Spellbook.Kind.String()
	}
	return "unknown"
}

// WizardCreationError is the error NewWizardCreationChoices joins, one per failing part. It matches
// srderr.ErrInvalidInput.
type WizardCreationError struct{ Rejection WizardCreationRejection }

// Error renders the part and cause that failed.
func (e WizardCreationError) Error() string {
	return "character: Wizard creation: " + e.Rejection.String()
}

// Unwrap returns srderr.ErrInvalidInput.
func (WizardCreationError) Unwrap() error { return srderr.ErrInvalidInput }
