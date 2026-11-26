package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
	"github.com/Whealinghold/srd5e-go/srderr"
)

// ZoneInvalidError reports a value that breaks an invariant of the persistent zone
// model: a blank id, a round below 1, a negative amount or distance, a duplicate
// or unsupported designation. It matches srderr.ErrInvalidInput through
// errors.Is. What names the invalid value; it is data for code, not a message.
type ZoneInvalidError struct {
	What string
}

// Error returns a lowercase description of the invalid value.
func (e *ZoneInvalidError) Error() string { return "combat: invalid " + e.What }

// Unwrap returns srderr.ErrInvalidInput.
func (e *ZoneInvalidError) Unwrap() error { return srderr.ErrInvalidInput }

// PersistentZoneID identifies one persistent zone. The zero value is invalid;
// build an id with [NewPersistentZoneID].
type PersistentZoneID string

// NewPersistentZoneID returns the id s.
//
// It returns an *ZoneInvalidError for a blank s (only white space or nothing). It
// never panics.
func NewPersistentZoneID(s string) (PersistentZoneID, error) {
	if strings.TrimSpace(s) == "" {
		return "", &ZoneInvalidError{What: "persistent zone id"}
	}
	return PersistentZoneID(s), nil
}

// String returns the id.
func (id PersistentZoneID) String() string { return string(id) }

// TurnMarker names one turn of the tracker: a round and the combatant whose turn
// it is (SRD 5.2.1, "Combat Step by Step", p. 13). The zero value is invalid.
// It is comparable: two markers are the same turn when they are ==.
type TurnMarker struct {
	round  int
	active CombatantID
}

// NewTurnMarker returns the marker of active's turn in round.
//
// It returns an *ZoneInvalidError for a round below 1. It never panics.
func NewTurnMarker(round int, active CombatantID) (TurnMarker, error) {
	if round < 1 {
		return TurnMarker{}, &ZoneInvalidError{What: "turn marker round"}
	}
	return TurnMarker{round: round, active: active}, nil
}

// Round returns the round of the turn.
func (m TurnMarker) Round() int { return m.round }

// Combatants returns the combatant whose turn the marker names.
func (m TurnMarker) Combatants() []CombatantID { return []CombatantID{m.active} }

// ZoneCreatureFacts are the facts the host reports about one creature for a
// zone: the regions of the zone it is in (an empty set is outside), whether it
// is an enemy of the caster and whether the caster can see it. The zero value
// names no creature and lies outside every region.
type ZoneCreatureFacts struct {
	creature CombatantID
	regions  []content.ZoneRegionID
	enemy    bool
	seen     bool
}

// NewZoneCreatureFacts returns the facts of creature: the regions it is in
// (copied; an empty slice means outside the zone), whether it is an enemy of the
// caster and whether the caster sees it. It never fails and never panics.
func NewZoneCreatureFacts(creature CombatantID, regions []content.ZoneRegionID, enemyOfCaster, seenByCaster bool) ZoneCreatureFacts {
	return ZoneCreatureFacts{creature: creature, regions: slices.Clone(regions), enemy: enemyOfCaster, seen: seenByCaster}
}

// Combatants returns the creature the facts are about.
func (f ZoneCreatureFacts) Combatants() []CombatantID { return []CombatantID{f.creature} }

// Regions returns a copy of the regions the creature is in.
func (f ZoneCreatureFacts) Regions() []content.ZoneRegionID { return slices.Clone(f.regions) }

// IsEnemyOfCaster reports whether the creature is hostile to the caster.
func (f ZoneCreatureFacts) IsEnemyOfCaster() bool { return f.enemy }

// SeenByCaster reports whether the caster can see the creature.
func (f ZoneCreatureFacts) SeenByCaster() bool { return f.seen }

// inRegion reports whether the creature is in region r.
func (f ZoneCreatureFacts) inRegion(r content.ZoneRegionID) bool {
	return slices.Contains(f.regions, r)
}

// ZoneExemptions are the creatures a caster designated at the cast, unaffected
// by the zone for its whole life (SRD 5.2.1, "Spirit Guardians", p. 164). The
// zero value designates nobody. Values are fixed at creation.
type ZoneExemptions struct {
	ids []CombatantID
}

// Contains reports whether c is designated.
func (e ZoneExemptions) Contains(c CombatantID) bool { return slices.Contains(e.ids, c) }

// Creatures returns a copy of the designated creatures in the order given.
func (e ZoneExemptions) Creatures() []CombatantID { return slices.Clone(e.ids) }

// Equal reports whether e and o designate the same creatures in the same order.
func (e ZoneExemptions) Equal(o ZoneExemptions) bool { return slices.Equal(e.ids, o.ids) }

// ZoneEndReason says why a persistent zone ended.
type ZoneEndReason int

// The reasons a zone ends.
const (
	// ZoneEndConcentrationEnded: the creator lost Concentration (SRD 5.2.1,
	// "Concentration", p. 179).
	ZoneEndConcentrationEnded ZoneEndReason = iota + 1
	// ZoneEndDurationElapsed: the rounds of a RoundsDuration zone ran out.
	ZoneEndDurationElapsed
	// ZoneEndDispelled: the host dispelled the zone.
	ZoneEndDispelled
	// ZoneEndDispersed: the zone was dispersed (strong wind, a collapsed web).
	ZoneEndDispersed
	// ZoneEndBudgetSpent: the zone dealt its DamageBudget (SRD 5.2.1, "Guardian
	// of Faith", p. 137).
	ZoneEndBudgetSpent
	// ZoneEndCasterEnded: the caster ended the zone.
	ZoneEndCasterEnded
)

func zoneEndReasonNames() []string {
	return []string{"concentration_ended", "duration_elapsed", "dispelled", "dispersed", "budget_spent", "caster_ended"}
}

// String returns the snake_case name of the reason.
func (r ZoneEndReason) String() string { return zoneName(zoneEndReasonNames(), int(r)) }

// ZoneSkipReason says why a rule the event matched did not make an effect due.
// The reasons apply in the order of the constants (persistent-spell-zones FR-014).
type ZoneSkipReason int

// The skip reasons.
const (
	// SkipDesignatedExempt: the creature was designated at the cast.
	SkipDesignatedExempt ZoneSkipReason = iota + 1
	// SkipOriginCreature: the caster of a moving Emanation is its origin.
	SkipOriginCreature
	// SkipNotInside: the creature is not in the rule's region.
	SkipNotInside
	// SkipNotAnEnemy: the rule acts on enemies of the caster only.
	SkipNotAnEnemy
	// SkipNotSeenByCaster: the rule acts on creatures the caster can see only.
	SkipNotSeenByCaster
	// SkipAlreadyAffectedThisTurn: a once-per-turn zone already applied to the
	// creature this turn.
	SkipAlreadyAffectedThisTurn
	// SkipNotFirstEntryThisTurn: the creature already entered this turn.
	SkipNotFirstEntryThisTurn
	// SkipLessThanOneUnit: the creature travelled less than one unit.
	SkipLessThanOneUnit
	// SkipDueNotForThisZone: an optional due effect that is not of this zone (ApplyOptionalZoneEffect only).
	SkipDueNotForThisZone
)

func zoneSkipReasonNames() []string {
	return []string{"designated_exempt", "origin_creature", "not_inside", "not_an_enemy",
		"not_seen_by_caster", "already_affected_this_turn", "not_first_entry_this_turn", "less_than_one_unit", "due_not_for_this_zone"}
}

// String returns the snake_case name of the reason.
func (r ZoneSkipReason) String() string { return zoneName(zoneSkipReasonNames(), int(r)) }

// ZoneSkip is a rule an event matched and did not make due: the part it names
// and the first reason it did not apply.
type ZoneSkip struct {
	PartIndex int
	Reason    ZoneSkipReason
}

// zoneName returns names[v-1], or "invalid" for a value outside the enum.
func zoneName(names []string, v int) string {
	if v < 1 || v > len(names) {
		return "invalid"
	}
	return names[v-1]
}
