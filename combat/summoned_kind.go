package combat

import (
	"github.com/Whealinghold/srd5e-go/content"
)

// SummonKind is the spell or feature that brought a creature into the fight (summoned-and-companion-creatures.md
// section 3, Table K). The zero value is invalid.
type SummonKind int

// The eight kinds.
const (
	// SummonAnimatedObject: an object given life by Animate Objects (SRD 5.2.1, p. 108).
	SummonAnimatedObject SummonKind = iota + 1
	// SummonGiantInsect: a creature of Giant Insect (p. 135).
	SummonGiantInsect
	// SummonDraconicSpirit: the spirit of Summon Dragon (p. 166).
	SummonDraconicSpirit
	// SummonOtherworldlySteed: the steed of Find Steed (p. 130).
	SummonOtherworldlySteed
	// SummonAnimatedUndead: a Skeleton or Zombie of Animate Dead (p. 108).
	SummonAnimatedUndead
	// SummonCreatedUndead: an undead creature of Create Undead (p. 121).
	SummonCreatedUndead
	// SummonFamiliar: the familiar of Find Familiar, Pact of the Chain or Wild Companion (p. 130).
	SummonFamiliar
	// SummonSimulacrum: the duplicate of Simulacrum (p. 162).
	SummonSimulacrum
)

func summonKindNames() []string {
	return []string{"animated_object", "giant_insect", "draconic_spirit", "otherworldly_steed", "animated_undead", "created_undead", "familiar", "simulacrum"}
}

// Valid reports whether k is one of the eight kinds.
func (k SummonKind) Valid() bool { return k >= SummonAnimatedObject && k <= SummonSimulacrum }

// String returns the snake_case name of k (the value of srd5e.combat.summon_kind), or "unknown".
func (k SummonKind) String() string { return enumName(summonKindNames(), int(k)) }

// Spell returns the id of the spell that brings a creature of kind k, or the empty id for an undefined kind.
func (k SummonKind) Spell() content.SpellID {
	if !k.Valid() {
		return ""
	}
	return [...]content.SpellID{"animate_objects", "giant_insect", "summon_dragon", "find_steed", "animate_dead", "create_undead", "find_familiar", "simulacrum"}[k-1]
}

// Placement returns the PlacementRule of k: the creature shares its summoner's Initiative count for the kinds the SRD says
// share it, and rolls its own for the undead and the familiar (FR-002, FR-003).
func (k SummonKind) Placement() PlacementRule {
	if k.undead() || k == SummonFamiliar {
		return OwnCount
	}
	return SharesSummonerCount
}

// Channel returns how the summoner commands a creature of kind k (Table K).
func (k SummonKind) Channel() CommandChannel {
	switch k {
	case SummonAnimatedObject:
		return ChannelBonusActionMental{RangeFeet: 500}
	case SummonAnimatedUndead:
		return ChannelBonusActionMental{RangeFeet: 60}
	case SummonCreatedUndead:
		return ChannelBonusActionMental{RangeFeet: 120}
	case SummonGiantInsect, SummonDraconicSpirit:
		return ChannelVerbalNoAction{}
	case SummonOtherworldlySteed, SummonFamiliar, SummonSimulacrum:
		return ChannelObeysIndependently{}
	}
	return ChannelObeysIndependently{}
}

// dodgesByDefault reports whether a creature of kind k with no order takes the Dodge action (FR-013).
func (k SummonKind) dodgesByDefault() bool {
	return k == SummonAnimatedObject || k == SummonGiantInsect || k == SummonDraconicSpirit || k.undead()
}

// commandable reports whether an order can be given to a creature of kind k.
func (k SummonKind) commandable() bool { return k.Valid() && k != SummonOtherworldlySteed }

// undead reports whether k is one of the two undead kinds.
func (k SummonKind) undead() bool { return k == SummonAnimatedUndead || k == SummonCreatedUndead }

// lastsForSpell reports whether a creature of kind k ends with its spell (FR-022): the Concentration kinds; a simulacrum lasts
// until dispelled and is handled apart.
func (k SummonKind) lastsForSpell() bool {
	return k == SummonAnimatedObject || k == SummonGiantInsect || k == SummonDraconicSpirit
}

// dropsItems reports whether a creature of kind k leaves what it carried behind when it goes (FR-019, FR-023, FR-050).
func (k SummonKind) dropsItems() bool {
	return k == SummonGiantInsect || k == SummonDraconicSpirit || k == SummonOtherworldlySteed || k == SummonFamiliar
}

// PlacementRule says whether a summoned creature shares its summoner's Initiative count or has its own. The zero value is
// invalid.
type PlacementRule int

// The two rules.
const (
	// SharesSummonerCount: the creature takes its turn directly after its summoner with the same count.
	SharesSummonerCount PlacementRule = iota + 1
	// OwnCount: the creature rolls its own Initiative.
	OwnCount
)

// Valid reports whether p is one of the two rules.
func (p PlacementRule) Valid() bool { return p == SharesSummonerCount || p == OwnCount }

// String returns "shares_summoner_count", "own_count" or "unknown".
func (p PlacementRule) String() string {
	return enumName([]string{"shares_summoner_count", "own_count"}, int(p))
}

// CommandChannel is how the summoner gives an order to a creature (Table K). Variants: ChannelBonusActionMental,
// ChannelVerbalNoAction, ChannelObeysIndependently. All compare by value.
type CommandChannel interface{ isCommandChannel() }

// ChannelBonusActionMental: a Bonus Action and a mental command to creatures within RangeFeet (Animate Objects, Animate Dead,
// Create Undead). Build it with a keyed literal.
type ChannelBonusActionMental struct{ RangeFeet int }

// ChannelVerbalNoAction: a verbal command that costs no action (Giant Insect, Summon Dragon).
type ChannelVerbalNoAction struct{}

// ChannelObeysIndependently: the creature acts on its own and obeys at no action cost, at any distance (familiar, simulacrum),
// or is a mount (steed).
type ChannelObeysIndependently struct{}

func (ChannelBonusActionMental) isCommandChannel()  {}
func (ChannelVerbalNoAction) isCommandChannel()     {}
func (ChannelObeysIndependently) isCommandChannel() {}

// SummonEndCause is why a summoned creature stopped being linked to its summoner (FR-019 to FR-026). The zero value is
// invalid.
type SummonEndCause int

// The eight causes.
const (
	// SummonEndDroppedToZeroHitPoints: the creature dropped to 0 Hit Points.
	SummonEndDroppedToZeroHitPoints SummonEndCause = iota + 1
	// SummonEndSpellEnded: the spell that made the creature ended.
	SummonEndSpellEnded
	// SummonEndSummonerDied: the summoner died (a steed).
	SummonEndSummonerDied
	// SummonEndLongRestFinished: a Long Rest ended a Wild Companion familiar.
	SummonEndLongRestFinished
	// SummonEndDismissedForever: the summoner dismissed the familiar for good.
	SummonEndDismissedForever
	// SummonEndDismissedTemporarily: the summoner sent the familiar to its pocket dimension.
	SummonEndDismissedTemporarily
	// SummonEndReplaced: a new cast replaced the creature.
	SummonEndReplaced
	// SummonEndControlLapsed: an undead creature's 24 hours of control ran out.
	SummonEndControlLapsed
)

func summonEndCauseNames() []string {
	return []string{"dropped_to_zero_hit_points", "spell_ended", "summoner_died", "long_rest_finished", "dismissed_forever", "dismissed_temporarily", "replaced", "control_lapsed"}
}

// Valid reports whether c is one of the eight causes.
func (c SummonEndCause) Valid() bool {
	return c >= SummonEndDroppedToZeroHitPoints && c <= SummonEndControlLapsed
}

// String returns the snake_case name of c (the value of srd5e.combat.summon_end_cause), or "unknown".
func (c SummonEndCause) String() string { return enumName(summonEndCauseNames(), int(c)) }

// removal returns the RemovalCause the host passes to RemoveCombatantFromEncounter for c (FR-006): Departed for a familiar
// dismissed or ended by a Long Rest, Disappeared for every other cause.
func (c SummonEndCause) removal() RemovalCause {
	if c == SummonEndDismissedForever || c == SummonEndDismissedTemporarily || c == SummonEndLongRestFinished {
		return RemovalCauseDeparted
	}
	return RemovalCauseDisappeared
}

// FamiliarOrigin is what gave a familiar to its summoner (FR-043, FR-053, FR-054). The zero value is invalid.
type FamiliarOrigin int

// The three origins.
const (
	// FamiliarOriginFindFamiliarSpell: the Find Familiar spell.
	FamiliarOriginFindFamiliarSpell FamiliarOrigin = iota + 1
	// FamiliarOriginPactOfTheChain: the Warlock's Pact of the Chain, which allows the special forms.
	FamiliarOriginPactOfTheChain
	// FamiliarOriginWildCompanion: the Druid's Wild Companion, whose familiar is Fey and leaves at a Long Rest.
	FamiliarOriginWildCompanion
)

// Valid reports whether o is one of the three origins.
func (o FamiliarOrigin) Valid() bool {
	return o >= FamiliarOriginFindFamiliarSpell && o <= FamiliarOriginWildCompanion
}

// String returns "find_familiar_spell", "pact_of_the_chain", "wild_companion" or "unknown".
func (o FamiliarOrigin) String() string {
	return enumName([]string{"find_familiar_spell", "pact_of_the_chain", "wild_companion"}, int(o))
}

// SteedType is the creature type the summoner chooses for an Otherworldly Steed (SRD 5.2.1, "Find Steed", p. 130). The zero
// value is invalid.
type SteedType int

// The three types.
const (
	// SteedCelestial: Radiant Slam, Healing Touch.
	SteedCelestial SteedType = iota + 1
	// SteedFey: Psychic Slam, Fey Step.
	SteedFey
	// SteedFiend: Necrotic Slam, Fell Glare.
	SteedFiend
)

// Valid reports whether t is one of the three types.
func (t SteedType) Valid() bool { return t >= SteedCelestial && t <= SteedFiend }

// String returns "celestial", "fey", "fiend" or "unknown".
func (t SteedType) String() string { return enumName([]string{"celestial", "fey", "fiend"}, int(t)) }

// CreatureType returns the content.CreatureType of the steed.
func (t SteedType) CreatureType() content.CreatureType {
	switch t {
	case SteedCelestial:
		return content.CreatureTypeCelestial
	case SteedFey:
		return content.CreatureTypeFey
	case SteedFiend:
		return content.CreatureTypeFiend
	}
	return 0
}

// SteedMode is how a steed behaves (FR-004). The zero value is invalid.
type SteedMode int

// The two modes.
const (
	// SteedModeControlledWhileRidden: the steed functions as a controlled mount while its summoner rides it.
	SteedModeControlledWhileRidden SteedMode = iota + 1
	// SteedModeIndependent: the summoner has the Incapacitated condition, so the steed acts as it likes.
	SteedModeIndependent
)

// String returns "controlled_while_ridden", "independent" or "unknown".
func (m SteedMode) String() string {
	return enumName([]string{"controlled_while_ridden", "independent"}, int(m))
}

// SteedModeFor returns the mode of a steed whose summoner has the given ability to act: independent while the summoner is
// Incapacitated, otherwise controlled while ridden (FR-004). In both modes the steed's place in the order is the same.
func SteedModeFor(summoner ConditionActionAvailability) SteedMode {
	if summoner == ConditionActionAvailabilityNone {
		return SteedModeIndependent
	}
	return SteedModeControlledWhileRidden
}

// InsectForm is the form the summoner chooses for a Giant Insect (SRD 5.2.1, "Giant Insect", p. 135). The zero value is
// invalid.
type InsectForm int

// The three forms.
const (
	// InsectWasp: has a Fly Speed.
	InsectWasp InsectForm = iota + 1
	// InsectSpider: has the Web Bolt attack.
	InsectSpider
	// InsectCentipede: has the Venomous Spew bonus action.
	InsectCentipede
)

// Valid reports whether f is one of the three forms.
func (f InsectForm) Valid() bool { return f >= InsectWasp && f <= InsectCentipede }

// String returns "wasp", "spider", "centipede" or "unknown".
func (f InsectForm) String() string { return enumName([]string{"wasp", "spider", "centipede"}, int(f)) }

// BonusActionGrantFor returns the BonusActionGrant of the spell of kind k, for the host to pass to TakeBonusAction: the
// Bonus Action that commands Animate Objects, Animate Dead or Create Undead creatures, or shares a familiar's senses (FR-010,
// FR-046). The grant names no standard action. It returns an error matching srderr.ErrInvalidInput for a kind with no Bonus
// Action. It never panics. This package spends no Bonus Action.
func BonusActionGrantFor(k SummonKind) (BonusActionGrant, error) {
	if k == SummonAnimatedObject || k.undead() || k == SummonFamiliar {
		return NewBonusActionGrant(BonusActionFromFeature{Feature: content.FeatureID(k.Spell())}, NoStandardAction{})
	}
	return BonusActionGrant{}, invalidInput("no Bonus Action for this kind of summoned creature")
}
