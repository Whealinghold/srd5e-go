package combat

import (
	"slices"
	"strings"

	"github.com/Whealinghold/srd5e-go/content"
)

// PersistentZone is the state of one lingering area of a spell (persistent-spell-
// zones section 5): its spell and caster, its area, its static facts, its
// designations, its once-per-turn and first-entry ledger, its duration and its
// end. It is immutable: every operation returns the new zone and the events it
// produced. The zero value is invalid; build a zone with [CreateZone]. It is not
// comparable with ==; use [PersistentZone.Equal].
type PersistentZone struct {
	id                    PersistentZoneID
	spell                 content.SpellID
	caster                CombatantID
	shape                 content.AreaShape
	facts                 content.ZoneFacts
	exemptions            ZoneExemptions
	ledger                ZoneLedger
	duration              ZoneDuration
	cast                  TurnMarker
	damage                int
	ended                 bool
	endReason             ZoneEndReason
	endsWithConcentration bool
	creatorIncludesOrigin bool
	stopped               TurnMarker
	hasStopped            bool
}

// ID returns the zone's id.
func (z PersistentZone) ID() PersistentZoneID { return z.id }

// Spell returns the spell the zone came from.
func (z PersistentZone) Spell() content.SpellID { return z.spell }

// Combatants returns the caster of the zone.
func (z PersistentZone) Combatants() []CombatantID { return []CombatantID{z.caster} }

// Shape returns the area the spell placed.
func (z PersistentZone) Shape() content.AreaShape { return z.shape }

// Facts returns the zone's static facts.
func (z PersistentZone) Facts() content.ZoneFacts { return z.facts }

// Exemptions returns the creatures designated at the cast.
func (z PersistentZone) Exemptions() ZoneExemptions { return z.exemptions }

// Ledger returns the once-per-turn and first-entry ledger.
func (z PersistentZone) Ledger() ZoneLedger { return z.ledger }

// Duration returns how long the zone lasts.
func (z PersistentZone) Duration() ZoneDuration { return z.duration }

// CastMarker returns the turn the zone was cast on.
func (z PersistentZone) CastMarker() TurnMarker { return z.cast }

// DamageDealt returns the damage the host has recorded against the zone's budget.
func (z PersistentZone) DamageDealt() int { return z.damage }

// Ended reports whether the zone has ended.
func (z PersistentZone) Ended() bool { return z.ended }

// EndReason returns why the zone ended; it is the zero value while the zone is
// active.
func (z PersistentZone) EndReason() ZoneEndReason { return z.endReason }

// EndsWithConcentration reports whether the host must end the zone when the
// creator loses Concentration (the spell requires Concentration).
func (z PersistentZone) EndsWithConcentration() bool { return z.endsWithConcentration }

// CreatorIncludesOrigin reports whether the caster chose to be in its own
// Emanation at the cast.
func (z PersistentZone) CreatorIncludesOrigin() bool { return z.creatorIncludesOrigin }

// StoppedFor reports whether a creature moved into the zone stopped it on turn.
func (z PersistentZone) StoppedFor(turn TurnMarker) bool { return z.hasStopped && z.stopped == turn }

// Equal reports whether z and o hold the same state, value by value.
func (z PersistentZone) Equal(o PersistentZone) bool {
	return z.id == o.id && z.spell == o.spell && z.caster == o.caster && z.shape == o.shape &&
		z.facts.Equal(o.facts) && z.exemptions.Equal(o.exemptions) && z.ledger.Equal(o.ledger) &&
		z.duration == o.duration && z.cast == o.cast && z.damage == o.damage && z.ended == o.ended &&
		z.endReason == o.endReason && z.endsWithConcentration == o.endsWithConcentration &&
		z.creatorIncludesOrigin == o.creatorIncludesOrigin && z.stopped == o.stopped && z.hasStopped == o.hasStopped
}

// CreateZoneCommand is the input of [CreateZone]: the zone's id, its caster, the
// cast spell, the area the spell placed, the creatures the caster designated, the
// caster's choice to be in its own Emanation, the cast's turn and an optional
// duration override (the Symbol's activation, SRD 5.2.1, "Symbol", p. 167). A nil
// DurationOverride means the spell's own duration applies.
type CreateZoneCommand struct {
	ID                    PersistentZoneID
	Caster                CombatantID
	Spell                 content.Spell
	Shape                 content.AreaShape
	Designated            []CombatantID
	CreatorIncludesOrigin bool
	Cast                  TurnMarker
	DurationOverride      ZoneDuration
}

// CreateZoneRejection is one problem that stops a zone being created.
type CreateZoneRejection int

// The reasons a zone cannot be created.
const (
	// CreateNoZoneFacts: the spell has no zone facts (text-only or not a zone).
	CreateNoZoneFacts CreateZoneRejection = iota + 1
	// CreatePartIndexOutOfRange: a rule names a part the spell's model lacks.
	CreatePartIndexOutOfRange
	// CreateDurationNotZone: the spell's duration is Instantaneous or Special.
	CreateDurationNotZone
	// CreateDesignationNotAllowed: creatures are designated but the spell does
	// not designate at casting.
	CreateDesignationNotAllowed
	// CreateDuplicateExemption: a creature is designated twice.
	CreateDuplicateExemption
	// CreateMissingZoneID: the zone id is the zero value (a blank id).
	CreateMissingZoneID
	// CreateMissingCastTurn: the cast turn is the zero TurnMarker (a round below 1).
	CreateMissingCastTurn
	// CreateLimitExceeded: more than 256 designated creatures or more than 64 rules (plan P10); it is the only reason listed.
	CreateLimitExceeded
)

// CreatePersistentZoneResult is the closed set of outcomes of [CreateZone]. Variants:
// CreateZoneCreated, CreateZoneRejected.
type CreatePersistentZoneResult interface{ isCreatePersistentZoneResult() }

// CreateZoneCreated is a zone made, with its ZoneCreated event.
type CreateZoneCreated struct {
	Zone  PersistentZone
	Event ZoneCreated
}

// CreateZoneRejected is a spell that cannot make a zone. Reasons lists every
// problem in the fixed order of [CreateZoneRejection], or holds only CreateNoZoneFacts.
type CreateZoneRejected struct {
	Reasons []CreateZoneRejection
}

func (CreateZoneCreated) isCreatePersistentZoneResult()  {}
func (CreateZoneRejected) isCreatePersistentZoneResult() {}

// CreateZone makes a persistent zone from a cast spell (persistent-spell-zones
// section 4.1). A spell with text-only effects, or without zone facts, is refused
// with CreateNoZoneFacts. Otherwise every problem is listed in fixed order: a
// rule naming a missing part, a duration that cannot be a zone's (unless the
// command overrides it), designations the spell does not allow, duplicates, a blank zone id and a zero cast TurnMarker. More than 64 rules or more than 256
// designated creatures is refused first, with CreateLimitExceeded alone (plan P10). It never panics.
func CreateZone(cmd CreateZoneCommand) CreatePersistentZoneResult {
	m, ok := cmd.Spell.Effects().(content.ModelledEffects)
	if !ok {
		return CreateZoneRejected{Reasons: []CreateZoneRejection{CreateNoZoneFacts}}
	}
	facts, ok := m.Zone()
	if !ok {
		return CreateZoneRejected{Reasons: []CreateZoneRejection{CreateNoZoneFacts}}
	}
	rules := facts.Rules()
	if len(rules) > maxZoneRules || len(cmd.Designated) > maxInputList {
		return CreateZoneRejected{Reasons: []CreateZoneRejection{CreateLimitExceeded}}
	}
	var reasons []CreateZoneRejection
	parts := len(m.Parts())
	for _, r := range rules {
		if r.PartIndex() < 0 || r.PartIndex() >= parts {
			reasons = append(reasons, CreatePartIndexOutOfRange)
			break
		}
	}
	duration := cmd.DurationOverride
	if duration == nil {
		conv, ok := ZoneDurationFromSpellDuration(cmd.Spell.Duration()).(ZoneDurationConverted)
		if !ok {
			reasons = append(reasons, CreateDurationNotZone)
		} else {
			duration = conv.Duration
		}
	}
	if len(cmd.Designated) > 0 {
		if facts.Designation() != content.DesignationAtCastingOnly {
			reasons = append(reasons, CreateDesignationNotAllowed)
		}
		if hasRepeat(cmd.Designated) {
			reasons = append(reasons, CreateDuplicateExemption)
		}
	}
	if strings.TrimSpace(string(cmd.ID)) == "" {
		reasons = append(reasons, CreateMissingZoneID)
	}
	if cmd.Cast.round < 1 {
		reasons = append(reasons, CreateMissingCastTurn)
	}
	if len(reasons) > 0 {
		return CreateZoneRejected{Reasons: reasons}
	}
	z := PersistentZone{
		id: cmd.ID, spell: cmd.Spell.ID(), caster: cmd.Caster, shape: cmd.Shape, facts: facts,
		exemptions: ZoneExemptions{ids: slices.Clone(cmd.Designated)}, duration: duration, cast: cmd.Cast,
		endsWithConcentration: cmd.Spell.RequiresConcentration(), creatorIncludesOrigin: cmd.CreatorIncludesOrigin,
	}
	return CreateZoneCreated{Zone: z, Event: ZoneCreated{zoneEventBase: zoneEventBase{zone: cmd.ID}, caster: cmd.Caster}}
}

// hasRepeat reports whether a creature appears twice in ids.
func hasRepeat(ids []CombatantID) bool {
	for i, c := range ids {
		if slices.Contains(ids[:i], c) {
			return true
		}
	}
	return false
}
