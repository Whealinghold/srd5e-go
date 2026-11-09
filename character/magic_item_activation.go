package character

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// MagicItemUseState is whether a consumable item is still usable. The zero value
// is invalid.
type MagicItemUseState int

// The use states (SRD 5.2.1, "Consumable Items", p. 206).
const (
	ItemIntact MagicItemUseState = iota + 1
	ItemConsumed
)

// MagicItemUseResult is the closed set of outcomes of MagicItemUse.Use.
// Variants: MagicItemUseApplied, MagicItemAlreadyConsumed.
type MagicItemUseResult interface{ isMagicItemUseResult() }

// MagicItemUseApplied: the item was used; State is its new state and Events the
// events raised (MagicItemConsumed for a consumable, none otherwise).
type MagicItemUseApplied struct {
	State  MagicItemUseState
	events []MagicItemEvent
}

// Events returns a new slice of the events.
func (r MagicItemUseApplied) Events() []MagicItemEvent { return slices.Clone(r.events) }

// MagicItemAlreadyConsumed: the item was already consumed; nothing changed.
type MagicItemAlreadyConsumed struct{}

func (MagicItemUseApplied) isMagicItemUseResult()      {}
func (MagicItemAlreadyConsumed) isMagicItemUseResult() {}

// MagicItemUse uses a magic item. The zero value is ready to use.
type MagicItemUse struct{}

// Use uses the item: a consumable becomes ItemConsumed with a MagicItemConsumed
// event naming holder and instance; any other item stays ItemIntact. Using a
// consumed item is MagicItemAlreadyConsumed. It returns an error matching
// srderr.ErrInvalidInput for a zero holder, a blank instance or an invalid state.
func (MagicItemUse) Use(holder Identifier, instance MagicItemInstanceID, consumable bool, state MagicItemUseState) (MagicItemUseResult, error) {
	if holder.IsZero() || !instance.Valid() || state != ItemIntact && state != ItemConsumed {
		return nil, invalidInput("use magic item: zero holder, blank instance or invalid state")
	}
	if state == ItemConsumed {
		return MagicItemAlreadyConsumed{}, nil
	}
	if consumable {
		return MagicItemUseApplied{State: ItemConsumed, events: []MagicItemEvent{MagicItemConsumed{id: holder, instance: instance}}}, nil
	}
	return MagicItemUseApplied{State: ItemIntact}, nil
}

// CommandWordDelivery is how a Command Word is delivered. The zero value is invalid.
type CommandWordDelivery int

// The deliveries (SRD 5.2.1, "Command Word", p. 206).
const (
	DeliverySpoken CommandWordDelivery = iota + 1
	DeliverySigned
)

// SoundEnvironment is whether sound carries. The zero value is invalid.
type SoundEnvironment int

// The sound environments.
const (
	SoundAudible SoundEnvironment = iota + 1
	SoundSuppressed
)

// CommandWordResult is the closed set of outcomes of CommandWord.Attempt.
// Variants: MagicItemCommandWordAccepted, MagicItemCommandWordFailed.
type CommandWordResult interface{ isCommandWordResult() }

// MagicItemCommandWordAccepted: the item heard or saw the Command Word.
type MagicItemCommandWordAccepted struct{}

// MagicItemCommandWordFailed: a spoken Command Word in suppressed sound.
type MagicItemCommandWordFailed struct{ Environment SoundEnvironment }

func (MagicItemCommandWordAccepted) isCommandWordResult() {}
func (MagicItemCommandWordFailed) isCommandWordResult()   {}

// CommandWord decides whether a Command Word works. The zero value is ready to use.
type CommandWord struct{}

// Attempt accepts a signed word in either environment and a spoken one only
// when sound is audible. It returns an error matching srderr.ErrInvalidInput for
// an invalid delivery or environment.
func (CommandWord) Attempt(delivery CommandWordDelivery, env SoundEnvironment) (CommandWordResult, error) {
	if delivery != DeliverySpoken && delivery != DeliverySigned || env != SoundAudible && env != SoundSuppressed {
		return nil, invalidInput("command word: invalid delivery or environment")
	}
	if delivery == DeliverySpoken && env == SoundSuppressed {
		return MagicItemCommandWordFailed{Environment: env}, nil
	}
	return MagicItemCommandWordAccepted{}, nil
}

// ScrollReadingResult is the closed set of outcomes of ScrollReading.CanAttempt.
// Variants: MagicItemScrollReadable, MagicItemScrollNotReadable.
type ScrollReadingResult interface{ isScrollReadingResult() }

// MagicItemScrollReadable: the creature may attempt to use the scroll.
type MagicItemScrollReadable struct{}

// MagicItemScrollNotReadable: the creature cannot read the scroll; Restricted
// says the description restricts reading, otherwise the language is not understood.
type MagicItemScrollNotReadable struct{ Restricted bool }

func (MagicItemScrollReadable) isScrollReadingResult()    {}
func (MagicItemScrollNotReadable) isScrollReadingResult() {}

// ScrollReading decides who may use a scroll. The zero value is ready to use.
type ScrollReading struct{}

// CanAttempt allows the attempt when the creature understands a written
// language and the description does not restrict reading (SRD 5.2.1, "Using a
// Scroll", p. 205). It does not roll the activation.
func (ScrollReading) CanAttempt(understandsWrittenLanguage, descriptionRestricts bool) ScrollReadingResult {
	if understandsWrittenLanguage && !descriptionRestricts {
		return MagicItemScrollReadable{}
	}
	return MagicItemScrollNotReadable{Restricted: descriptionRestricts}
}

// AmmunitionMagic decides when ammunition counts as magical. The zero value is
// ready to use.
type AmmunitionMagic struct{}

// IsMagical is true exactly when the firing weapon is magic and has the
// Ammunition property (SRD 5.2.1, "Weapons", p. 205).
func (AmmunitionMagic) IsMagical(firingWeaponIsMagic, firingWeaponHasAmmunitionProperty bool) bool {
	return firingWeaponIsMagic && firingWeaponHasAmmunitionProperty
}

// ItemCasterLevel is the caster level of a spell cast from an item. The zero
// value is invalid.
type ItemCasterLevel int

// ItemCasterLevelLowest marks "the lowest possible caster level": the spell's
// lowest level and, for a cantrip, its base unscaled tier.
const ItemCasterLevelLowest ItemCasterLevel = 1

// ItemSpellComponents is whether casting from the item needs components. The zero value is invalid.
type ItemSpellComponents int

// The component rules.
const (
	NoComponentsRequired ItemSpellComponents = iota + 1
	ComponentsPerDescription
)

// CastingTerms is whether the spell keeps its own casting time, range and
// duration. The zero value is invalid.
type CastingTerms int

// The casting terms.
const (
	StandardCastingTerms CastingTerms = iota + 1
	CastingTermsPerDescription
)

// SpellDelivery is how the item delivers the spell. The zero value is invalid.
type SpellDelivery int

// The deliveries.
const (
	CastFromItem SpellDelivery = iota + 1
	BypassesCasting
)

// SpellFromItem is a spell cast from a magic item (SRD 5.2.1, "Spells Cast from
// Items", p. 206). It has no spell-slot field, so it cannot spend the user's
// slots. Immutable and comparable; the zero value is invalid.
type SpellFromItem struct {
	spell       content.SpellID
	castAt      content.SpellLevel
	components  ItemSpellComponents
	terms       CastingTerms
	delivery    SpellDelivery
	concentrate bool
}

// NewSpellFromItem returns the spell cast from an item at castAt, which MUST NOT
// be below lowest, the spell's lowest possible level. spellRequiresConcentration
// is the spell's own Concentration flag. It returns an error matching
// srderr.ErrInvalidInput for an invalid spell, level, components, terms or
// delivery, or castAt below lowest.
func NewSpellFromItem(spell content.SpellID, lowest, castAt content.SpellLevel, components ItemSpellComponents, terms CastingTerms, delivery SpellDelivery, spellRequiresConcentration bool) (SpellFromItem, error) {
	okC := components == NoComponentsRequired || components == ComponentsPerDescription
	okT := terms == StandardCastingTerms || terms == CastingTermsPerDescription
	okD := delivery == CastFromItem || delivery == BypassesCasting
	if !spell.Valid() || !okC || !okT || !okD || castAt < lowest {
		return SpellFromItem{}, invalidInput("spell from item: invalid spell, level, components, terms or delivery")
	}
	return SpellFromItem{spell: spell, castAt: castAt, components: components, terms: terms, delivery: delivery,
		concentrate: delivery == CastFromItem && spellRequiresConcentration}, nil
}

// SpellID returns the spell.
func (s SpellFromItem) SpellID() content.SpellID { return s.spell }

// CastAtLevel returns the level the spell is cast at.
func (s SpellFromItem) CastAtLevel() content.SpellLevel { return s.castAt }

// CasterLevel returns ItemCasterLevelLowest.
func (SpellFromItem) CasterLevel() ItemCasterLevel { return ItemCasterLevelLowest }

// Components returns the component rule.
func (s SpellFromItem) Components() ItemSpellComponents { return s.components }

// CastingTerms returns the casting terms.
func (s SpellFromItem) CastingTerms() CastingTerms { return s.terms }

// Delivery returns how the item delivers the spell.
func (s SpellFromItem) Delivery() SpellDelivery { return s.delivery }

// UserMustConcentrate is true exactly when the delivery is CastFromItem and the
// spell requires Concentration; bypassing casting is not a cast.
func (s SpellFromItem) UserMustConcentrate() bool { return s.concentrate }

// UserSpellAbility is one spellcasting ability the user has and its modifier.
// Construct it with a keyed literal.
type UserSpellAbility struct {
	Ability  ability.Ability
	Modifier ability.Modifier
}

// ItemSpellcastingResult is the closed set of outcomes of
// ItemSpellcastingAbility.Resolve. Variants: ItemSpellcastingResolved,
// ItemSpellcastingAbilityNotAvailable.
type ItemSpellcastingResult interface{ isItemSpellcastingResult() }

// ItemSpellcastingResolved: the ability used (zero when the user has none), its
// modifier (+0 then) and the Proficiency Bonus, the inputs of the spell save DC
// and attack bonus.
type ItemSpellcastingResolved struct {
	Ability          ability.Ability
	Modifier         ability.Modifier
	ProficiencyBonus ProficiencyBonus
}

// ItemSpellcastingAbilityNotAvailable: the user has several abilities and the
// choice is missing or not one of them.
type ItemSpellcastingAbilityNotAvailable struct{}

func (ItemSpellcastingResolved) isItemSpellcastingResult()            {}
func (ItemSpellcastingAbilityNotAvailable) isItemSpellcastingResult() {}

// ItemSpellcastingAbility resolves the ability an item that uses the user's
// spellcasting ability applies. The zero value is ready to use.
type ItemSpellcastingAbility struct{}

// Resolve uses the one ability the user has; the chosen one when there are
// several; and +0 with the Proficiency Bonus when there is none. A supplied
// choice is ignored for one or none.
func (ItemSpellcastingAbility) Resolve(user []UserSpellAbility, choice ability.Ability, pb ProficiencyBonus) ItemSpellcastingResult {
	switch len(user) {
	case 0:
		return ItemSpellcastingResolved{ProficiencyBonus: pb}
	case 1:
		return ItemSpellcastingResolved{Ability: user[0].Ability, Modifier: user[0].Modifier, ProficiencyBonus: pb}
	}
	if i := slices.IndexFunc(user, func(u UserSpellAbility) bool { return u.Ability == choice }); i >= 0 {
		return ItemSpellcastingResolved{Ability: choice, Modifier: user[i].Modifier, ProficiencyBonus: pb}
	}
	return ItemSpellcastingAbilityNotAvailable{}
}
