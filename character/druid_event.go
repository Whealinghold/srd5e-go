package character

import (
	"slices"
	"time"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/dice"
)

// DruidEvent is something that happened to a Druid, in the past tense and immutable, the closed family of Druid events
// (class-druid.md FR-166). Every one is also an Event and carries the Identifier of the character from the DruidCharacterFacts.
// They are outside the Character aggregate's RecordEvent family. The interface cannot be implemented outside this package.
// Members: DruidLevelGained, PrimalOrderChosen, ElementalFuryChosen, LandTypeChosen, KnownFormsChosen, KnownFormReplaced,
// WildShapeAssumed, WildShapeEnded, WildCompanionCast, WildCompanionDismissed, WildShapeUsesChanged, SpellSlotsExchanged,
// LandsAidUsed, NaturalRecoveryUsed, NaturesSanctuaryCreated, NaturesSanctuaryMoved, NaturesSanctuaryEnded, PrimalStrikeUsed.
type DruidEvent interface {
	Event
	isDruidEvent()
}

// DruidPendingChoiceKind says which choice a Druid level leaves open.
type DruidPendingChoiceKind int

const (
	// DruidPendingPrimalOrder: choose the Primal Order (level 1).
	DruidPendingPrimalOrder DruidPendingChoiceKind = iota + 1
	// DruidPendingKnownForms: choose Count more known Beast forms (levels 2, 4 and 8).
	DruidPendingKnownForms
	// DruidPendingElementalFury: choose the Elemental Fury option (level 7).
	DruidPendingElementalFury
)

// DruidPendingChoice is a Druid-owned choice a level gain opens. Count is the number of forms for DruidPendingKnownForms and
// 1 otherwise. Comparable with ==; construct it with a keyed literal.
type DruidPendingChoice struct {
	Kind  DruidPendingChoiceKind
	Count int
}

// DruidLevelGained is the event of a Druid level gained.
type DruidLevelGained struct {
	id       Identifier
	level    int
	features []content.FeatureID
	pending  []DruidPendingChoice
}

// CharacterID returns the id of the Character.
func (e DruidLevelGained) CharacterID() Identifier { return e.id }

// Level returns the new Druid level.
func (e DruidLevelGained) Level() int { return e.level }

// Features returns the ids of the features granted at exactly the new level, as a new slice.
func (e DruidLevelGained) Features() []content.FeatureID { return slices.Clone(e.features) }

// PendingChoices returns the Druid-owned choices the level opens, as a new slice.
func (e DruidLevelGained) PendingChoices() []DruidPendingChoice { return slices.Clone(e.pending) }

// PrimalOrderChosen is the event of the Primal Order chosen.
type PrimalOrderChosen struct {
	id    Identifier
	order PrimalOrder
}

// CharacterID returns the id of the Character.
func (e PrimalOrderChosen) CharacterID() Identifier { return e.id }

// Order returns the order chosen.
func (e PrimalOrderChosen) Order() PrimalOrder { return e.order }

// ElementalFuryChosen is the event of the Elemental Fury option chosen.
type ElementalFuryChosen struct {
	id     Identifier
	option ElementalFury
}

// CharacterID returns the id of the Character.
func (e ElementalFuryChosen) CharacterID() Identifier { return e.id }

// Option returns the option chosen.
func (e ElementalFuryChosen) Option() ElementalFury { return e.option }

// LandTypeChosen is the event of a land type chosen, the first time or after a Long Rest.
type LandTypeChosen struct {
	id   Identifier
	land LandType
}

// CharacterID returns the id of the Character.
func (e LandTypeChosen) CharacterID() Identifier { return e.id }

// Land returns the land chosen.
func (e LandTypeChosen) Land() LandType { return e.land }

// KnownFormsChosen is the event of Beast forms added to the known forms.
type KnownFormsChosen struct {
	id    Identifier
	forms []content.MonsterID
}

// CharacterID returns the id of the Character.
func (e KnownFormsChosen) CharacterID() Identifier { return e.id }

// Forms returns the forms added, in order, as a new slice.
func (e KnownFormsChosen) Forms() []content.MonsterID { return slices.Clone(e.forms) }

// KnownFormReplaced is the event of one known form replaced by another.
type KnownFormReplaced struct {
	id                  Identifier
	replaced, replacing content.MonsterID
}

// CharacterID returns the id of the Character.
func (e KnownFormReplaced) CharacterID() Identifier { return e.id }

// Replaced returns the form that is no longer known.
func (e KnownFormReplaced) Replaced() content.MonsterID { return e.replaced }

// Replacing returns the form now known in its place.
func (e KnownFormReplaced) Replacing() content.MonsterID { return e.replacing }

// WildShapeAssumed is the event of a Beast form assumed. The Temporary Hit Points are an amount for the consumer to apply
// (class-druid.md FR-071); the event holds no Concentration field (FR-083).
type WildShapeAssumed struct {
	id           Identifier
	form         content.MonsterID
	duration     time.Duration
	temporaryHP  int
	worn, merged []DruidItemKey
}

// CharacterID returns the id of the Character.
func (e WildShapeAssumed) CharacterID() Identifier { return e.id }

// Form returns the Beast form assumed.
func (e WildShapeAssumed) Form() content.MonsterID { return e.form }

// Duration returns how long the form lasts: half the Druid level in hours, exactly.
func (e WildShapeAssumed) Duration() time.Duration { return e.duration }

// TemporaryHitPoints returns the Temporary Hit Points to apply: the Druid level.
func (e WildShapeAssumed) TemporaryHitPoints() int { return e.temporaryHP }

// WornItems returns the items that keep functioning in the form, as a new slice.
func (e WildShapeAssumed) WornItems() []DruidItemKey { return slices.Clone(e.worn) }

// MergedItems returns the items that merged into the form and do not function, as a new slice.
func (e WildShapeAssumed) MergedItems() []DruidItemKey { return slices.Clone(e.merged) }

// ActionCost returns DruidActionCostBonus.
func (WildShapeAssumed) ActionCost() DruidActionCost { return DruidActionCostBonus }

// WildShapeEnded is the event of a Beast form ended. A use is never refunded.
type WildShapeEnded struct {
	id     Identifier
	form   content.MonsterID
	reason WildShapeEndReason
}

// CharacterID returns the id of the Character.
func (e WildShapeEnded) CharacterID() Identifier { return e.id }

// Form returns the Beast form that ended.
func (e WildShapeEnded) Form() content.MonsterID { return e.form }

// Reason returns why the form ended.
func (e WildShapeEnded) Reason() WildShapeEndReason { return e.reason }

// ActionCost returns DruidActionCostBonus for a form left early and DruidActionCostNone otherwise.
func (e WildShapeEnded) ActionCost() DruidActionCost {
	if e.reason == WildShapeEndLeftEarly {
		return DruidActionCostBonus
	}
	return DruidActionCostNone
}

// DruidPayment says what a Wild Companion cast was paid with. Variants: DruidPaymentSlot, DruidPaymentWildShape.
type DruidPayment interface{ isDruidPayment() }

// DruidPaymentSlot pays with one spell slot of Level (1 or higher) from Pool. Construct it with a keyed literal.
type DruidPaymentSlot struct {
	Pool  SpellSlotPool
	Level int
}

// DruidPaymentWildShape pays with one use of Wild Shape.
type DruidPaymentWildShape struct{}

func (DruidPaymentSlot) isDruidPayment()      {}
func (DruidPaymentWildShape) isDruidPayment() {}

// WildCompanionCast is the event of Find Familiar cast through Wild Companion; the familiar is Fey.
type WildCompanionCast struct {
	id      Identifier
	payment DruidPayment
}

// CharacterID returns the id of the Character.
func (e WildCompanionCast) CharacterID() Identifier { return e.id }

// Payment returns what the cast was paid with.
func (e WildCompanionCast) Payment() DruidPayment { return e.payment }

// ActionCost returns DruidActionCostMagic.
func (WildCompanionCast) ActionCost() DruidActionCost { return DruidActionCostMagic }

// FamiliarType returns CreatureTypeFey: the familiar summoned this way is Fey.
func (WildCompanionCast) FamiliarType() content.CreatureType { return content.CreatureTypeFey }

// WildCompanionDismissed is the event of the Wild Companion familiar disappearing at the end of a Long Rest.
type WildCompanionDismissed struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e WildCompanionDismissed) CharacterID() Identifier { return e.id }

// WildShapeUsesCause is why the Wild Shape uses changed outside a feature use.
type WildShapeUsesCause int

const (
	// WildShapeUsesEvergreenWildShape: Evergreen Wild Shape regained a use on rolling Initiative.
	WildShapeUsesEvergreenWildShape WildShapeUsesCause = iota + 1
)

// WildShapeUsesChanged is the event of the Wild Shape uses changing by a passive effect.
type WildShapeUsesChanged struct {
	id       Identifier
	from, to int
	cause    WildShapeUsesCause
}

// CharacterID returns the id of the Character.
func (e WildShapeUsesChanged) CharacterID() Identifier { return e.id }

// From returns the uses remaining before.
func (e WildShapeUsesChanged) From() int { return e.from }

// To returns the uses remaining after.
func (e WildShapeUsesChanged) To() int { return e.to }

// Cause returns why the uses changed.
func (e WildShapeUsesChanged) Cause() WildShapeUsesCause { return e.cause }

// SlotExchangeKind says which exchange between spell slots and Wild Shape uses happened.
type SlotExchangeKind int

const (
	// SlotExchangeSlotForUse: Wild Resurgence spent a slot for a Wild Shape use.
	SlotExchangeSlotForUse SlotExchangeKind = iota + 1
	// SlotExchangeUseForSlot: Wild Resurgence spent a use for a level 1 slot.
	SlotExchangeUseForSlot
	// SlotExchangeNatureMagician: Nature Magician converted uses into a slot of twice their number.
	SlotExchangeNatureMagician
)

// SpellSlotsExchanged is the event of an exchange between spell slots and Wild Shape uses.
type SpellSlotsExchanged struct {
	id        Identifier
	kind      SlotExchangeKind
	slotLevel int
	uses      int
}

// CharacterID returns the id of the Character.
func (e SpellSlotsExchanged) CharacterID() Identifier { return e.id }

// Kind returns which exchange it was.
func (e SpellSlotsExchanged) Kind() SlotExchangeKind { return e.kind }

// SlotLevel returns the level of the slot spent or created.
func (e SpellSlotsExchanged) SlotLevel() int { return e.slotLevel }

// Uses returns the Wild Shape uses gained (1) or spent.
func (e SpellSlotsExchanged) Uses() int { return e.uses }

// LandsAidUsed is the event of Land's Aid: a 10-foot-radius Sphere within 60 feet, a Constitution saving throw against the
// spell save DC, Necrotic damage (half on a success) and healing of one creature chosen in the area.
type LandsAidUsed struct {
	id              Identifier
	dc              SpellSaveDC
	damage, healing dice.Expression
}

// CharacterID returns the id of the Character.
func (e LandsAidUsed) CharacterID() Identifier { return e.id }

// SaveDC returns the spell save DC supplied by the caller.
func (e LandsAidUsed) SaveDC() SpellSaveDC { return e.dc }

// Damage returns the Necrotic damage dice, for the consumer to roll.
func (e LandsAidUsed) Damage() dice.Expression { return e.damage }

// Healing returns the healing dice, for the consumer to roll.
func (e LandsAidUsed) Healing() dice.Expression { return e.healing }

// RangeFeet returns 60, the distance to the point.
func (LandsAidUsed) RangeFeet() int { return 60 }

// RadiusFeet returns 10, the radius of the Sphere.
func (LandsAidUsed) RadiusFeet() int { return 10 }

// ActionCost returns DruidActionCostMagic.
func (LandsAidUsed) ActionCost() DruidActionCost { return DruidActionCostMagic }

// NaturalRecoveryUsed is the event of Natural Recovery: a Circle Spell cast without a slot, or slots recovered.
type NaturalRecoveryUsed struct {
	id    Identifier
	spell content.SpellID
	slots []int
}

// CharacterID returns the id of the Character.
func (e NaturalRecoveryUsed) CharacterID() Identifier { return e.id }

// Spell returns the Circle Spell cast without a slot, or "" for a slot recovery.
func (e NaturalRecoveryUsed) Spell() content.SpellID { return e.spell }

// RecoveredSlots returns the levels of the slots recovered, as a new slice; empty for a cast.
func (e NaturalRecoveryUsed) RecoveredSlots() []int { return slices.Clone(e.slots) }

// NaturesSanctuaryCreated is the event of a 15-foot Cube created on the ground within 120 feet, for 1 minute. The Druid and its
// allies in it have Half Cover and the allies have the Resistance.
type NaturesSanctuaryCreated struct {
	id         Identifier
	resistance content.DamageType
}

// CharacterID returns the id of the Character.
func (e NaturesSanctuaryCreated) CharacterID() Identifier { return e.id }

// Resistance returns the Resistance of Nature's Ward for the current land, and false when no land type is chosen.
func (e NaturesSanctuaryCreated) Resistance() (content.DamageType, bool) {
	return e.resistance, e.resistance != 0
}

// ActionCost returns DruidActionCostMagic.
func (NaturesSanctuaryCreated) ActionCost() DruidActionCost { return DruidActionCostMagic }

// HalfCover reports that the Druid and its allies in the Cube have Half Cover, a fact for the host: this package defines no cover type.
func (NaturesSanctuaryCreated) HalfCover() bool { return true }

// NaturesSanctuaryMoved is the event of the Cube moved up to 60 feet.
type NaturesSanctuaryMoved struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e NaturesSanctuaryMoved) CharacterID() Identifier { return e.id }

// ActionCost returns DruidActionCostBonus.
func (NaturesSanctuaryMoved) ActionCost() DruidActionCost { return DruidActionCostBonus }

// NaturesSanctuaryEnded is the event of the Cube ending.
type NaturesSanctuaryEnded struct {
	id     Identifier
	reason SanctuaryEndReason
}

// CharacterID returns the id of the Character.
func (e NaturesSanctuaryEnded) CharacterID() Identifier { return e.id }

// Reason returns why the Cube ended.
func (e NaturesSanctuaryEnded) Reason() SanctuaryEndReason { return e.reason }

// PrimalStrikeUsed is the event of Primal Strike: extra damage of one of the types, chosen when the attack hits.
type PrimalStrikeUsed struct {
	id    Identifier
	extra dice.Expression
}

// CharacterID returns the id of the Character.
func (e PrimalStrikeUsed) CharacterID() Identifier { return e.id }

// Damage returns the extra damage dice, for the consumer to roll.
func (e PrimalStrikeUsed) Damage() dice.Expression { return e.extra }

// DamageTypes returns Cold, Fire, Lightning and Thunder, of which one is chosen.
func (PrimalStrikeUsed) DamageTypes() []content.DamageType {
	return []content.DamageType{content.DamageCold, content.DamageFire, content.DamageLightning, content.DamageThunder}
}

func (DruidLevelGained) isEvent()             {}
func (PrimalOrderChosen) isEvent()            {}
func (ElementalFuryChosen) isEvent()          {}
func (LandTypeChosen) isEvent()               {}
func (KnownFormsChosen) isEvent()             {}
func (KnownFormReplaced) isEvent()            {}
func (WildShapeAssumed) isEvent()             {}
func (WildShapeEnded) isEvent()               {}
func (WildCompanionCast) isEvent()            {}
func (WildCompanionDismissed) isEvent()       {}
func (WildShapeUsesChanged) isEvent()         {}
func (SpellSlotsExchanged) isEvent()          {}
func (LandsAidUsed) isEvent()                 {}
func (NaturalRecoveryUsed) isEvent()          {}
func (NaturesSanctuaryCreated) isEvent()      {}
func (NaturesSanctuaryMoved) isEvent()        {}
func (NaturesSanctuaryEnded) isEvent()        {}
func (PrimalStrikeUsed) isEvent()             {}
func (DruidLevelGained) isDruidEvent()        {}
func (PrimalOrderChosen) isDruidEvent()       {}
func (ElementalFuryChosen) isDruidEvent()     {}
func (LandTypeChosen) isDruidEvent()          {}
func (KnownFormsChosen) isDruidEvent()        {}
func (KnownFormReplaced) isDruidEvent()       {}
func (WildShapeAssumed) isDruidEvent()        {}
func (WildShapeEnded) isDruidEvent()          {}
func (WildCompanionCast) isDruidEvent()       {}
func (WildCompanionDismissed) isDruidEvent()  {}
func (WildShapeUsesChanged) isDruidEvent()    {}
func (SpellSlotsExchanged) isDruidEvent()     {}
func (LandsAidUsed) isDruidEvent()            {}
func (NaturalRecoveryUsed) isDruidEvent()     {}
func (NaturesSanctuaryCreated) isDruidEvent() {}
func (NaturesSanctuaryMoved) isDruidEvent()   {}
func (NaturesSanctuaryEnded) isDruidEvent()   {}
func (PrimalStrikeUsed) isDruidEvent()        {}
