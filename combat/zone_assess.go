package combat

import (
	"slices"

	"github.com/Whealinghold/srd5e-go/content"
)

// AssessZoneTriggerResult is the closed set of outcomes of [AssessZoneTrigger].
// Variants: AssessZoneTriggerDue, AssessZoneTriggerEnded,
// AssessZoneTriggerStaleTurn, AssessZoneTriggerLimitExceeded. The last three are also outcomes of
// [ApplyOptionalZoneEffect].
type AssessZoneTriggerResult interface{ isAssessZoneTriggerResult() }

// AssessZoneTriggerDue is an event assessed on an active zone. Mandatory effects
// are recorded in Zone's ledger; Optional effects wait for ApplyOptionalZoneEffect.
// Skipped lists each matched rule that was not due with its first reason.
// ZoneStoppedForTurn reports that a creature moved into the zone stopped it. Events
// holds one ZoneTriggered per due effect. No effect due is a result with empty
// Mandatory and Optional.
type AssessZoneTriggerDue struct {
	Zone               PersistentZone
	Mandatory          []ZoneEffectDue
	Optional           []ZoneEffectDue
	Skipped            []ZoneSkip
	ZoneStoppedForTurn bool
	Events             []ZoneEvent
}

// AssessZoneTriggerEnded is an operation on an ended zone, which is unchanged.
type AssessZoneTriggerEnded struct{}

// AssessZoneTriggerStaleTurn is a turn earlier than the zone's cast turn or than a
// turn the ledger already holds for the creature. The zone is unchanged.
type AssessZoneTriggerStaleTurn struct{}

// AssessZoneTriggerLimitExceeded is a call that would leave the ledger holding more than 256 creatures (plan P10; a zone has at most 64 rules, so a call
// makes at most 64 events and the per-call event cap cannot be reached): the shared limit_exceeded value, class invalid. The zone is unchanged.
type AssessZoneTriggerLimitExceeded struct{}

func (AssessZoneTriggerDue) isAssessZoneTriggerResult()           {}
func (AssessZoneTriggerEnded) isAssessZoneTriggerResult()         {}
func (AssessZoneTriggerStaleTurn) isAssessZoneTriggerResult()     {}
func (AssessZoneTriggerLimitExceeded) isAssessZoneTriggerResult() {}

// AssessZoneTrigger matches one reported event against the zone's rules at the
// given turn (persistent-spell-zones section 4.3 and 4.4). Each rule whose trigger
// the event satisfies is either due, optional (CasterChoice), or skipped with the
// first reason of FR-014. Mandatory effects are recorded in the once-per-turn
// ledger of a OncePerTurn zone and the first entry of a creature is recorded.
// Two mandatory rules naming one part give one effect. A nil event is AssessZoneTriggerStaleTurn (class invalid), after the ended check, and the zone is unchanged. It never panics.
func AssessZoneTrigger(zone PersistentZone, turn TurnMarker, event ZoneTriggerEvent) AssessZoneTriggerResult {
	if zone.ended {
		return AssessZoneTriggerEnded{}
	}
	if event == nil {
		return AssessZoneTriggerStaleTurn{} // nothing reported, nothing to assess: the invalid class, the zone unchanged
	}
	f := event.Facts()
	if staleTurn(zone, turn, f.creature) {
		return AssessZoneTriggerStaleTurn{}
	}
	var out AssessZoneTriggerDue
	ledger := zone.ledger
	for _, rule := range zone.facts.Rules() {
		if !eventSatisfies(event, rule.Trigger()) {
			continue
		}
		units, reason, skipped := assessRule(zone, ledger, turn, rule, event)
		if skipped {
			if reason == SkipAlreadyAffectedThisTurn && hasPart(out.Mandatory, rule.PartIndex()) {
				continue // the part is due once already (FR-017): not a second skip
			}
			out.Skipped = append(out.Skipped, ZoneSkip{PartIndex: rule.PartIndex(), Reason: reason})
			continue
		}
		due := newZoneEffectDue(zone, turn, rule, f.creature, units)
		if rule.CasterChoice() {
			if hasPart(out.Optional, rule.PartIndex()) || hasPart(out.Mandatory, rule.PartIndex()) {
				continue // the part is due once already (FR-017)
			}
			out.Optional = append(out.Optional, due)
			continue
		}
		if hasPart(out.Mandatory, rule.PartIndex()) {
			continue
		}
		out.Mandatory = append(out.Mandatory, due)
		if zone.facts.OncePerTurn() {
			ledger = ledger.withApplied(f.creature, turn)
		}
	}
	// Only a FirstEntryOnTurn rule reads the entry ledger, so a zone without one never records it.
	if _, ok := event.(CreatureEntered); ok && slices.ContainsFunc(zone.facts.Rules(), func(r content.ZoneRuleFact) bool { return r.Trigger() == content.CreatureFirstEntryOnTurn }) {
		ledger = ledger.withEntered(f.creature, turn)
	}
	if ledger.Size() > maxZoneLedgerCreatures {
		return AssessZoneTriggerLimitExceeded{}
	}
	zone.ledger = ledger
	if _, ok := event.(ZoneMovedIntoSpace); ok && zone.facts.StopsMoverOnContact() {
		zone.stopped, zone.hasStopped = turn, true
		out.ZoneStoppedForTurn = true
	}
	out.Zone = zone
	for _, d := range out.Mandatory {
		out.Events = append(out.Events, ZoneTriggered{zoneEventBase: zoneEventBase{zone: zone.id}, due: d})
	}
	for _, d := range out.Optional {
		out.Events = append(out.Events, ZoneTriggered{zoneEventBase: zoneEventBase{zone: zone.id}, due: d})
	}
	return out
}

// eventSatisfies reports whether event satisfies a rule with trigger trig (table E).
func eventSatisfies(event ZoneTriggerEvent, trig content.ZoneTrigger) bool {
	switch event.(type) {
	case CreatureEntered:
		return trig == content.CreatureEnters || trig == content.CreatureFirstEntryOnTurn
	case CreatureEndedTurn:
		return trig == content.CreatureEndsTurnInside
	case CreatureStartedTurn:
		return trig == content.CreatureStartsTurnInside
	case ZoneMovedIntoSpace:
		return trig == content.ZoneMovesIntoSpace
	case CreatureTravelled:
		return trig == content.TravelsWithin
	}
	return false
}

// assessRule returns the units of a matched rule, or the first reason it is
// skipped (persistent-spell-zones FR-014 order). ledger is the ledger as the
// earlier rules of the same event left it.
func assessRule(zone PersistentZone, ledger ZoneLedger, turn TurnMarker, rule content.ZoneRuleFact, event ZoneTriggerEvent) (int, ZoneSkipReason, bool) {
	f := event.Facts()
	c := f.creature
	if zone.exemptions.Contains(c) {
		return 0, SkipDesignatedExempt, true
	}
	// A moving Emanation's caster is its origin: excluded unless it chose otherwise (FR-008).
	if _, ok := zone.facts.Mobility().(content.MovesWithOrigin); ok && c == zone.caster && !zone.creatorIncludesOrigin {
		return 0, SkipOriginCreature, true
	}
	if !f.inRegion(rule.Region()) {
		return 0, SkipNotInside, true
	}
	if rule.Scope() == content.EnemiesOfCaster && !f.IsEnemyOfCaster() {
		return 0, SkipNotAnEnemy, true
	}
	if rule.Scope() == content.CreaturesCasterCanSee && !f.SeenByCaster() {
		return 0, SkipNotSeenByCaster, true
	}
	if m, ok := ledger.Applied(c); ok && zone.facts.OncePerTurn() && m == turn {
		return 0, SkipAlreadyAffectedThisTurn, true
	}
	if m, ok := ledger.Entered(c); ok && rule.Trigger() == content.CreatureFirstEntryOnTurn && m == turn {
		return 0, SkipNotFirstEntryThisTurn, true
	}
	if t, ok := event.(CreatureTravelled); ok {
		units := t.FeetInside() / rule.UnitFeet()
		if units < 1 {
			return 0, SkipLessThanOneUnit, true
		}
		return units, 0, false
	}
	return 1, 0, false
}

// newZoneEffectDue returns the due effect of rule against creature at turn.
func newZoneEffectDue(zone PersistentZone, turn TurnMarker, rule content.ZoneRuleFact, creature CombatantID, units int) ZoneEffectDue {
	return ZoneEffectDue{
		zone: zone.id, spell: zone.spell, cast: zone.cast, turn: turn, caster: zone.caster,
		creature: creature, part: rule.PartIndex(), trigger: rule.Trigger(), units: units,
		casterChoice: rule.CasterChoice(),
	}
}

// hasPart reports whether a due effect already names part.
func hasPart(dues []ZoneEffectDue, part int) bool {
	for _, d := range dues {
		if d.part == part {
			return true
		}
	}
	return false
}

// ApplyOptionalZoneEffectResult is the closed set of outcomes of
// [ApplyOptionalZoneEffect]. Variants: OptionalZoneEffectApplied,
// OptionalZoneEffectRefused, AssessZoneTriggerEnded, AssessZoneTriggerStaleTurn, AssessZoneTriggerLimitExceeded.
type ApplyOptionalZoneEffectResult interface{ isApplyOptionalZoneEffectResult() }

// OptionalZoneEffectApplied is an optional effect the caster chose to apply, now
// recorded in the zone's ledger.
type OptionalZoneEffectApplied struct {
	Zone PersistentZone
}

// OptionalZoneEffectRefused is an optional effect that cannot be applied now,
// with the reason (AlreadyAffectedThisTurn).
type OptionalZoneEffectRefused struct {
	Reason ZoneSkipReason
}

func (OptionalZoneEffectApplied) isApplyOptionalZoneEffectResult()      {}
func (OptionalZoneEffectRefused) isApplyOptionalZoneEffectResult()      {}
func (AssessZoneTriggerEnded) isApplyOptionalZoneEffectResult()         {}
func (AssessZoneTriggerStaleTurn) isApplyOptionalZoneEffectResult()     {}
func (AssessZoneTriggerLimitExceeded) isApplyOptionalZoneEffectResult() {}

// ApplyOptionalZoneEffect records an optional effect the caster chose to apply at
// turn (persistent-spell-zones FR-023). Declining is doing nothing, so the ledger
// is untouched until this call. A once-per-turn zone refuses a creature it already
// affected this turn. A due effect of another zone, of another cast or of a part the zone has no
// optional rule for is refused with SkipDueNotForThisZone, after the ended and stale turn checks. It emits no event: AssessZoneTrigger emitted it. It never panics.
func ApplyOptionalZoneEffect(zone PersistentZone, turn TurnMarker, due ZoneEffectDue) ApplyOptionalZoneEffectResult {
	if zone.ended {
		return AssessZoneTriggerEnded{}
	}
	if staleTurn(zone, turn, due.creature) {
		return AssessZoneTriggerStaleTurn{}
	}
	if due.zone != zone.id || due.spell != zone.spell || due.cast != zone.cast || due.caster != zone.caster || !zone.namesOptionalPart(due.part) {
		return OptionalZoneEffectRefused{Reason: SkipDueNotForThisZone}
	}
	if !zone.facts.OncePerTurn() {
		return OptionalZoneEffectApplied{Zone: zone}
	}
	if m, ok := zone.ledger.Applied(due.creature); ok && m == turn {
		return OptionalZoneEffectRefused{Reason: SkipAlreadyAffectedThisTurn}
	}
	ledger := zone.ledger.withApplied(due.creature, turn)
	if ledger.Size() > maxZoneLedgerCreatures {
		return AssessZoneTriggerLimitExceeded{}
	}
	zone.ledger = ledger
	return OptionalZoneEffectApplied{Zone: zone}
}

// namesOptionalPart reports whether a rule of the zone with a CasterChoice names part.
func (z PersistentZone) namesOptionalPart(part int) bool {
	return slices.ContainsFunc(z.facts.Rules(), func(r content.ZoneRuleFact) bool { return r.PartIndex() == part && r.CasterChoice() })
}

// RecordCastApplicationResult is the closed set of outcomes of
// [RecordZoneCastApplication]. Variants: RecordCastApplicationRecorded,
// RecordCastApplicationZoneEnded, RecordCastApplicationNotOncePerTurn,
// RecordCastApplicationStaleTurn, RecordCastApplicationLimitExceeded.
type RecordCastApplicationResult interface{ isRecordCastApplicationResult() }

// RecordCastApplicationRecorded is the creature recorded as affected at the turn.
type RecordCastApplicationRecorded struct {
	Zone PersistentZone
}

// RecordCastApplicationZoneEnded is a refusal: the zone has ended.
type RecordCastApplicationZoneEnded struct{}

// RecordCastApplicationNotOncePerTurn is a refusal: the zone has no once-per-turn
// ledger (the appearance counts only for a once-per-turn zone).
type RecordCastApplicationNotOncePerTurn struct{}

// RecordCastApplicationStaleTurn is a turn earlier than the cast or the ledger.
type RecordCastApplicationStaleTurn struct{}

// RecordCastApplicationLimitExceeded is a call that would leave the ledger holding more than 256 creatures (plan P10):
// the shared limit_exceeded value, class invalid. The zone is unchanged.
type RecordCastApplicationLimitExceeded struct{}

func (RecordCastApplicationRecorded) isRecordCastApplicationResult()       {}
func (RecordCastApplicationZoneEnded) isRecordCastApplicationResult()      {}
func (RecordCastApplicationNotOncePerTurn) isRecordCastApplicationResult() {}
func (RecordCastApplicationStaleTurn) isRecordCastApplicationResult()      {}
func (RecordCastApplicationLimitExceeded) isRecordCastApplicationResult()  {}

// RecordZoneCastApplication records that a zone's appearance applied to creature
// at turn, so the turn's once-per-turn application is used up (persistent-spell-
// zones FR-030, Q-6). It never panics.
func RecordZoneCastApplication(zone PersistentZone, creature CombatantID, turn TurnMarker) RecordCastApplicationResult {
	if zone.ended {
		return RecordCastApplicationZoneEnded{}
	}
	if staleTurn(zone, turn, creature) {
		return RecordCastApplicationStaleTurn{}
	}
	if !zone.facts.OncePerTurn() {
		return RecordCastApplicationNotOncePerTurn{}
	}
	ledger := zone.ledger.withApplied(creature, turn)
	if ledger.Size() > maxZoneLedgerCreatures {
		return RecordCastApplicationLimitExceeded{}
	}
	zone.ledger = ledger
	return RecordCastApplicationRecorded{Zone: zone}
}
