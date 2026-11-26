package combat

import "github.com/Whealinghold/srd5e-go/content"

// ZoneEvent is the closed family of events a persistent zone produces. Variants:
// ZoneCreated, ZoneTriggered, ZoneMoved and ZoneEnded. Each is an immutable value
// named in the past tense, is an Event, and has a constant snake_case Kind. Each
// variant embeds zoneEventBase, which carries the zone id and implements the
// sealed markers.
type ZoneEvent interface {
	Event
	isZoneEvent()
}

// zoneEventBase carries the zone every zone event is about.
type zoneEventBase struct {
	zone PersistentZoneID
}

func (zoneEventBase) event()       {}
func (zoneEventBase) isZoneEvent() {}

// ZoneID returns the zone the event is about.
func (b zoneEventBase) ZoneID() PersistentZoneID { return b.zone }

// ZoneCreated is a persistent zone made by its caster. Its Kind is zone_created.
type ZoneCreated struct {
	zoneEventBase
	caster CombatantID
}

// Kind returns "zone_created".
func (ZoneCreated) Kind() string { return "zone_created" }

// Combatants returns the caster.
func (e ZoneCreated) Combatants() []CombatantID { return []CombatantID{e.caster} }

// ZoneTriggered is one effect of a zone that is due against a creature. There is
// one per due effect, mandatory or optional. Its Kind is zone_triggered.
type ZoneTriggered struct {
	zoneEventBase
	due ZoneEffectDue
}

// Kind returns "zone_triggered".
func (ZoneTriggered) Kind() string { return "zone_triggered" }

// Combatants returns the creature the effect is due against.
func (e ZoneTriggered) Combatants() []CombatantID { return []CombatantID{e.due.creature} }

// Due returns the effect that is due.
func (e ZoneTriggered) Due() ZoneEffectDue { return e.due }

// ZoneMoved is a zone moved by its caster's action or by its drift. Its Kind is
// zone_moved.
type ZoneMoved struct {
	zoneEventBase
	caster CombatantID
	feet   int
	how    content.ZoneActionKind
}

// Kind returns "zone_moved".
func (ZoneMoved) Kind() string { return "zone_moved" }

// Combatants returns the caster.
func (e ZoneMoved) Combatants() []CombatantID { return []CombatantID{e.caster} }

// DistanceFeet returns the distance the zone moved.
func (e ZoneMoved) DistanceFeet() int { return e.feet }

// How returns the action kind the move was spent as; the zero value marks a
// drift, which spends no action.
func (e ZoneMoved) How() content.ZoneActionKind { return e.how }

// ZoneEnded is a zone that ended, with the reason. Its Kind is zone_ended.
type ZoneEnded struct {
	zoneEventBase
	caster CombatantID
	reason ZoneEndReason
}

// Kind returns "zone_ended".
func (ZoneEnded) Kind() string { return "zone_ended" }

// Combatants returns the caster.
func (e ZoneEnded) Combatants() []CombatantID { return []CombatantID{e.caster} }

// Reason returns why the zone ended.
func (e ZoneEnded) Reason() ZoneEndReason { return e.reason }
