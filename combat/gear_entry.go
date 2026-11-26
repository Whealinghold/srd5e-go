package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/ability"
	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/d20"
)

// EntryRecord is one creature setting one piece of gear off on one turn. Construct with a
// keyed literal; NewEntryLedger validates a list of them.
type EntryRecord struct {
	// GearID is the gear that was set off.
	GearID DeployedGearID
	// Creature is the creature that set it off.
	Creature CombatantID
	// Stamp is the turn in progress, whoever's it was.
	Stamp TurnStamp
}

func (r EntryRecord) valid() bool { return r.GearID != "" && !r.Creature.IsZero() && !r.Stamp.IsZero() }

// maxEntryRecords is the cap of plan P10 on the records of an EntryLedger.
const maxEntryRecords = 1024

// EntryLedger remembers which creature set off which gear on which turn, so a rule that
// applies "the first time on a turn" or "only once per turn" applies once (SRD 5.2.1,
// "Caltrops", "Ball Bearings", p. 96; "Oil", p. 98). The host stores it and may drop the
// entries of past turns. It is immutable: Record returns a new ledger. The zero value is the
// empty ledger. Compare with Equal. Safe for concurrent use.
type EntryLedger struct{ records []EntryRecord }

// NewEntryLedger returns the ledger of records, copying the slice; a repeated record counts
// once. It returns an error matching srderr.ErrInvalidInput for a record with a blank gear
// id, a zero creature or the zero stamp, and a LimitExceededError for more than 1024 records (plan P10), before it reads one. It never panics.
func NewEntryLedger(records []EntryRecord) (EntryLedger, error) {
	if err := overCap("entry records", len(records), maxEntryRecords); err != nil {
		return EntryLedger{}, err
	}
	var l EntryLedger
	for _, r := range records {
		if !r.valid() {
			return EntryLedger{}, invalidInput("entry record needs a gear id, a creature and a turn stamp")
		}
		l = l.Record(r)
	}
	return l, nil
}

// Records returns a copy of the records, oldest first.
func (l EntryLedger) Records() []EntryRecord { return slices.Clone(l.records) }

// Len returns the number of records.
func (l EntryLedger) Len() int { return len(l.records) }

// Has reports whether the ledger holds r. The key is the gear, the creature and the stamp, so
// two creatures, or two pieces of gear, on one turn never share a record.
func (l EntryLedger) Has(r EntryRecord) bool { return slices.Contains(l.records, r) }

// Record returns the ledger with r added, the ledger itself when it already holds r. A ledger holds at most 1024
// records (plan P10): when a host that never prunes adds one more, the oldest record is dropped, so a drowned
// turn can be triggered again only after a thousand entries since. The
// caller passes a record from a validated source; NewEntryLedger checks untrusted ones.
func (l EntryLedger) Record(r EntryRecord) EntryLedger {
	if l.Has(r) {
		return l
	}
	kept := l.records
	if len(kept) >= maxEntryRecords {
		kept = kept[len(kept)-maxEntryRecords+1:]
	}
	return EntryLedger{append(slices.Clone(kept), r)}
}

// Equal reports whether l and o hold the same records, whatever the order they were added in.
func (l EntryLedger) Equal(o EntryLedger) bool {
	return len(l.records) == len(o.records) && !slices.ContainsFunc(l.records, func(r EntryRecord) bool { return !o.Has(r) })
}

// GearEffect is the closed set of consequences of a creature setting gear off. The use cases
// return them and apply none. Variants: GearDamageEffect, GearConditionEffect,
// GearSpeedZeroEffect, GearCaughtEffect.
type GearEffect interface{ isGearEffect() }

// GearDamageEffect: the creature takes Damage; the host passes it to ApplyDamage.
type GearDamageEffect struct{ Damage Damage }

// GearConditionEffect: the creature gains the Condition; the host applies it to its
// ConditionSet.
type GearConditionEffect struct{ Condition content.ConditionKind }

// GearSpeedZeroEffect: the creature's Speed is 0 until the start of its next turn. It is
// effect data: no condition models it.
type GearSpeedZeroEffect struct{}

// GearCaughtEffect: a Hunting Trap caught the creature; the host begins the Hold with
// GearHolds.Begin.
type GearCaughtEffect struct{ Hold GearHold }

func (GearDamageEffect) isGearEffect()    {}
func (GearConditionEffect) isGearEffect() {}
func (GearSpeedZeroEffect) isGearEffect() {}
func (GearCaughtEffect) isGearEffect()    {}

// GearEntryOutcome is the closed set of ways a creature's entry into deployed gear can end.
// Variants: GearEntrySaved, GearEntryFailed, GearEntryDamaged.
type GearEntryOutcome interface{ isGearEntryOutcome() }

// GearEntrySaved: the creature passed the saving throw and nothing happens.
type GearEntrySaved struct{}

// GearEntryFailed: the creature failed the saving throw and suffers the effects.
type GearEntryFailed struct{ effects []GearEffect }

// Effects returns a copy of the effects.
func (o GearEntryFailed) Effects() []GearEffect { return slices.Clone(o.effects) }

// GearEntryDamaged: the creature takes damage with no saving throw (burning Oil).
type GearEntryDamaged struct{ effects []GearEffect }

// Effects returns a copy of the effects.
func (o GearEntryDamaged) Effects() []GearEffect { return slices.Clone(o.effects) }

func (GearEntrySaved) isGearEntryOutcome()   {}
func (GearEntryFailed) isGearEntryOutcome()  {}
func (GearEntryDamaged) isGearEntryOutcome() {}

// GearEntryAssessment is the closed set of what AreaEntryRules.Assess finds. Variants:
// GearEntryRollRequired, GearEntryDamageDue, GearEntryRefused.
type GearEntryAssessment interface{ isGearEntryAssessment() }

// GearEntryRollRequired: the creature makes a saving throw. The ledger holds the new record.
type GearEntryRollRequired struct {
	ledger EntryLedger
	dc     d20.DifficultyClass
}

// Ledger returns the ledger with this entry recorded.
func (a GearEntryRollRequired) Ledger() EntryLedger { return a.ledger }

// Ability returns the ability of the saving throw, Dexterity.
func (GearEntryRollRequired) Ability() ability.Ability { return ability.Dexterity }

// DifficultyClass returns the Difficulty Class of the saving throw.
func (a GearEntryRollRequired) DifficultyClass() d20.DifficultyClass { return a.dc }

// GearEntryDamageDue: the creature takes damage with no saving throw. The ledger holds the
// new record.
type GearEntryDamageDue struct {
	ledger  EntryLedger
	effects []GearEffect
}

// Ledger returns the ledger with this entry recorded.
func (a GearEntryDamageDue) Ledger() EntryLedger { return a.ledger }

// Outcome returns the GearEntryDamaged to report.
func (a GearEntryDamageDue) Outcome() GearEntryDamaged {
	return GearEntryDamaged{slices.Clone(a.effects)}
}

// GearEntryRefused: the entry sets nothing off. Nothing is rolled or recorded.
type GearEntryRefused struct{ reason GearRejection }

// Reason returns why: triggerNotApplicable, oilNotBurning, trapAlreadySprung or
// alreadyTriggeredThisTurn.
func (a GearEntryRefused) Reason() GearRejection { return a.reason }

func (GearEntryRollRequired) isGearEntryAssessment() {}
func (GearEntryDamageDue) isGearEntryAssessment()    {}
func (GearEntryRefused) isGearEntryAssessment()      {}

// AreaEntryRules is the rules of a creature entering the area of deployed gear, or ending its
// turn there (SRD 5.2.1, pp. 96-98). The zero value is ready to use. Pure; safe for
// concurrent use.
type AreaEntryRules struct{}

// oilDamage is the damage burning Oil deals: 5 Fire, with no saving throw.
const oilDamage = 5

// Assess decides what a creature setting off gear at stamp must face. For Caltrops and Ball
// Bearings only entering applies; for lit Oil entering and ending a turn both do and share one
// record per turn; for a Hunting Trap only the step onto the plate applies, modelled as
// entering. The refusals, first that applies: triggerNotApplicable, oilNotBurning (unlit, or
// burned through a round before the stamp's), trapAlreadySprung, alreadyTriggeredThisTurn
// (Caltrops, Ball Bearings, Oil). A turn is the one in progress, whoever's (Q-1). Oil that burns
// through (r, A) still counts as burning at (r, B) until the host calls DeployedGears.Expire.
// A Hunting Trap takes no record: it springs once.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a zero
// gear or creature, GearProblemInvalidValue for a trigger that is not valid or the zero stamp,
// or GearProblemEndTurnStampMismatch for an ending of a turn whose stamp is another
// creature's. It never panics.
func (AreaEntryRules) Assess(gear DeployedGear, creature CombatantID, trigger AreaTrigger,
	stamp TurnStamp, ledger EntryLedger) (GearEntryAssessment, error) {
	switch {
	case gear.IsZero() || creature.IsZero():
		return nil, gearInvalid(GearProblemBlankID)
	case !trigger.Valid() || stamp.IsZero():
		return nil, gearInvalid(GearProblemInvalidValue)
	case trigger == AreaTriggerEndsTurn && stamp.combatant != creature:
		return nil, gearInvalid(GearProblemEndTurnStampMismatch)
	}
	refuse := func(r GearRejection) (GearEntryAssessment, error) { return GearEntryRefused{r}, nil }
	if trigger == AreaTriggerEndsTurn && gear.kind != DeployedGearKindPouredOil {
		return refuse(GearRejectionTriggerNotApplicable)
	}
	rec := EntryRecord{gear.id, creature, stamp}
	switch gear.kind {
	case DeployedGearKindHuntingTrap:
		if gear.state == DeployedGearState(GearSprung{}) {
			return refuse(GearRejectionTrapAlreadySprung)
		}
		return GearEntryRollRequired{ledger, DeployedGearProfileFor(gear.kind).saveDC}, nil
	case DeployedGearKindPouredOil:
		if o, burning := gear.state.(OilBurning); !burning || o.burnsThrough.round < stamp.round {
			return refuse(GearRejectionOilNotBurning)
		}
		if ledger.Has(rec) {
			return refuse(GearRejectionAlreadyTriggeredThisTurn)
		}
		return GearEntryDamageDue{ledger.Record(rec), []GearEffect{GearDamageEffect{Damage{oilDamage, content.DamageFire}}}}, nil
	case DeployedGearKindCaltrops, DeployedGearKindBallBearings:
	}
	if ledger.Has(rec) {
		return refuse(GearRejectionAlreadyTriggeredThisTurn)
	}
	return GearEntryRollRequired{ledger.Record(rec), DeployedGearProfileFor(gear.kind).saveDC}, nil
}

// maxTrapDamage is the largest face of the Hunting Trap's 1d4.
const maxTrapDamage = 4

// Resolve returns what follows the saving throw against gear: GearEntrySaved when saved, else
// GearEntryFailed with the effects of the gear. Caltrops give 1 Piercing damage and Speed 0 until
// the start of the creature's next turn; Ball Bearings give the Prone condition; a Hunting Trap
// gives trapDamage Piercing damage (the 1d4 rolled), Speed 0 and a GearCaughtEffect holding a hold
// (id holdID, escape DC 13, not restraining, a chain limit of the profile's default). trapDamage is
// read for a failed Hunting Trap save only.
//
// It returns a *GearError (matching srderr.ErrInvalidInput) with GearProblemBlankID for a zero
// gear, creature or a blank hold id, or GearProblemInvalidValue for Oil (which has no save) or a
// trapDamage outside 1 to 4 on a failed Hunting Trap save. It never panics.
func (AreaEntryRules) Resolve(gear DeployedGear, creature CombatantID, holdID HoldID, saved bool, trapDamage int) (GearEntryOutcome, error) {
	if gear.IsZero() || creature.IsZero() || holdID == "" {
		return nil, gearInvalid(GearProblemBlankID)
	}
	switch {
	case gear.kind == DeployedGearKindPouredOil:
		return nil, gearInvalid(GearProblemInvalidValue)
	case !saved && gear.kind == DeployedGearKindHuntingTrap && (trapDamage < 1 || trapDamage > maxTrapDamage):
		return nil, gearInvalid(GearProblemInvalidValue)
	}
	return gear.entryOutcome(creature, holdID, saved, trapDamage), nil
}

// entryOutcome is Resolve for values known to be valid.
func (g DeployedGear) entryOutcome(creature CombatantID, holdID HoldID, saved bool, trapDamage int) GearEntryOutcome {
	if saved {
		return GearEntrySaved{}
	}
	switch g.kind {
	case DeployedGearKindCaltrops:
		return GearEntryFailed{[]GearEffect{GearDamageEffect{Damage{1, content.DamagePiercing}}, GearSpeedZeroEffect{}}}
	case DeployedGearKindBallBearings:
		return GearEntryFailed{[]GearEffect{GearConditionEffect{content.ConditionProne}}}
	case DeployedGearKindPouredOil, DeployedGearKindHuntingTrap:
	}
	hold := GearHold{holdID, HoldKindHuntingTrap, creature, HoldProfileFor(HoldKindHuntingTrap).escape.dc, HoldRestraintDoesNotRestrain,
		g.id, HoldProfileFor(HoldKindHuntingTrap).movementLimitFeet, ObjectStatus{}}
	return GearEntryFailed{[]GearEffect{GearDamageEffect{Damage{trapDamage, content.DamagePiercing}}, GearSpeedZeroEffect{}, GearCaughtEffect{hold}}}
}
