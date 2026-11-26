package combat

import "math"

// EndZoneResult is the closed set of outcomes of [EndZone]. Variants: EndZoneEnded,
// EndZoneAlreadyEnded, EndZoneInvalidReason.
type EndZoneResult interface{ isEndZoneResult() }

// EndZoneEnded is a zone that ended now, with its ZoneEnded event.
type EndZoneEnded struct {
	Zone  PersistentZone
	Event ZoneEnded
}

// EndZoneAlreadyEnded is a refusal: the zone had ended already. It is unchanged.
type EndZoneAlreadyEnded struct{}

// EndZoneInvalidReason is a refusal: the reason is not one of the six ZoneEndReason values (the zero value included).
// The zone is unchanged.
type EndZoneInvalidReason struct{}

func (EndZoneEnded) isEndZoneResult()         {}
func (EndZoneAlreadyEnded) isEndZoneResult()  {}
func (EndZoneInvalidReason) isEndZoneResult() {}

// EndZone ends an active zone for reason (persistent-spell-zones FR-040): the host
// calls it when the creator loses Concentration, the zone's duration runs out, or
// the spell is dispelled. The ledger and facts are kept. A reason that is not one of
// the six ZoneEndReason values (the zero value included) gives EndZoneInvalidReason and no change, before the zone is read. It never panics.
func EndZone(zone PersistentZone, reason ZoneEndReason) EndZoneResult {
	if reason < ZoneEndConcentrationEnded || reason > ZoneEndCasterEnded {
		return EndZoneInvalidReason{}
	}
	if zone.ended {
		return EndZoneAlreadyEnded{}
	}
	zone = endedBy(zone, reason)
	return EndZoneEnded{Zone: zone, Event: zoneEndedEvent(zone)}
}

// endedBy returns the zone ended for reason.
func endedBy(zone PersistentZone, reason ZoneEndReason) PersistentZone {
	zone.ended = true
	zone.endReason = reason
	return zone
}

// zoneEndedEvent returns the ZoneEnded event of an ended zone.
func zoneEndedEvent(zone PersistentZone) ZoneEnded {
	return ZoneEnded{zoneEventBase: zoneEventBase{zone: zone.id}, caster: zone.caster, reason: zone.endReason}
}

// ZoneRoundResult is the outcome of [ZoneRoundStarted]: the zone, and the ZoneEnded
// event when the round ended it. An ended, host-timed or until-dispelled zone is
// returned unchanged with no events.
type ZoneRoundResult struct {
	Zone   PersistentZone
	Events []ZoneEvent
}

// ZoneRoundStarted reports the start of a round to a zone (persistent-spell-zones
// FR-041). A rounds zone ends with ZoneEndDurationElapsed once round reaches its
// cast round plus its rounds (Q-8). It returns an *ZoneInvalidError for a round below
// the cast round. It never panics.
func ZoneRoundStarted(zone PersistentZone, round int) (ZoneRoundResult, error) {
	if round < zone.cast.round {
		return ZoneRoundResult{}, &ZoneInvalidError{What: "round below the zone's cast round"}
	}
	d, ok := zone.duration.(RoundsDuration)
	// round is at least the cast round here, so the difference cannot overflow where the sum would.
	if zone.ended || !ok || round-zone.cast.round < d.rounds {
		return ZoneRoundResult{Zone: zone}, nil
	}
	zone = endedBy(zone, ZoneEndDurationElapsed)
	return ZoneRoundResult{Zone: zone, Events: []ZoneEvent{zoneEndedEvent(zone)}}, nil
}

// RecordDamageResult is the closed set of outcomes of [RecordZoneDamage]. Variants:
// RecordDamageRecorded, RecordDamageNoBudget, RecordDamageZoneEnded.
type RecordDamageResult interface{ isRecordDamageResult() }

// RecordDamageRecorded is damage added to the zone's total. Events holds the
// ZoneEnded event, with ZoneEndBudgetSpent, when the total reached the budget.
type RecordDamageRecorded struct {
	Zone   PersistentZone
	Events []ZoneEvent
}

// RecordDamageNoBudget is a refusal: the zone has no damage budget.
type RecordDamageNoBudget struct{}

// RecordDamageZoneEnded is a refusal: the zone has ended.
type RecordDamageZoneEnded struct{}

func (RecordDamageRecorded) isRecordDamageResult()  {}
func (RecordDamageNoBudget) isRecordDamageResult()  {}
func (RecordDamageZoneEnded) isRecordDamageResult() {}

// RecordZoneDamage adds the damage the host reports to a zone's budget (SRD 5.2.1,
// "Guardian of Faith", p. 137; the host decides what counts, Q-10). Once the total
// reaches the budget the zone ends in the same result. It returns an *ZoneInvalidError
// for a negative amount. It never panics.
func RecordZoneDamage(zone PersistentZone, amount int) (RecordDamageResult, error) {
	if amount < 0 {
		return nil, &ZoneInvalidError{What: "damage amount"}
	}
	if zone.ended {
		return RecordDamageZoneEnded{}, nil
	}
	budget := zone.facts.DamageBudget()
	if budget == 0 {
		return RecordDamageNoBudget{}, nil
	}
	// shortcut: the total saturates at the largest int; a host figure that large is past every budget.
	zone.damage = addSaturating(zone.damage, amount)
	if zone.damage < budget {
		return RecordDamageRecorded{Zone: zone}, nil
	}
	zone = endedBy(zone, ZoneEndBudgetSpent)
	return RecordDamageRecorded{Zone: zone, Events: []ZoneEvent{zoneEndedEvent(zone)}}, nil
}

// addSaturating returns a+b for non-negative a and b, saturating at math.MaxInt.
func addSaturating(a, b int) int {
	if b > math.MaxInt-a {
		return math.MaxInt
	}
	return a + b
}
