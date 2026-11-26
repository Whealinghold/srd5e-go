package combat

// Event is the base of every combat domain event. Each event is an immutable
// value named in the past tense, with no timestamp and no display text. Kind is
// a constant snake_case name per event type (the value of srd5e.combat.event);
// Combatants returns a fresh copy of the creatures the event is about, in the
// order its defining spec documents. Implemented through the closed sub-families
// of the specs that own events: HitPointEvent so far.
type Event interface {
	event()
	Kind() string
	Combatants() []CombatantID
}

// transportEnvelope marks the ...Occurred events that wrap another spec's event
// for transport: only envelopes implement it, and they report the wrapped
// event's Kind and Combatants. Implemented by TurnEventOccurred, HitPointEventOccurred,
// ConditionEventOccurred and GrappleEventOccurred.
type transportEnvelope interface{ transportEnvelope() }
