package character

import (
	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
)

// Event is something that happened to a Character, in the past tense and
// immutable. Every event carries the Identifier it concerns. The interface
// cannot be implemented outside this package. Members: Created,
// ExperiencePointsGained, LevelGained, AbilityScoreChanged,
// HitPointMaximumChanged, HeroicInspirationGained, HeroicInspirationExpended,
// RestApplied, RestInterrupted, HitPointDieSpent, MountedPaceBoostSpent,
// MountedPaceBoostRestored, LightFuelExhausted, SpellSlotExpended, SpellSlotsRestored,
// PreparedSpellsChanged, MagicItemAttuned, MagicItemAttunementEnded, MagicItemConsumed,
// MagicItemChargesSpent, MagicItemRecharged, SentientControlTaken, SentientControlEnded,
// SubclassChosen, SpellCast, CastingCountered, ConcentrationBegan, ConcentrationEnded,
// CastingFailed, MagicItemLastChargeResolved, RingSpellStored,
// RingStoredSpellCast, StaffSpellAbsorbed, FeatTaken, FeatSpellReplaced, FeatUseExpended,
// FeatUsesRenewed, FeatReplaced, WishDuplicated, WishStressEndured, WishWeaknessEnded,
// WishCastDamageEnded, WishStressDamageRolled, SecondWindUsed, TacticalMindUsed, ActionSurgeUsed,
// IndomitableUsed, FighterFightingStyleChosen, FighterFightingStyleReplaced,
// FighterWeaponMasteriesChosen, FighterWeaponMasterySwapped, RageStarted, RageEnded,
// RelentlessRageUsed, RagesRegained, IntimidatingPresenceUsed, BarbarianWeaponMasteriesChosen,
// BarbarianWeaponMasterySwapped, FocusPointsExpended, FocusPointsRegained, UncannyMetabolismUsed,
// AttackDeflected, DeflectedForceRedirected, WholenessOfBodyUsed, StunningStrikeAttempted,
// QuiveringPalmStarted, QuiveringPalmEnded, DisciplinedSurvivorUsed, SuperiorDefenseActivated, StrokeOfLuckUsed,
// WizardSkillsChosen, WizardEquipmentChosen, WizardCantripsChosen, ScholarExpertiseChosen, SpellsAddedToSpellbook,
// ArcaneRecoveryUsed, SpellMemorized, SpellMasteryChosen, SpellMasteryReplaced, SignatureSpellsChosen, OverchannelUsed, ClericOptionChosen, ClericAdvanced, ClericChannelDivinityUsed,
// ClericChannelDivinityRegained, DivineInterventionUsed, DivineInterventionRecharged, DruidLevelGained, PrimalOrderChosen, ElementalFuryChosen,
// LandTypeChosen, KnownFormsChosen, KnownFormReplaced, WildShapeAssumed, WildShapeEnded, WildCompanionCast, WildCompanionDismissed, WildShapeUsesChanged, SpellSlotsExchanged,
// LandsAidUsed, NaturalRecoveryUsed, NaturesSanctuaryCreated, NaturesSanctuaryMoved, NaturesSanctuaryEnded, PrimalStrikeUsed, BardicInspirationGranted,
// BardicInspirationExpended, BardicInspirationExpired, BardicInspirationRestored, BardExpertiseGained, InnateSorceryActivated, InnateSorceryEnded, SpellSlotConvertedToSorceryPoints, SpellSlotCreated, CreatedSlotSpent, CreatedSlotsExpired, MetamagicUsed, SorceryPointsRestored, MetamagicOptionsChosen, ElementalAffinityChosen, ElementalAffinityApplied, DragonWingsActivated, DragonWingsEnded, DragonWingsUseRestored, DragonCompanionCast, InvocationsChosen, MysticArcanumChosen, MagicalCunningUsed, ContactPatronUsed, MysticArcanumCast, InvocationSpellCastWithoutSlot, DarkOnesBlessingGained, FiendishResilienceChosen, DarkOnesOwnLuckUsed, HurlThroughHellUsed, HurlThroughHellRestored, EldritchSmiteUsed, WarlockFeaturesRestored, PactWeaponBonded, PactWeaponBondEnded, FamiliarFormChosen, FamiliarSpeedChosen, FamiliarAttackForgone, FamiliarAttackCommanded, FamiliarResistanceGranted, BookOfShadowsConjured, BookOfShadowsLost, GiftOfTheProtectorsNameWritten, GiftOfTheProtectorsNameErased, GiftOfTheProtectorsTriggered, LifedrinkerUsed, GazeLinkBegun, GazeLinkMaintained, LayOnHandsUsed, PaladinChannelDivinityUsed, FreeCastSpent, HolyNimbusActivated, HolyNimbusRestored, SmiteOfProtectionTriggered, PaladinEffectEnded, PaladinChoiceMade, PaladinLevelOneGained, PaladinAdvanced, PaladinResourcesRestored, FavoredEnemyCastSpent, TirelessUsed, NaturesVeilUsed, ExhaustionReductionGranted, RangerChoiceMade, RangerChoiceReplaced, SpeciesUseExpended, SpeciesUsesRenewed, HighElfCantripReplaced, HalflingLuckUsed, RelentlessEnduranceUsed, StonesEnduranceUsed.
type Event interface {
	// CharacterID returns the id of the Character the event concerns.
	CharacterID() Identifier
	isEvent()
}

// RecordEvent is the closed sub-family of Event produced by the Character
// aggregate itself, so a consumer can switch over it exhaustively. Members:
// Created, ExperiencePointsGained, LevelGained, AbilityScoreChanged,
// HitPointMaximumChanged, HeroicInspirationGained, HeroicInspirationExpended.
type RecordEvent interface {
	Event
	isRecordEvent()
}

// Created is the event of a new Character. Its fields are read through the
// accessors.
type Created struct {
	id              Identifier
	class           content.ClassID
	species         content.SpeciesID
	background      content.BackgroundID
	scores          ability.Scores
	experience      ExperiencePoints
	hitPointMaximum HitPointMaximum
}

// CharacterID returns the id of the new Character.
func (e Created) CharacterID() Identifier { return e.id }

// Class returns the initial class.
func (e Created) Class() content.ClassID { return e.class }

// Species returns the species.
func (e Created) Species() content.SpeciesID { return e.species }

// Background returns the background.
func (e Created) Background() content.BackgroundID { return e.background }

// AbilityScores returns the starting ability scores.
func (e Created) AbilityScores() ability.Scores { return e.scores }

// ExperiencePoints returns the starting Experience Points.
func (e Created) ExperiencePoints() ExperiencePoints { return e.experience }

// HitPointMaximum returns the initial Hit Point maximum.
func (e Created) HitPointMaximum() HitPointMaximum { return e.hitPointMaximum }

// ExperiencePointsGained is the event of an Experience Points award.
type ExperiencePointsGained struct {
	id     Identifier
	amount ExperienceAmount
	total  ExperiencePoints
}

// CharacterID returns the id of the Character.
func (e ExperiencePointsGained) CharacterID() Identifier { return e.id }

// Amount returns the Experience Points awarded.
func (e ExperiencePointsGained) Amount() ExperienceAmount { return e.amount }

// Total returns the Experience Points after the award.
func (e ExperiencePointsGained) Total() ExperiencePoints { return e.total }

// LevelGained is the event of gaining a level.
type LevelGained struct {
	id              Identifier
	class           content.ClassID
	classLevel      int
	totalLevel      Level
	hitPointsGained int
	hitPointMaximum HitPointMaximum
	proficiency     ProficiencyBonus
}

// CharacterID returns the id of the Character.
func (e LevelGained) CharacterID() Identifier { return e.id }

// Class returns the class the level was gained in.
func (e LevelGained) Class() content.ClassID { return e.class }

// ClassLevel returns the new level in that class.
func (e LevelGained) ClassLevel() int { return e.classLevel }

// TotalLevel returns the new total level.
func (e LevelGained) TotalLevel() Level { return e.totalLevel }

// HitPointsGained returns the Hit Points the level added to the maximum.
func (e LevelGained) HitPointsGained() int { return e.hitPointsGained }

// HitPointMaximum returns the new Hit Point maximum.
func (e LevelGained) HitPointMaximum() HitPointMaximum { return e.hitPointMaximum }

// ProficiencyBonus returns the Proficiency Bonus at the new total level.
func (e LevelGained) ProficiencyBonus() ProficiencyBonus { return e.proficiency }

// AbilityScoreChanged is the event of one ability score changing.
type AbilityScoreChanged struct {
	id       Identifier
	ability  ability.Ability
	previous ability.Score
	current  ability.Score
}

// CharacterID returns the id of the Character.
func (e AbilityScoreChanged) CharacterID() Identifier { return e.id }

// Ability returns the ability that changed.
func (e AbilityScoreChanged) Ability() ability.Ability { return e.ability }

// Previous returns the score before the change.
func (e AbilityScoreChanged) Previous() ability.Score { return e.previous }

// Current returns the score after the change.
func (e AbilityScoreChanged) Current() ability.Score { return e.current }

// HitPointMaximumChanged is the event of the stored Hit Point maximum changing
// outside a level-up.
type HitPointMaximumChanged struct {
	id       Identifier
	previous HitPointMaximum
	current  HitPointMaximum
}

// CharacterID returns the id of the Character.
func (e HitPointMaximumChanged) CharacterID() Identifier { return e.id }

// Previous returns the maximum before the change.
func (e HitPointMaximumChanged) Previous() HitPointMaximum { return e.previous }

// Current returns the maximum after the change.
func (e HitPointMaximumChanged) Current() HitPointMaximum { return e.current }

// HeroicInspirationGained is the event of a Character gaining Heroic
// Inspiration.
type HeroicInspirationGained struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e HeroicInspirationGained) CharacterID() Identifier { return e.id }

// HeroicInspirationExpended is the event of a Character expending Heroic
// Inspiration.
type HeroicInspirationExpended struct{ id Identifier }

// CharacterID returns the id of the Character.
func (e HeroicInspirationExpended) CharacterID() Identifier { return e.id }

func (Created) isEvent()                   {}
func (ExperiencePointsGained) isEvent()    {}
func (LevelGained) isEvent()               {}
func (AbilityScoreChanged) isEvent()       {}
func (HitPointMaximumChanged) isEvent()    {}
func (HeroicInspirationGained) isEvent()   {}
func (HeroicInspirationExpended) isEvent() {}

func (Created) isRecordEvent()                   {}
func (ExperiencePointsGained) isRecordEvent()    {}
func (LevelGained) isRecordEvent()               {}
func (AbilityScoreChanged) isRecordEvent()       {}
func (HitPointMaximumChanged) isRecordEvent()    {}
func (HeroicInspirationGained) isRecordEvent()   {}
func (HeroicInspirationExpended) isRecordEvent() {}
